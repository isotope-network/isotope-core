package core

// D:\isotope\node\node_settings.go
// node_settings.go — настройки: read_enabled, display_name, уведомления,
// TTL, Sphinx, Mixing, Masking, deleted, SendMessage/SendToPeer, GetMessages/Peers/Weight.
// Вынесено из node.go.

import (
	"fmt"
	"log"
	"time"
)

// SetMyReadEnabled — устанавливает мою настройку "делюсь ли статусом прочтения".
func (n *Node) SetMyReadEnabled(enabled bool) {
	n.myReadEnabled = enabled
	if n.settingsStore != nil {
		if err := n.settingsStore.SetMyReadEnabled(enabled); err != nil {
			log.Printf("[SETTINGS] save my_read_enabled failed: %v", err)
		}
	}
}

// GetMyReadEnabled — возвращает текущую настройку (из кэша).
func (n *Node) GetMyReadEnabled() bool {
	return n.myReadEnabled
}

// SetMyDisplayName — устанавливает представление по умолчанию.
func (n *Node) SetMyDisplayName(name string) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetMyDisplayName(name)
}

// GetMyDisplayName — возвращает представление по умолчанию.
func (n *Node) GetMyDisplayName() string {
	if n.settingsStore == nil {
		return ""
	}
	return n.settingsStore.GetMyDisplayName()
}

// GetShowNotificationContent — показывать ли содержимое в уведомлениях.
func (n *Node) GetShowNotificationContent() bool {
	if n.settingsStore == nil {
		return true
	}
	return n.settingsStore.GetShowNotificationContent()
}

// SetShowNotificationContent — устанавливает настройку и сохраняет.
func (n *Node) SetShowNotificationContent(enabled bool) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetShowNotificationContent(enabled)
}

// GetNotificationCount — показывать ли количество непрочитанных в шторке.
func (n *Node) GetNotificationCount() bool {
	if n.settingsStore == nil {
		return false
	}
	return n.settingsStore.GetNotificationCount()
}

// SetNotificationCount — устанавливает и сохраняет.
func (n *Node) SetNotificationCount(enabled bool) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetNotificationCount(enabled)
}

// GetNotificationNames — показывать ли имена контактов в шторке.
func (n *Node) GetNotificationNames() bool {
	if n.settingsStore == nil {
		return false
	}
	return n.settingsStore.GetNotificationNames()
}

// SetNotificationNames — устанавливает и сохраняет.
func (n *Node) SetNotificationNames(enabled bool) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetNotificationNames(enabled)
}

// GetNotificationProgress — показывать ли прогресс передачи в шторке.
func (n *Node) GetNotificationProgress() bool {
	if n.settingsStore == nil {
		return false
	}
	return n.settingsStore.GetNotificationProgress()
}

// SetNotificationProgress — устанавливает и сохраняет.
func (n *Node) SetNotificationProgress(enabled bool) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetNotificationProgress(enabled)
}

// GetDeletedPeers — возвращает список удалённых peerID (для Dart).
func (n *Node) GetDeletedPeers() []string {
	if n.deleted == nil {
		return []string{}
	}
	return n.deleted.GetAll()
}

// RemoveFromDeleted — убирает peerID из списка удалённых.
func (n *Node) RemoveFromDeleted(peerID string) error {
	if n.deleted == nil {
		return fmt.Errorf("deleted store not initialized")
	}
	return n.deleted.RemoveFromDeleted(peerID)
}

// IsDeleted — true, если peerID в списке удалённых.
func (n *Node) IsDeleted(peerID string) bool {
	if n.deleted == nil {
		return false
	}
	return n.deleted.IsDeleted(peerID)
}

// SetTtl — устанавливает период и режим удаления сообщений.
func (n *Node) SetTtl(period string, mode string) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	var modePtr *string
	if mode != "" && period != "never" && period != "forever" {
		m := mode
		modePtr = &m
	}
	return n.settingsStore.SetTtl(period, modePtr)
}

// GetTtl — возвращает период и режим удаления сообщений.
func (n *Node) GetTtl() (string, string) {
	if n.settingsStore == nil {
		return "never", ""
	}
	period, modePtr := n.settingsStore.GetTtl()
	mode := ""
	if modePtr != nil {
		mode = *modePtr
	}
	return period, mode
}

// SetSphinxEnabled — включает/выключает Sphinx на лету + сохраняет в настройках.
func (n *Node) SetSphinxEnabled(enabled bool) error {
	if n.settingsStore != nil {
		if err := n.settingsStore.SetSphinxEnabled(enabled); err != nil {
			return err
		}
	}
	n.mu.Lock()
	n.useOnion = enabled
	n.mu.Unlock()
	log.Printf("[SPHINX] enabled=%v", enabled)
	return nil
}

