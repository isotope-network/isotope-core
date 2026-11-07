package core

// D:\isotope\node\node_service.go
// node_service.go — сервисные сообщения: [DELIVERED], [READ], [FILE_COMPLETE],
// [TTL_UPDATE], статусы, батчи подтверждений, TTL after_read.
// Вынесено из node.go.

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// isServiceType — классификация. Только тип, без побочных эффектов.
// Сервисные сообщения не идут в UI.
func (n *Node) isServiceType(t MessageType) bool {
	switch t {
	case TypeDelivered, TypeRead, TypeContactRequest, TypeContactAccept, TypeContactReject,
		TypeContactHello, TypeContactHelloAck, TypeTtlUpdate, TypeFileComplete,
		TypeFileOffer, TypeFileAccept, TypeFileDecline, TypeFileCancel, TypeFileAbort,
		TypeFileAbortAck:
		return true
	}
	return false
}

// handleServiceMessage — обработка сервисного сообщения.
// Вызывается только если Recipient == myID (проверка в handleStream).
func (n *Node) handleServiceMessage(m Message) {
	switch m.Type {
	case TypeDelivered:
		refs := m.Refs
		if len(refs) == 0 && m.Ref != "" {
			refs = []string{m.Ref}
		}
		if m.ReadEnabled != nil && n.contacts != nil {
			if cur, ok := n.GetContact(m.Sender); ok && cur.ReadEnabled != *m.ReadEnabled {
				_ = n.SetContactReadEnabled(m.Sender, *m.ReadEnabled)
			}
		}
		for _, ref := range refs {
			n.handleDeliveredRef(ref, m.Sender)
		}

	case TypeRead:
		refs := m.Refs
		if len(refs) == 0 && m.Ref != "" {
			refs = []string{m.Ref}
		}
		for _, ref := range refs {
			statusToSet := StatusRead
			if !n.myReadEnabled {
				statusToSet = StatusHidden
			}
			if msg, ok := n.findMyMessageByID(ref); ok {
				if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
					all := n.memory.GetAll()
					for _, cm := range all {
						if cm.MediaID == msg.MediaID {
							n.setMessageStatus(cm.ID, statusToSet)
						}
					}
					log.Printf("[SERVICE] read ack (file %s, %d chunks) ref=%s from=%s",
						msg.MediaID, msg.ChunkTotal, ref, m.Sender)
				} else {
					n.setMessageStatus(ref, statusToSet)
					log.Printf("[SERVICE] read ack ref=%s from=%s", ref, m.Sender)
				}
			} else {
				all := n.memory.GetAll()
				found := false
				for _, cm := range all {
					if cm.MediaID == ref {
						n.setMessageStatus(cm.ID, statusToSet)
						found = true
					}
				}
				if found {
					log.Printf("[SERVICE] read ack (file mediaID=%s) from=%s", ref, m.Sender)
				} else {
					n.setMessageStatus(ref, statusToSet)
					log.Printf("[SERVICE] read ack ref=%s (not found in memory) from=%s", ref, m.Sender)
				}
			}

			if msg, ok := n.findMyMessageByID(ref); ok {
				if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
					expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
					if n.memory.SetExpiresAt(ref, expiresAt) {
						log.Printf("[TTL] after_read: set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
						n.scheduleSaveState()
						if updated, ok := n.findMyMessageByID(ref); ok {
							if data, err := json.Marshal(updated); err == nil {
								if n.messageHook != nil {
									n.messageHook(string(data))
								}
							}
						}
					}
				}
			}
		}

	case TypeContactRequest:
		n.handleContactRequest(m)

	case TypeContactAccept:
		n.handleContactAccept(m)

	case TypeContactReject:
		n.handleContactReject(m)

	case TypeContactHello:
		n.handleContactHello(m)

	case TypeContactHelloAck:
		n.handleContactHelloAck(m)

	case TypeTtlUpdate:
		n.handleTtlUpdate(m)

	case TypeFileComplete:
		n.handleFileComplete(m)

	case TypeFileOffer:
		n.handleFileOffer(m)

	case TypeFileAccept:
		n.handleFileAccept(m)

	case TypeFileDecline:
		n.handleFileDecline(m)

	case TypeFileCancel:
		n.handleFileCancel(m)

	case TypeFileAbort:
		n.handleFileAbort(m)

	case TypeFileAbortAck:
		n.handleFileAbortAck(m)
	}
}

