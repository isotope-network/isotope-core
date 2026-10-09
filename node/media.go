// node/media.go
package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// ============================================================
// МЕДИА (голосовые, файлы) — отдельный путь от текста.
// Text = шифротекст (base64 nonce||ciphertext).
// PlainText = открытый base64 (для себя).
// MediaType — маркер ("voice" | "file").
// Duration — секунды (для голосовых).
// MediaData — не используем (поле есть, пустое).
// ============================================================

// sendMedia — общий путь для медиа.
// Шифрует mediaData, создаёт Message с MediaType/Duration,
// сохраняет в memory, реплицирует, возвращает ID.
func (n *Node) sendMedia(peerID, mediaType, mediaData string, duration int, ttlPeriod, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if peerID == "" {
		return "", fmt.Errorf("peerID is required")
	}
	if mediaData == "" {
		return "", fmt.Errorf("mediaData is required")
	}
	if mediaType == "" {
		return "", fmt.Errorf("mediaType is required")
	}

	encrypted, err := n.encryptForRecipient(peerID, mediaData)
	if err != nil {
		return "", fmt.Errorf("encrypt failed: %w", err)
	}

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	id := generateMsgID(fmt.Sprintf("%s:%s:%d", mediaType, peerID, time.Now().UnixNano()))
	readEnabled := n.myReadEnabled

	msg := Message{
		ID:               id,
		Text:             encrypted,
		PlainText:        mediaData,
		Sender:           n.host.ID().String(),
		Recipient:        peerID,
		Version:          MESSAGE_VERSION_E2E,
		Type:             TypeMessage,
		MediaType:        mediaType,
		Duration:         duration,
		TtlPeriodSeconds: ttlSeconds,
		TtlMode:          ttlMode,
		ReadEnabled:      &readEnabled,
		Time:             time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:            true,
		Weight:           0.5,
		Priority:         0,
		Mode:             0,
		Score:            0,
		ExpiresAt:        expiresAt,
	}

	if n.memory.Add(msg) {
		n.setMessageStatus(id, StatusSent)
		n.replicateMessage(msg)
		if n.messageHook != nil {
			data, _ := json.Marshal(msg)
			n.messageHook(string(data))
		}
		log.Printf("[MEDIA] sent %s to %s (id=%s, %d bytes, %ds)",
			mediaType, peerID, id, len(mediaData), duration)
	} else {
		log.Printf("[MEDIA] memory.Add rejected %s (id=%s)", mediaType, id)
	}

	go n.saveState()
	return id, nil
}

// SendVoice — отправляет голосовое сообщение.
// mediaData — base64 Opus/Ogg. duration — секунды.
// ttlPeriod: "10s" | "30s" | ... | "never".
// ttlMode: "" (при never) | "after_read" | "hard".
func (n *Node) SendVoice(peerID string, mediaData string, duration int, ttlPeriod, ttlMode string) (string, error) {
	return n.sendMedia(peerID, "voice", mediaData, duration, ttlPeriod, ttlMode)
}

// SendPhoto — отправляет фото. Один чанк (сжатое фото < 1 МБ).
// photoBase64 — base64 JPEG. duration = 0.
func (n *Node) SendPhoto(peerID string, photoBase64 string, ttlPeriod, ttlMode string) (string, error) {
	return n.sendMedia(peerID, "photo", photoBase64, 0, ttlPeriod, ttlMode)
}

// SendPhotoByPath — отправляет фото, читая файл с диска по пути.
// Путь — внутри app dir (Dart передаёт путь из image_picker).
// Возвращает MediaID.
func (n *Node) SendPhotoByPath(peerID, filePath, fileName string, ttlPeriod, ttlMode string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("filePath is required")
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file failed: %w", err)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("empty file")
	}
	photoB64 := base64.StdEncoding.EncodeToString(raw)
	return n.sendMedia(peerID, "photo", photoB64, 0, ttlPeriod, ttlMode)
}