// GetSphinxEnabled — текущее значение Sphinx.
func (n *Node) GetSphinxEnabled() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.useOnion
}

// SetMixingEnabled — включает/выключает Mixing на лету + сохраняет в настройках.
func (n *Node) SetMixingEnabled(enabled bool) error {
	if n.settingsStore != nil {
		if err := n.settingsStore.SetMixingEnabled(enabled); err != nil {
			return err
		}
	}
	n.mu.Lock()
	n.mixingEnabled = enabled
	n.mu.Unlock()
	log.Printf("[MIXING] enabled=%v", enabled)
	return nil
}

// GetMixingEnabled — текущее значение Mixing.
func (n *Node) GetMixingEnabled() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.mixingEnabled
}

// GetMaskingEnabled — текущее значение dummy-трафика (v2.0.4 d).
func (n *Node) GetMaskingEnabled() bool {
	if n.settingsStore == nil {
		return false
	}
	return n.settingsStore.GetMaskingEnabled()
}

// SetMaskingEnabled — включает/выключает dummy-трафик на лету + сохраняет.
func (n *Node) SetMaskingEnabled(enabled bool) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	log.Printf("[MASKING] enabled=%v", enabled)
	return n.settingsStore.SetMaskingEnabled(enabled)
}

// GetFileRequestThreshold — порог запроса на приём файла (МБ).
// -1 — всегда. 0 — никогда. N — от N МБ.
func (n *Node) GetFileRequestThreshold() int {
	if n.settingsStore == nil {
		return -1
	}
	return n.settingsStore.GetFileRequestThreshold()
}

// SetFileRequestThreshold — устанавливает порог запроса и сохраняет.
// -1 — всегда. 0 — никогда. N — от N МБ.
func (n *Node) SetFileRequestThreshold(thresholdMB int) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	log.Printf("[SETTINGS] file_request_threshold=%d MB", thresholdMB)
	return n.settingsStore.SetFileRequestThreshold(thresholdMB)
}

// SendMessage — отправляет сообщение всем пирам (broadcast).
func (n *Node) SendMessage(text string, ttlPeriod string, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	id := generateMsgID(text)
	n.processMessageInternal(text, n.host.ID().String(), true, expiresAt, id, "", 0, "", TypeMessage, false, ttlSeconds, ttlMode)
	return id, nil
}

// SendToPeer — отправляет сообщение конкретному пиру (адресно).
func (n *Node) SendToPeer(peerID string, text string, ttlPeriod string, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if peerID == "" {
		return "", fmt.Errorf("peerID is required")
	}
	log.Printf("[SEND] SendToPeer: peerID=%s me=%s text=%q ttlPeriod=%s ttlMode=%s",
		peerID, n.host.ID().String(), truncate(text, 40), ttlPeriod, ttlMode)

	encrypted, err := n.encryptForRecipient(peerID, text)
	if err != nil {
		return "", fmt.Errorf("encrypt failed: %w", err)
	}

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	id := generateMsgID(text)
	n.processMessageInternal(encrypted, n.host.ID().String(), true, expiresAt, id, peerID, MESSAGE_VERSION_E2E, text, TypeMessage, false, ttlSeconds, ttlMode)
	return id, nil
}

// GetMessages — возвращает все НЕ истёкшие сообщения.
func (n *Node) GetMessages() []Message {
	all := n.memory.GetAll()
	now := time.Now()
	alive := make([]Message, 0, len(all))
	for _, m := range all {
		if !m.ExpiresAt.IsZero() && now.After(m.ExpiresAt) {
			continue
		}
		if isChunkedMediaType(m.MediaType) && m.ChunkTotal > 0 {
			m.Text = ""
			m.PlainText = ""
		}
		alive = append(alive, m)
	}
	return alive
}

// GetPeers — возвращает список активных пиров.
func (n *Node) GetPeers() []string {
	if n.host == nil {
		return []string{}
	}
	peers := n.host.Network().Peers()
	result := make([]string, 0, len(peers))
	for _, p := range peers {
		result = append(result, p.String())
	}
	return result
}

// GetWeight — возвращает вес узла.
func (n *Node) GetWeight() float64 {
	msgs := n.memory.GetActiveMessages(0.5)
	if len(msgs) == 0 {
		return 0.5
	}
	total := 0.0
	for _, m := range msgs {
		total += m.Weight
	}
	return total / float64(len(msgs))
}

// node/node_settings.go