// handleFileComplete — получатель подтвердил сборку файла.
func (n *Node) handleFileComplete(m Message) {
	if m.Ref == "" {
		log.Printf("[FILE_COMPLETE] empty ref, dropped")
		return
	}
	mediaID := m.Ref
	log.Printf("[FILE_COMPLETE] mediaID=%s from=%s", mediaID, m.Sender)

	// v2.0.7+: файл всё-таки принят — убрать из aborted (не нужен).
	if n.abortedStore != nil {
		_ = n.abortedStore.Remove(mediaID)
	}

	statusToSet := StatusDelivered
	peerReadEnabled := n.getPeerReadEnabled(m.Sender)
	if !n.myReadEnabled || !peerReadEnabled {
		statusToSet = StatusHidden
	}
	all := n.memory.GetAll()
	total := 0
	for _, cm := range all {
		if cm.MediaID == mediaID {
			n.setMessageStatus(cm.ID, statusToSet)
			if total == 0 {
				total = cm.ChunkTotal
			}
		}
	}
	if total > 0 {
		n.deliveredCountMu.Lock()
		n.deliveredCount[mediaID] = total
		n.deliveredCountMu.Unlock()
		n.pushFileStatus(mediaID, total, total)
	}

	sentPath := n.sentPath(mediaID)
	if err := os.Remove(sentPath); err == nil {
		log.Printf("[SENT] removed sent file %s (file_complete from %s)", mediaID, m.Sender)
	}
}

// SendFileComplete — получатель → отправитель: файл собран.
func (n *Node) SendFileComplete(mediaID, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if mediaID == "" || recipient == "" {
		return fmt.Errorf("mediaID and recipient are required")
	}
	id := generateMsgID(fmt.Sprintf("file_complete:%s:%d", mediaID, time.Now().UnixNano()))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        TypeFileComplete,
		Ref:         mediaID,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}
	n.flushDelivered(recipient)
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		log.Printf("[FILE_COMPLETE] send failed mediaID=%s: %v", mediaID, err)
		return err
	}
	log.Printf("[FILE_COMPLETE] sent mediaID=%s to=%s", mediaID, recipient)
	return nil
}

// handleTtlUpdate — обновляет ExpiresAt для сообщения по ref.
func (n *Node) handleTtlUpdate(m Message) {
	if m.Ref == "" {
		return
	}
	seconds, err := strconv.Atoi(m.Text)
	if err != nil || seconds <= 0 {
		log.Printf("[TTL_UPDATE] invalid seconds %q ref=%s", m.Text, m.Ref)
		return
	}
	expiresAt := time.Now().Add(time.Duration(seconds) * time.Second)
	if n.memory.SetExpiresAt(m.Ref, expiresAt) {
		log.Printf("[TTL_UPDATE] set ExpiresAt for %s (+%ds)", m.Ref, seconds)
		n.scheduleSaveState()
		if updated, ok := n.findMyMessageByID(m.Ref); ok {
			if data, err := json.Marshal(updated); err == nil {
				if n.messageHook != nil {
					n.messageHook(string(data))
				}
			}
		}
	} else {
		log.Printf("[TTL_UPDATE] message not found ref=%s", m.Ref)
	}
}

// SendDelivered — отправляет подтверждение доставки сообщения отправителю.
func (n *Node) SendDelivered(ref, recipient string) error {
	return n.sendConfirmation(TypeDelivered, ref, recipient)
}

// SendDeliveredBatch — отправляет батч подтверждений доставки.
func (n *Node) SendDeliveredBatch(refs []string, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}
	if len(refs) == 0 {
		return nil
	}
	id := generateMsgID(fmt.Sprintf("delivered-batch:%d", len(refs)))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        TypeDelivered,
		Refs:        refs,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		log.Printf("[CONFIRM] delivered-batch (%d refs) to %s failed: %v", len(refs), recipient, err)
		return err
	}
	log.Printf("[CONFIRM] delivered-batch (%d refs) to %s", len(refs), recipient)
	return nil
}

