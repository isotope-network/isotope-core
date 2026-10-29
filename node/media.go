// node/media.go
package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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

// SendVoiceByPath — отправляет голосовое, читая файл с диска по пути (v2.0.5).
// filePath — внутри app dir. Go режет на чанки по 47 КБ (Sphinx payload 64 КБ).
// duration — секунды. Base64 через MethodChannel не проходит — OOM устранён.
// Возвращает MediaID.
func (n *Node) SendVoiceByPath(peerID, filePath, fileName string, duration int, ttlPeriod, ttlMode string) (string, error) {
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

	mediaID := generateMsgID(fmt.Sprintf("voice:%s:%s:%d", peerID, fileName, time.Now().UnixNano()))

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

	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".opus"
	}
	if err := os.MkdirAll(n.filesDirPath(), 0700); err == nil {
		if _, err := src.Seek(0, io.SeekStart); err == nil {
			permPath := filepath.Join(n.filesDirPath(), mediaID+ext)
			if permDst, err := os.Create(permPath); err == nil {
				io.Copy(permDst, src)
				permDst.Close()
			}
		}
	}

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	go n.sendFileChunksAsync(peerID, mediaID, fileName, fileSize, "voice", ttlSeconds, ttlMode, expiresAt)

	return mediaID, nil
}

// SendPhotoByPath — отправляет фото, читая файл с диска по пути.
// filePath — внутри app dir (Dart передаёт путь из image_picker).
// v2.0.5: нарезка на чанки 47 КБ (Sphinx payload 64 КБ). Base64 через
// MethodChannel не проходит — OOM устранён.
// Возвращает MediaID.
func (n *Node) SendPhotoByPath(peerID, filePath, fileName string, ttlPeriod, ttlMode string) (string, error) {
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

	mediaID := generateMsgID(fmt.Sprintf("photo:%s:%s:%d", peerID, fileName, time.Now().UnixNano()))

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

	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".jpg"
	}
	if err := os.MkdirAll(n.filesDirPath(), 0700); err == nil {
		if _, err := src.Seek(0, io.SeekStart); err == nil {
			permPath := filepath.Join(n.filesDirPath(), mediaID+ext)
			if permDst, err := os.Create(permPath); err == nil {
				io.Copy(permDst, src)
				permDst.Close()
			}
		}
	}

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	go n.sendFileChunksAsync(peerID, mediaID, fileName, fileSize, "photo", ttlSeconds, ttlMode, expiresAt)

	return mediaID, nil
}

// SendFileByPath — отправляет файл, читая его с диска по пути.
// Схема:
//  1. Копирует исходный файл в <stateFile dir>/isotope_media/sent/<MediaID>.bin.
//     Копия нужна для retry (unackedChunks) и flushPending — переслать чанк,
//     если получатель не подтвердил.
//  2. Запускает фоновую горутину sendFileChunksAsync, которая читает
//     чанки из sent/, шифрует, кладёт в chunkQueue (блокирующая, backpressure).
//  3. Возвращает MediaID немедленно — MethodChannel не блокируется.
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

	mediaID := generateMsgID(fmt.Sprintf("file:%s:%s:%d", peerID, fileName, time.Now().UnixNano()))

	// Копия в sent/ — источник для sendFileChunksAsync и retry.
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

	// Постоянная копия в files/<mediaID>.<ext> — для UI (превью, «Открыть»,
	// «Сохранить»). Не зависит от sent/ (временный, для retry).
	// Как у получателя после сборки.
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".bin"
	}
	if err := os.MkdirAll(n.filesDirPath(), 0700); err != nil {
		log.Printf("[FILE] mkdir files failed for %s: %v", mediaID, err)
	} else if _, err := src.Seek(0, io.SeekStart); err != nil {
		log.Printf("[FILE] seek src failed for %s: %v", mediaID, err)
	} else {
		permPath := filepath.Join(n.filesDirPath(), mediaID+ext)
		if permDst, err := os.Create(permPath); err == nil {
			if _, err := io.Copy(permDst, src); err != nil {
				log.Printf("[FILE] copy to files failed for %s: %v", mediaID, err)
			}
			permDst.Close()
		} else {
			log.Printf("[FILE] create files copy failed for %s: %v", mediaID, err)
		}
	}

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	// Фоновая отправка. MethodChannel возвращается немедленно.
	go n.sendFileChunksAsync(peerID, mediaID, fileName, fileSize, "file", ttlSeconds, ttlMode, expiresAt)

	return mediaID, nil
}

