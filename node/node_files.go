package core

// D:\isotope\node\node_files.go
// node_files.go — передача файлов: chunkWorker, retryUnacked, pushFileStatus,
// GetChunk, ClearChunkData, FileAck, DeleteFile, GetFileStatuses.
// Вынесено из node.go.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// chunkWorker — воркер отправки файловых чанков.
// Читает из chunkQueue, маршалит, отправляет через sendToRecipient.
// 10 воркеров запускаются в InitP2P.
func (n *Node) chunkWorker() {
	if n.isRelay {
		return
	}
	for msg := range n.chunkQueue {
		// v2.0.7+: отправитель прервал — не отправлять.
		n.abortedMu.Lock()
		aborted := n.aborted[msg.MediaID]
		n.abortedMu.Unlock()
		if aborted {
			log.Printf("[CHUNK] aborted, skip %s[%d]", msg.MediaID, msg.ChunkIndex)
			continue
		}

		msg.PlainText = ""
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[CHUNK] marshal failed id=%s: %v", msg.ID, err)
			continue
		}
		n.unackedChunksMu.Lock()
		info := n.unackedChunks[msg.ID]
		info.mediaID = msg.MediaID
		info.mediaType = msg.MediaType
		info.chunkIndex = msg.ChunkIndex
		info.recipient = msg.Recipient
		info.msgID = msg.ID
		info.fileName = msg.FileName
		info.fileSize = msg.FileSize
		info.chunkTotal = msg.ChunkTotal
		info.ttlSeconds = msg.TtlPeriodSeconds
		info.ttlMode = msg.TtlMode
		info.expiresAt = msg.ExpiresAt
		if info.lastSent.IsZero() {
			info.attempts = 0
		}
		info.lastSent = time.Now()
		n.unackedChunks[msg.ID] = info
		n.unackedChunksMu.Unlock()

		if msg.MixingEnabled {
			n.sphinxFileSem <- struct{}{}
			if n.sendViaSphinx(msg) {
				<-n.sphinxFileSem
				continue
			}
			<-n.sphinxFileSem
			log.Printf("[CHUNK] sphinx failed for %s[%d], fallback to direct", msg.MediaID, msg.ChunkIndex)
		}
		n.sendToRecipient(msg, data, true)
	}
}

// fillFileChunkFromSent — читает чанк файла из sent/<MediaID>.bin,
// шифрует, дополняет msg (Text/PlainText).
func (n *Node) fillFileChunkFromSent(msg *Message) error {
	path := n.sentPath(msg.MediaID)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open sent failed: %w", err)
	}
	defer f.Close()

	buf := make([]byte, ChunkSize)
	off := int64(msg.ChunkIndex) * int64(ChunkSize)
	nr, err := f.ReadAt(buf, off)
	if err != nil && err != io.EOF {
		return fmt.Errorf("ReadAt failed: %w", err)
	}
	if nr == 0 {
		return fmt.Errorf("empty chunk")
	}

	encrypted, err := n.encryptBytesForRecipient(msg.Recipient, buf[:nr])
	if err != nil {
		return fmt.Errorf("encrypt failed: %w", err)
	}
	msg.PlainText = base64.StdEncoding.EncodeToString(buf[:nr])
	msg.Text = encrypted
	return nil
}

// pushFileStatus — push агрегата по файлу в Dart.
// Throttle: не чаще раза в секунду на mediaID.
func (n *Node) pushFileStatus(mediaID string, delivered, total int) {
	if mediaID == "" || total <= 0 {
		return
	}
	// v2.0.7+: aborted — не push прогресс (шторка не висит).
	if n.abortedStore != nil && n.abortedStore.IsAborted(mediaID) {
		return
	}
	n.fileStatusMu.Lock()
	last := n.fileStatusLastPush[mediaID]
	complete := delivered >= total
	alreadyCompleted := n.completedFiles[mediaID]
	if complete && alreadyCompleted {
		if !last.IsZero() && time.Since(last) < 5*time.Second {
			n.fileStatusMu.Unlock()
			return
		}
	}
	if !complete && time.Since(last) < time.Second {
		n.fileStatusMu.Unlock()
		return
	}
	n.fileStatusLastPush[mediaID] = time.Now()
	if complete {
		n.completedFiles[mediaID] = true
	}
	n.fileStatusMu.Unlock()

	if n.messageHook == nil {
		return
	}

	payload := map[string]interface{}{
		"file_status": map[string]interface{}{
			"mediaID":   mediaID,
			"delivered": delivered,
			"total":     total,
		},
	}
	if data, err := json.Marshal(payload); err == nil {
		n.messageHook(string(data))
	}

	if complete {
		done := map[string]interface{}{
			"file_complete": map[string]interface{}{
				"mediaID": mediaID,
			},
		}
		if data, err := json.Marshal(done); err == nil {
			n.messageHook(string(data))
		}
	}
}

// retryBackoff — интервалы переотправки чанков (d, 2026-10-29).
// Сумма ~1 ч. После — giving up (файл не дожат, честно).
var retryBackoff = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
	10 * time.Minute,
	10 * time.Minute,
	10 * time.Minute,
	10 * time.Minute,
	10 * time.Minute,
}