// queueDelivered — добавляет ref в буфер.
func (n *Node) queueDelivered(sender, ref string) {
	if sender == "" || ref == "" {
		return
	}
	n.pendingDeliveredMu.Lock()
	already := false
	for _, r := range n.pendingDelivered[sender] {
		if r == ref {
			already = true
			break
		}
	}
	if already {
		n.pendingDeliveredMu.Unlock()
		return
	}
	n.pendingDelivered[sender] = append(n.pendingDelivered[sender], ref)
	count := len(n.pendingDelivered[sender])

	needFlush := count >= 10
	if needFlush {
		if t, ok := n.pendingDeliveredTimers[sender]; ok {
			t.Stop()
			delete(n.pendingDeliveredTimers, sender)
		}
	} else {
		if _, hasTimer := n.pendingDeliveredTimers[sender]; !hasTimer {
			n.pendingDeliveredTimers[sender] = time.AfterFunc(2*time.Second, func() {
				n.pendingDeliveredMu.Lock()
				delete(n.pendingDeliveredTimers, sender)
				n.pendingDeliveredMu.Unlock()
				n.flushDelivered(sender)
			})
		}
	}
	n.pendingDeliveredMu.Unlock()

	if needFlush {
		go n.flushDelivered(sender)
	}
}

// flushDelivered — отправляет батч [DELIVERED] для указанного sender.
func (n *Node) flushDelivered(sender string) {
	n.pendingDeliveredMu.Lock()
	if t, ok := n.pendingDeliveredTimers[sender]; ok {
		t.Stop()
		delete(n.pendingDeliveredTimers, sender)
	}
	refs, ok := n.pendingDelivered[sender]
	if !ok || len(refs) == 0 {
		n.pendingDeliveredMu.Unlock()
		return
	}
	refsCopy := make([]string, len(refs))
	copy(refsCopy, refs)
	delete(n.pendingDelivered, sender)
	n.pendingDeliveredMu.Unlock()

	if err := n.SendDeliveredBatch(refsCopy, sender); err != nil {
		log.Printf("[CONFIRM] delivered-batch flush failed for %s: %v", sender, err)
		n.pendingDeliveredMu.Lock()
		existing := make(map[string]bool, len(n.pendingDelivered[sender]))
		for _, r := range n.pendingDelivered[sender] {
			existing[r] = true
		}
		for _, r := range refsCopy {
			if !existing[r] {
				n.pendingDelivered[sender] = append(n.pendingDelivered[sender], r)
				existing[r] = true
			}
		}
		n.pendingDeliveredMu.Unlock()
		time.AfterFunc(5*time.Second, func() {
			n.flushDelivered(sender)
		})
	}
}

// handleDeliveredRef — обработка одного ref из [DELIVERED] (одиночного или батча).
func (n *Node) handleDeliveredRef(ref, sender string) {
	if ref == "" {
		return
	}
	n.removePendingByRef(ref)

	n.unackedChunksMu.Lock()
	delete(n.unackedChunks, ref)
	n.unackedChunksMu.Unlock()

	peerReadEnabled := n.getPeerReadEnabled(sender)
	if !n.myReadEnabled || !peerReadEnabled {
		isFileChunkHidden := false
		if msg, ok := n.findMyMessageByID(ref); ok && isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
			isFileChunkHidden = true
		}
		if !isFileChunkHidden {
			n.setMessageStatus(ref, StatusHidden)
			log.Printf("[SERVICE] delivered ack (hidden) ref=%s from=%s", ref, sender)
		}

		if msg, ok := n.findMyMessageByID(ref); ok {
			if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
				expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
				if n.memory.SetExpiresAt(ref, expiresAt) {
					log.Printf("[TTL] after_read → hard (hidden): set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
					n.scheduleSaveState()
					if updated, ok := n.findMyMessageByID(ref); ok {
						if data, err := json.Marshal(updated); err == nil {
							if n.messageHook != nil {
								n.messageHook(string(data))
							}
						}
					}
					if err := n.SendTtlUpdate(ref, msg.TtlPeriodSeconds, sender); err != nil {
						log.Printf("[TTL_UPDATE] send failed ref=%s: %v", ref, err)
					}
				}
			}
		}
	} else {
		isFileChunk := false
		if msg, ok := n.findMyMessageByID(ref); ok && isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
			isFileChunk = true
		}
		if !isFileChunk {
			n.setMessageStatus(ref, StatusDelivered)
			log.Printf("[SERVICE] delivered ack ref=%s from=%s", ref, sender)
		}
	}

	if msg, ok := n.findMyMessageByID(ref); ok {
		if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 && msg.MediaID != "" {
			n.deliveredCountMu.Lock()
			cur := n.deliveredCount[msg.MediaID]
			total := msg.ChunkTotal
			mediaID := msg.MediaID
			if cur < total {
				n.deliveredCount[msg.MediaID] = cur + 1
				cur = cur + 1
			}
			n.deliveredCountMu.Unlock()

			n.pushFileStatus(mediaID, cur, total)

			if cur >= total {
				n.unackedChunksMu.Lock()
				for id, info := range n.unackedChunks {
					if info.mediaID == mediaID {
						delete(n.unackedChunks, id)
					}
				}
				n.unackedChunksMu.Unlock()
			}
		}
	}
}