// SendFileByPath — отправляет файл, читая его с диска по пути.
// Схема:
//  1. Копирует исходный файл в <stateFile dir>/isotope_media/sent/<MediaID>.bin.
//     Копия нужна для flushPending — переслать чанк, если получатель не подтвердил.
//  2. Читает чанки через f.ReadAt (не os.ReadFile) — в память только один чанк.
//  3. Шифрует, отправляет, в memory кладёт только метаданные.
//
// Base64 через MethodChannel не проходит — OOM устранён.
// Возвращает MediaID.
func (n *Node) SendFileByPath(peerID, filePath, fileName string, ttlPeriod, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if peerID == "" {
		return "", fmt.Errorf("peerID is required")
	}
	if filePath == "" {
		return "", fmt.Errorf("filePath is required")
	}

	src, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file failed: %w", err)
	}
	defer src.Close()

	stat, err := src.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file failed: %w", err)
	}
	fileSize := stat.Size()
	if fileSize == 0 {
		return "", fmt.Errorf("empty file")
	}

	const chunkSize = 64 * 1024
	chunkTotal := (fileSize + chunkSize - 1) / chunkSize

	mediaID := generateMsgID(fmt.Sprintf("file:%s:%s:%d", peerID, fileName, time.Now().UnixNano()))

	// Копия в sent/ — для flushPending.
	sentPath := n.sentPath(mediaID)
	if err := os.MkdirAll(n.sentDirPath(), 0700); err != nil {
		return "", fmt.Errorf("mkdir sent failed: %w", err)
	}
	dst, err := os.Create(sentPath)
	if err != nil {
		return "", fmt.Errorf("create sent copy failed: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(sentPath)
		return "", fmt.Errorf("copy to sent failed: %w", err)
	}
	dst.Close()

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	readEnabled := n.myReadEnabled
	myID := n.host.ID().String()

	const maxParallel = 5
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	for i := 0; i < int(chunkTotal); i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			buf := make([]byte, chunkSize)
			off := int64(idx) * int64(chunkSize)
			nr, err := src.ReadAt(buf, off)
			if err != nil && err != io.EOF {
				log.Printf("[FILE] ReadAt chunk %d failed: %v", idx, err)
				return
			}
			if nr == 0 {
				return
			}
			chunkRaw := buf[:nr]
			chunkB64 := base64.StdEncoding.EncodeToString(chunkRaw)

			encrypted, err := n.encryptForRecipient(peerID, chunkB64)
			if err != nil {
				log.Printf("[FILE] encrypt chunk %d failed: %v", idx, err)
				return
			}

			id := generateMsgID(fmt.Sprintf("file:%s:%d:%d", mediaID, idx, time.Now().UnixNano()))
			msg := Message{
				ID:               id,
				Text:             encrypted,
				PlainText:        chunkB64,
				Sender:           myID,
				Recipient:        peerID,
				Version:          MESSAGE_VERSION_E2E,
				Type:             TypeMessage,
				MediaType:        "file",
				MediaID:          mediaID,
				ChunkIndex:       idx,
				ChunkTotal:       int(chunkTotal),
				FileName:         fileName,
				FileSize:         fileSize,
				TtlPeriodSeconds: ttlSeconds,
				TtlMode:          ttlMode,
				ReadEnabled:      &readEnabled,
				Time:             time.Now().UTC().Format("2006-01-02T15:04:05"),
				IsOwn:            true,
				Weight:           0.5,
				Priority:         0,
				Mode:             0,
				Score:            0,
				ExpiresAt:        expiresAt,
			}
			// Отправка: replicateMessage идёт с base64 (нужен получателю).
			n.replicateMessage(msg)
			n.setMessageStatus(id, StatusSent)

			// В memory — только метаданные (без base64).
			memMsg := msg
			memMsg.Text = ""
			memMsg.PlainText = ""
			if n.memory.Add(memMsg) {
				if n.messageHook != nil {
					data, _ := json.Marshal(memMsg)
					n.messageHook(string(data))
				}
			} else {
				log.Printf("[FILE] memory.Add rejected chunk %d/%d of %s", idx, chunkTotal, mediaID)
			}
		}(i)
	}
	wg.Wait()

	log.Printf("[FILE] sent %s (%s, %d bytes, %d chunks) to %s",
		mediaID, fileName, fileSize, chunkTotal, peerID)
	go n.saveState()
	return mediaID, nil
}