// retryLoop — периодическая проверка unackedChunks.
// Интервал — retryBackoff[attempts] (30 сек → 10 мин, потолок).
// После len(retryBackoff) попыток (~1 ч) — сдаться, удалить.
func (n *Node) retryLoop() {
	go func() {
		for {
			time.Sleep(15 * time.Second)
			n.retryUnacked()
		}
	}()
}

// retryUnacked — одна итерация проверки unackedChunks.
func (n *Node) retryUnacked() {
	now := time.Now()
	type retryItem struct {
		id   string
		info retryInfo
	}
	var toRetry []retryItem

	n.unackedChunksMu.Lock()
	for id, info := range n.unackedChunks {
		// v2.0.7+: aborted — не досылать. Отправитель/получатель прервал.
		n.abortedMu.Lock()
		aborted := n.aborted[info.mediaID]
		n.abortedMu.Unlock()
		if aborted {
			delete(n.unackedChunks, id)
			continue
		}
		if info.attempts >= len(retryBackoff) {
			log.Printf("[RETRY] giving up on %s after %d attempts", id, info.attempts)
			delete(n.unackedChunks, id)
			continue
		}
		backoff := retryBackoff[info.attempts]

		if now.Sub(info.lastSent) < backoff {
			continue
		}
		toRetry = append(toRetry, retryItem{id: id, info: info})
	}
	n.unackedChunksMu.Unlock()

	for _, item := range toRetry {
		msg, err := n.restoreChunkMessage(item.info)
		if err != nil {
			n.unackedChunksMu.Lock()
			info := n.unackedChunks[item.id]
			info.attempts++
			if info.attempts >= 3 {
				log.Printf("[RETRY] giving up on %s after %d failed restores: %v", item.id, info.attempts, err)
				delete(n.unackedChunks, item.id)
			} else {
				log.Printf("[RETRY] restore %s failed (attempt %d): %v", item.id, info.attempts, err)
				info.lastSent = time.Now()
				n.unackedChunks[item.id] = info
			}
			n.unackedChunksMu.Unlock()
			continue
		}
		select {
		case n.chunkQueue <- msg:
			n.unackedChunksMu.Lock()
			if info, ok := n.unackedChunks[item.id]; ok {
				info.attempts++
				info.lastSent = time.Now()
				n.unackedChunks[item.id] = info
				log.Printf("[RETRY] re-enqueue %s (attempt %d)", item.id, info.attempts)
			}
			n.unackedChunksMu.Unlock()
		case <-time.After(2 * time.Second):
			log.Printf("[RETRY] chunkQueue busy, deferring %s", item.id)
		}
	}
}

// restoreChunkMessage — восстанавливает Message файлового чанка
// из метаданных retryInfo и sent/<mediaID>.bin.
func (n *Node) restoreChunkMessage(info retryInfo) (Message, error) {
	if info.mediaID == "" || info.recipient == "" {
		return Message{}, fmt.Errorf("incomplete retryInfo")
	}
	msg := Message{
		ID:               info.msgID,
		Sender:           n.host.ID().String(),
		Recipient:        info.recipient,
		Version:          MESSAGE_VERSION_E2E,
		Type:             TypeMessage,
		MediaType:        info.mediaType,
		MediaID:          info.mediaID,
		ChunkIndex:       info.chunkIndex,
		ChunkTotal:       info.chunkTotal,
		FileName:         info.fileName,
		FileSize:         info.fileSize,
		TtlPeriodSeconds: info.ttlSeconds,
		TtlMode:          info.ttlMode,
		ExpiresAt:        info.expiresAt,
		Time:             time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:            true,
		Weight:           0.5,
	}
	readEnabled := n.myReadEnabled
	msg.ReadEnabled = &readEnabled
	if err := n.fillFileChunkFromSent(&msg); err != nil {
		return Message{}, fmt.Errorf("fillFileChunkFromSent: %w", err)
	}
	return msg, nil
}

// GetChunk — возвращает base64 одного чанка файла по mediaID + chunkIndex.
func (n *Node) GetChunk(mediaID string, chunkIndex int) (string, error) {
	if mediaID == "" {
		return "", fmt.Errorf("mediaID is required")
	}
	msg, ok := n.memory.GetFileChunk(mediaID, chunkIndex)
	if !ok {
		return "", fmt.Errorf("chunk not found: mediaID=%s index=%d", mediaID, chunkIndex)
	}
	if msg.IsOwn {
		return msg.PlainText, nil
	}
	return msg.Text, nil
}

// GetMessageChunkBase64 — обёртка над GetChunk для MethodChannel.
func (n *Node) GetMessageChunkBase64(mediaID string, chunkIndex int) (string, error) {
	return n.GetChunk(mediaID, chunkIndex)
}

// ClearChunkData — очищает Text/PlainText у конкретного чанка в memory.
func (n *Node) ClearChunkData(mediaID string, chunkIndex int) bool {
	if mediaID == "" {
		return false
	}
	return n.memory.ClearFileChunk(mediaID, chunkIndex)
}