// SendTtlUpdate — отправляет получателю [TTL_UPDATE] с expires_in_seconds.
func (n *Node) SendTtlUpdate(ref string, seconds int, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if ref == "" || recipient == "" || seconds <= 0 {
		return fmt.Errorf("ref, recipient, seconds required")
	}
	text := strconv.Itoa(seconds)
	return n.sendTtlUpdate(ref, text, recipient)
}

// sendTtlUpdate — общая логика отправки [TTL_UPDATE].
func (n *Node) sendTtlUpdate(ref, text, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	id := generateMsgID(ref + ":" + recipient)
	msg := Message{
		ID:        id,
		Text:      text,
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Version:   0,
		Type:      TypeTtlUpdate,
		Ref:       ref,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:     true,
		Weight:    0.5,
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("sendServiceViaBootstrap: %w", err)
	}
	return nil
}

// SendRead — отправляет подтверждение прочтения сообщения отправителю.
func (n *Node) SendRead(ref, recipient string) error {
	if msg, ok := n.findMyMessageByID(ref); ok {
		if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
			if msg.ChunkIndex != msg.ChunkTotal-1 {
				return nil
			}
		}
	}
	if msg, ok := n.findMyMessageByID(ref); ok {
		if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
			expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
			if n.memory.SetExpiresAt(ref, expiresAt) {
				log.Printf("[TTL] after_read (recipient): set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
				n.scheduleSaveState()
				if updated, ok := n.findMyMessageByID(ref); ok {
					if data, err := json.Marshal(updated); err == nil {
						if n.messageHook != nil {
							n.messageHook(string(data))
						}
					}
				}
			}
		}
	}
	return n.sendConfirmation(TypeRead, ref, recipient)
}

// SendReadBatch — отправляет батч подтверждений прочтения.
func (n *Node) SendReadBatch(refs []string, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}
	if len(refs) == 0 {
		return nil
	}
	if !n.myReadEnabled {
		return nil
	}

	seenMedia := make(map[string]bool)
	filtered := make([]string, 0, len(refs))
	for _, ref := range refs {
		if msg, ok := n.findMyMessageByID(ref); ok {
			if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
				if seenMedia[msg.MediaID] {
					continue
				}
				seenMedia[msg.MediaID] = true
				filtered = append(filtered, msg.MediaID)
				continue
			}
		}
		filtered = append(filtered, ref)
	}
	refs = filtered
	if len(refs) == 0 {
		return nil
	}

	for _, ref := range refs {
		if msg, ok := n.findMyMessageByID(ref); ok {
			if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
				expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
				if n.memory.SetExpiresAt(ref, expiresAt) {
					log.Printf("[TTL] after_read (recipient batch): set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
					n.scheduleSaveState()
					if updated, ok := n.findMyMessageByID(ref); ok {
						if data, err := json.Marshal(updated); err == nil {
							if n.messageHook != nil {
								n.messageHook(string(data))
							}
						}
					}
				}
			}
		}
	}

	id := generateMsgID(fmt.Sprintf("read-batch:%d", len(refs)))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        TypeRead,
		Refs:        refs,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[CONFIRM] delivered-batch marshal failed: %v", err)
		return err
	}
	if !n.sendToRecipient(msg, data, false) {
		log.Printf("[CONFIRM] delivered-batch (%d refs) to %s failed (sendToRecipient)", len(refs), recipient)
		return fmt.Errorf("delivered-batch send failed")
	}
	log.Printf("[CONFIRM] delivered-batch (%d refs) to %s", len(refs), recipient)
	return nil
}

