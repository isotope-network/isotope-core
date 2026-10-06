// node/media.go
package core

import (
	"encoding/json"
	"fmt"
	"log"
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

// node/media.go