// SendFile — отправляет файл. Режет на чанки по 64 КБ (сырых байтов),
// шифрует каждый чанк отдельно, отправляет как отдельный Message
// с MediaType="file", MediaID, ChunkIndex, ChunkTotal.
// fileBase64 — base64 исходного файла. fileSize — размер в байтах.
// Возвращает MediaID (не msg_id одного чанка).
func (n *Node) SendFile(peerID, fileBase64, fileName string, fileSize int64, ttlPeriod, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if peerID == "" {
		return "", fmt.Errorf("peerID is required")
	}
	if fileBase64 == "" {
		return "", fmt.Errorf("fileBase64 is required")
	}

	raw, err := base64.StdEncoding.DecodeString(fileBase64)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("empty file")
	}

	const chunkSize = 64 * 1024 // 64 КБ
	chunkTotal := (len(raw) + chunkSize - 1) / chunkSize

	mediaID := generateMsgID(fmt.Sprintf("file:%s:%s:%d", peerID, fileName, time.Now().UnixNano()))

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	readEnabled := n.myReadEnabled
	myID := n.host.ID().String()

	// Параллельная отправка — 5 потоков. Каждый чанк шифруется
	// и отправляется независимо. Порядок не важен: получатель
	// собирает по ChunkIndex.
	const maxParallel = 5
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	for i := 0; i < chunkTotal; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			start := idx * chunkSize
			end := start + chunkSize
			if end > len(raw) {
				end = len(raw)
			}
			chunkRaw := raw[start:end]
			chunkB64 := base64.StdEncoding.EncodeToString(chunkRaw)

			encrypted, err := n.encryptForRecipient(peerID, chunkB64)
			if err != nil {
				log.Printf("[FILE] encrypt chunk %d failed: %v", idx, err)
				return
			}

			id := generateMsgID(fmt.Sprintf("file:%s:%d:%d", mediaID, idx, time.Now().UnixNano()))
			msg := Message{
				ID:               id,
				Text:             encrypted,
				PlainText:        chunkB64,
				Sender:           myID,
				Recipient:        peerID,
				Version:          MESSAGE_VERSION_E2E,
				Type:             TypeMessage,
				MediaType:        "file",
				MediaID:          mediaID,
				ChunkIndex:       idx,
				ChunkTotal:       chunkTotal,
				FileName:         fileName,
				FileSize:         fileSize,
				TtlPeriodSeconds: ttlSeconds,
				TtlMode:          ttlMode,
				ReadEnabled:      &readEnabled,
				Time:             time.Now().UTC().Format("2006-01-02T15:04:05"),
				IsOwn:            true,
				Weight:           0.5,
				Priority:         0,
				Mode:             0,
				Score:            0,
				ExpiresAt:        expiresAt,
			}
			if n.memory.Add(msg) {
				n.setMessageStatus(id, StatusSent)
				n.replicateMessage(msg)
				if n.messageHook != nil {
					data, _ := json.Marshal(msg)
					n.messageHook(string(data))
				}
			} else {
				log.Printf("[FILE] memory.Add rejected chunk %d/%d of %s", idx, chunkTotal, mediaID)
			}
		}(i)
	}
	wg.Wait()

	log.Printf("[FILE] sent %s (%s, %d bytes, %d chunks) to %s",
		mediaID, fileName, fileSize, chunkTotal, peerID)
	go n.saveState()
	return mediaID, nil
}

// node/media.go