// FileAck — подтверждает приём группы чанков файла от Dart.
func (n *Node) FileAck(mediaID string, indices []int) {
	if mediaID == "" || len(indices) == 0 {
		return
	}

	for _, idx := range indices {
		n.memory.ClearFileChunk(mediaID, idx)
	}

	n.pendingHookAcksMu.Lock()
	if w := n.pendingHookAcks[mediaID]; w != nil {
		for _, idx := range indices {
			delete(w.Indices, idx)
		}
		if len(w.Indices) == 0 {
			delete(n.pendingHookAcks, mediaID)
		}
	}
	n.pendingHookAcksMu.Unlock()

	log.Printf("[ACK] mediaID=%s count=%d", mediaID, len(indices))
}

// DeleteFile — удаляет все чанки файла из memory по mediaID.
func (n *Node) DeleteFile(mediaID string) int {
	if mediaID == "" {
		return 0
	}
	// v2.0.7+: пометить aborted — переживёт перезапуск, retry не досылает.
	n.abortedMu.Lock()
	n.aborted[mediaID] = true
	n.abortedMu.Unlock()
	if n.abortedStore != nil {
		_ = n.abortedStore.Add(mediaID)
	}
	all := n.memory.GetAll()
	removed := 0
	for _, m := range all {
		if m.MediaID == mediaID {
			if n.memory.Remove(m.ID) {
				removed++
			}
		}
	}
	sentPath := n.sentPath(mediaID)
	_ = os.Remove(sentPath)
	_ = os.Remove(n.metaPath(mediaID))
	// v2.0.7+: недособранные чанки на диске (получатель) — тоже удалить.
	// Без этого tryAssemble соберёт файл из старых chunks/ — файл «принят».
	_ = os.RemoveAll(filepath.Join(n.filesDirPath(), "chunks", mediaID))
	filesPattern := filepath.Join(n.filesDirPath(), mediaID+".*")
	if matches, err := filepath.Glob(filesPattern); err == nil {
		for _, p := range matches {
			_ = os.Remove(p)
		}
	}
	n.deliveredCountMu.Lock()
	delete(n.deliveredCount, mediaID)
	n.deliveredCountMu.Unlock()
	n.acceptedCountMu.Lock()
	delete(n.acceptedCount, mediaID)
	n.acceptedCountMu.Unlock()
	n.fileStatusMu.Lock()
	delete(n.completedFiles, mediaID)
	delete(n.fileStatusLastPush, mediaID)
	n.fileStatusMu.Unlock()
	n.pendingHookAcksMu.Lock()
	delete(n.pendingHookAcks, mediaID)
	n.pendingHookAcksMu.Unlock()

	// v2.0.7+: убрать чанки из unackedChunks — retry не досылает.
	n.unackedChunksMu.Lock()
	for id, info := range n.unackedChunks {
		if info.mediaID == mediaID {
			delete(n.unackedChunks, id)
		}
	}
	n.unackedChunksMu.Unlock()

	if removed > 0 {
		log.Printf("[DELETE] file %s — removed %d chunks", mediaID, removed)
		n.scheduleSaveState()
	}
	return removed
}

// GetAborted — возвращает список aborted mediaID (v2.0.7+, для Dart).
func (n *Node) GetAborted() []string {
	if n.abortedStore == nil {
		return []string{}
	}
	return n.abortedStore.GetAll()
}

// FileStatusEntry — статус файловой отправки для Dart.
type FileStatusEntry struct {
	MediaID   string `json:"media_id"`
	Delivered int    `json:"delivered"`
	Total     int    `json:"total"`
	Status    int    `json:"status"`
}

// GetFileStatuses — статус всех файловых отправок (delivered/total).
func (n *Node) GetFileStatuses() []FileStatusEntry {
	all := n.memory.GetAll()
	type acc struct {
		total       int
		delivered   int
		status      MessageStatus
		lastChunkID string
		isOwn       bool
	}
	m := make(map[string]*acc)
	for _, msg := range all {
		if !isChunkedMediaType(msg.MediaType) || msg.ChunkTotal <= 0 || msg.MediaID == "" {
			continue
		}
		a, ok := m[msg.MediaID]
		if !ok {
			a = &acc{total: msg.ChunkTotal, isOwn: msg.IsOwn}
			m[msg.MediaID] = a
		}
		if msg.Status >= StatusDelivered {
			a.delivered++
		}
		if msg.Status > a.status {
			a.status = msg.Status
		}
		if msg.ChunkIndex == msg.ChunkTotal-1 {
			a.lastChunkID = msg.ID
		}
	}
	result := make([]FileStatusEntry, 0, len(m))
	for id, a := range m {
		if !a.isOwn {
			if matches, _ := filepath.Glob(filepath.Join(n.filesDirPath(), id+".*")); len(matches) > 0 {
				a.delivered = a.total
				if a.status < StatusDelivered {
					a.status = StatusDelivered
				}
			}
		}
		result = append(result, FileStatusEntry{
			MediaID:   id,
			Delivered: a.delivered,
			Total:     a.total,
			Status:    int(a.status),
		})
	}
	return result
}

// D:\isotope\node\node_files.go