// MarkReadLocally — помечает входящие сообщения как прочитанные локально.
func (n *Node) MarkReadLocally(refs []string) int {
	if len(refs) == 0 {
		return 0
	}
	changed := n.memory.MarkReadLocally(refs)
	if changed > 0 {
		log.Printf("[READ] marked %d messages as read locally", changed)
		go func() {
			if err := n.saveState(); err != nil {
				log.Printf("[STATE] save failed: %v", err)
			}
		}()
	}
	return changed
}

// sendConfirmation — общая логика отправки [DELIVERED]/[READ].
func (n *Node) sendConfirmation(msgType MessageType, ref, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if ref == "" || recipient == "" {
		return fmt.Errorf("ref and recipient are required")
	}
	log.Printf("[CONFIRM] sendConfirmation: type=%s ref=%s recipient=%s me=%s",
		msgTypeString(msgType), ref, recipient, n.host.ID().String())
	if msgType == TypeRead && !n.myReadEnabled {
		return nil
	}

	id := generateMsgID(fmt.Sprintf("%s:%s", ref, msgTypeString(msgType)))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        msgType,
		Ref:         ref,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if n.fallbackSend(recipient, data, false) {
		log.Printf("[CONFIRM] %s ref=%s to %s (via fallback)", msgTypeString(msgType), ref, recipient)
		return nil
	}
	return fmt.Errorf("no route for confirmation")
}

// msgTypeString — строковое имя типа для логов.
func msgTypeString(t MessageType) string {
	switch t {
	case TypeDelivered:
		return "delivered"
	case TypeRead:
		return "read"
	case TypeContactRequest:
		return "contact_request"
	case TypeContactAccept:
		return "contact_accept"
	case TypeContactReject:
		return "contact_reject"
	case TypeContactHello:
		return "contact_hello"
	case TypeContactHelloAck:
		return "contact_hello_ack"
	case TypeTtlUpdate:
		return "ttl_update"
	default:
		return "message"
	}
}

// setMessageStatus — устанавливает статус сообщения по ID.
func (n *Node) setMessageStatus(id string, status MessageStatus) {
	if id == "" {
		return
	}
	if n.memory.SetStatus(id, status) {
		log.Printf("[STATUS] %s → %d", id, status)
		n.scheduleSaveState()
	}
}

// scheduleSaveState — отложенная запись state (throttle 5 сек).
func (n *Node) scheduleSaveState() {
	n.mu.Lock()
	if n.saveStateScheduled {
		n.mu.Unlock()
		return
	}
	n.saveStateScheduled = true
	n.mu.Unlock()

	go func() {
		time.Sleep(5 * time.Second)
		n.mu.Lock()
		n.saveStateScheduled = false
		n.mu.Unlock()
		if err := n.saveState(); err != nil {
			log.Printf("[STATE] save failed: %v", err)
		}
	}()
}

// getMessageStatus — возвращает статус сообщения по ID.
func (n *Node) getMessageStatus(id string) MessageStatus {
	if msg, ok := n.findMyMessageByID(id); ok {
		return msg.Status
	}
	return 0
}

// GetMessageStatuses — возвращает map статусов для UI.
func (n *Node) GetMessageStatuses() map[string]MessageStatus {
	all := n.memory.GetAll()
	result := make(map[string]MessageStatus, len(all))
	for _, msg := range all {
		if msg.Status > 0 {
			if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
				continue
			}
			result[msg.ID] = msg.Status
		}
	}
	return result
}

// D:\isotope\node\node_service.go