// sendFileChunksAsync — читает чанки из sent/<MediaID>.bin,
// шифрует, кладёт в chunkQueue. Блокирующая постановка (backpressure):
// если очередь полна — ждём воркеров, не дропаем.
// В memory — только метаданные (без base64). base64 хранится в sent/ на диске.
func (n *Node) sendFileChunksAsync(peerID, mediaID, fileName string, fileSize int64, mediaType string, ttlSeconds int, ttlMode string, expiresAt time.Time) {
	// Вариант A (Публикатор 2026-10-29): последовательная постановка файлов
	// в chunkQueue. Файл 1 ставит все чанки → файл 2 начинает ставить.
	n.mediaSendMu.Lock()
	for n.activeMediaID != "" && n.activeMediaID != mediaID {
		n.mediaSendCond.Wait()
	}
	n.activeMediaID = mediaID
	n.mediaSendMu.Unlock()
	defer func() {
		n.mediaSendMu.Lock()
		if n.activeMediaID == mediaID {
			n.activeMediaID = ""
			n.mediaSendCond.Broadcast()
		}
		n.mediaSendMu.Unlock()
	}()

	chunkTotal := (fileSize + ChunkSize - 1) / ChunkSize

	// Начальный прогресс: Dart сразу покажет "Отправка 0%", не "Подготовка…".
	n.pushFileStatus(mediaID, 0, int(chunkTotal))

	src, err := os.Open(n.sentPath(mediaID))

	if err != nil {
		log.Printf("[FILE] open sent copy failed for %s: %v", mediaID, err)
		return
	}
	defer src.Close()

	readEnabled := n.myReadEnabled
	myID := n.host.ID().String()

	for i := 0; i < int(chunkTotal); i++ {
		buf := make([]byte, ChunkSize)
		off := int64(i) * int64(ChunkSize)
		nr, err := src.ReadAt(buf, off)
		if err != nil && err != io.EOF {
			log.Printf("[FILE] ReadAt chunk %d/%d of %s failed: %v", i, chunkTotal, mediaID, err)
			continue
		}
		if nr == 0 {
			continue
		}
		chunkRaw := buf[:nr]

		encrypted, err := n.encryptBytesForRecipient(peerID, chunkRaw)
		if err != nil {
			log.Printf("[FILE] encrypt chunk %d/%d of %s failed: %v", i, chunkTotal, mediaID, err)
			continue
		}

		id := generateMsgID(fmt.Sprintf("file:%s:%d:%d", mediaID, i, time.Now().UnixNano()))
		// Mixing (v2.0.3): файловые чанки — если Sphinx + Mixing включены.
		mixingEnabled := false
		if n.settingsStore != nil && n.settingsStore.GetMixingEnabled() && n.useOnion {
			mixingEnabled = true
		}
		msg := Message{
			ID:               id,
			Text:             encrypted,
			PlainText:        "",
			Sender:           myID,
			Recipient:        peerID,
			Version:          MESSAGE_VERSION_E2E,
			Type:             TypeMessage,
			MediaType:        mediaType,
			MediaID:          mediaID,
			ChunkIndex:       i,
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
			MixingEnabled:    mixingEnabled,
		}

		// Блокирующая постановка в chunkQueue. Backpressure: ждём воркеров.
		// Это гарантирует, что чанк не будет потерян из-за переполнения.
		n.chunkQueue <- msg

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
			log.Printf("[FILE] memory.Add rejected chunk %d/%d of %s", i, chunkTotal, mediaID)
		}
	}

	log.Printf("[FILE] sent %s (%s, %d bytes, %d chunks) to %s",
		mediaID, fileName, fileSize, chunkTotal, peerID)
	go n.saveState()
}

// resumeIncompleteFiles — восстановление незавершённых файловых отправок
// после перезапуска приложения.
// Обходит sent/<mediaID>.bin: для каждого файла восстанавливает
// deliveredCount (из Status чанков в memory) и unackedChunks
// (для чанков со Status < StatusDelivered).
// Идемпотентно: retryUnacked досылает те же msgID, получатель подтверждает.
func (n *Node) resumeIncompleteFiles() {
	dir := n.sentDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".bin") {
			continue
		}
		mediaID := strings.TrimSuffix(e.Name(), ".bin")
		n.resumeFile(mediaID)
	}
}

// resumeFile — восстановление одного файла по mediaID.
// deliveredCount — из Status чанков в memory (источник истины).
// unackedChunks — для чанков со Status < StatusDelivered.
func (n *Node) resumeFile(mediaID string) {
	if mediaID == "" {
		return
	}
	all := n.memory.GetAll()
	var chunks []Message
	for _, m := range all {
		if m.MediaID == mediaID && isChunkedMediaType(m.MediaType) {
			chunks = append(chunks, m)
		}
	}
	if len(chunks) == 0 {
		// Нет метаданных — нечего восстанавливать.
		return
	}
	first := chunks[0]
	total := first.ChunkTotal
	recipient := first.Recipient
	if total <= 0 || recipient == "" {
		return
	}

	// Восстанавливаем deliveredCount по Status чанков.
	delivered := 0
	for _, c := range chunks {
		if c.Status >= StatusDelivered {
			delivered++
		}
	}
	n.deliveredCountMu.Lock()
	n.deliveredCount[mediaID] = delivered
	n.deliveredCountMu.Unlock()

	if delivered >= total {
		log.Printf("[RESUME] %s already delivered (%d/%d)", mediaID, delivered, total)
		return
	}

	log.Printf("[RESUME] %s incomplete (%d/%d) — resuming", mediaID, delivered, total)

	// Восстанавливаем unackedChunks для неподтверждённых чанков.
	restored := 0
	for _, c := range chunks {
		if c.Status >= StatusDelivered {
			continue
		}
		n.unackedChunksMu.Lock()
		n.unackedChunks[c.ID] = retryInfo{
			mediaID:    c.MediaID,
			mediaType:  c.MediaType,
			chunkIndex: c.ChunkIndex,
			recipient:  c.Recipient,
			msgID:      c.ID,
			fileName:   c.FileName,
			fileSize:   c.FileSize,
			chunkTotal: c.ChunkTotal,
			ttlSeconds: c.TtlPeriodSeconds,
			ttlMode:    c.TtlMode,
			expiresAt:  c.ExpiresAt,
			attempts:   0,
			lastSent:   time.Now(),
		}
		n.unackedChunksMu.Unlock()
		restored++
	}
	log.Printf("[RESUME] %s: restored %d chunks to unackedChunks", mediaID, restored)
}

// node/media.go
