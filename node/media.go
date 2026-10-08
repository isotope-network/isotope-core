// node/media.go
package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
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
