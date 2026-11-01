package core

// D:\isotope\node\node_contacts.go
// node_contacts.go — контакты: handleContact*, SendContact*, AddContact,
// RemoveContact, temp contacts, nameForPeer, requests accept/reject.
// Вынесено из node.go.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// handleContactRequest — обрабатывает входящий запрос на контакт.
func (n *Node) handleContactRequest(m Message) {
	if m.Version != MESSAGE_VERSION_E2E {
		log.Printf("[SERVICE] contact_request without E2E, dropped")
		return
	}
	plaintext, err := n.decryptFromSender(m.Sender, m.Text)
	if err != nil {
		log.Printf("[SERVICE] contact_request decrypt failed: %v", err)
		return
	}
	var payload struct {
		Name        string `json:"name"`
		Ed25519Pub  string `json:"ed25519_pub"`
		X25519Pub   string `json:"x25519_pub"`
		Signature   string `json:"signature"`
		ReadEnabled bool   `json:"read_enabled"`
	}
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		log.Printf("[SERVICE] contact_request parse failed: %v", err)
		return
	}
	if n.requests == nil {
		log.Printf("[SERVICE] requests store not initialized")
		return
	}
	req := ContactRequest{
		ID:          m.ID,
		PeerID:      m.Sender,
		Name:        payload.Name,
		Ed25519Pub:  payload.Ed25519Pub,
		X25519Pub:   payload.X25519Pub,
		Signature:   payload.Signature,
		ReadEnabled: payload.ReadEnabled,
		Status:      RequestStatusPending,
	}
	if err := n.requests.Add(req); err != nil {
		log.Printf("[SERVICE] contact_request add failed: %v", err)
		return
	}
	n.RemoveTempContact(m.Sender)
	log.Printf("[SERVICE] contact_request saved from %s", m.Sender)

	if n.messageHook != nil {
		data, _ := json.Marshal(m)
		n.messageHook(string(data))
	}
}

// handleContactAccept — обрабатывает принятие нашего запроса.
func (n *Node) handleContactAccept(m Message) {
	if m.Version != MESSAGE_VERSION_E2E {
		log.Printf("[SERVICE] contact_accept without E2E, dropped")
		return
	}
	plaintext, err := n.decryptFromSender(m.Sender, m.Text)
	if err != nil {
		log.Printf("[SERVICE] contact_accept decrypt failed: %v", err)
		return
	}
	var payload struct {
		RequestID   string `json:"request_id"`
		Name        string `json:"name"`
		Ed25519Pub  string `json:"ed25519_pub"`
		X25519Pub   string `json:"x25519_pub"`
		Signature   string `json:"signature"`
		ReadEnabled bool   `json:"read_enabled"`
	}
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		log.Printf("[SERVICE] contact_accept parse failed: %v", err)
		return
	}
	if n.contacts == nil {
		log.Printf("[SERVICE] contacts store not initialized")
		return
	}
	if err := n.AddContact(m.Sender, payload.Ed25519Pub, payload.X25519Pub, payload.Signature, "", payload.Name, payload.ReadEnabled); err != nil {
		log.Printf("[SERVICE] contact_accept add_contact failed: %v", err)
		return
	}
	if n.contacts != nil {
		_ = n.contacts.SetConfirmed(m.Sender)
	}
	log.Printf("[SERVICE] contact_accept saved contact %s (ref=%s)", m.Sender, m.Ref)

	if n.messageHook != nil {
		data, _ := json.Marshal(m)
		n.messageHook(string(data))
	}
}

// handleContactReject — обрабатывает отклонение нашего запроса.
func (n *Node) handleContactReject(m Message) {
	log.Printf("[SERVICE] contact_reject ref=%s from=%s", m.Ref, m.Sender)
}

// handleContactHello — обрабатывает [CONTACT_HELLO] от A.
func (n *Node) handleContactHello(m Message) {
	if m.Sender == "" {
		log.Printf("[SERVICE] contact_hello: empty sender, dropped")
		return
	}
	if n.host == nil {
		log.Printf("[SERVICE] contact_hello: host not started")
		return
	}
	myID := n.host.ID().String()
	if m.Sender == myID {
		log.Printf("[SERVICE] contact_hello: self, dropped")
		return
	}
	if m.Text == "" {
		log.Printf("[SERVICE] contact_hello: empty payload from %s", m.Sender)
		return
	}

	var payload struct {
		PeerID     string `json:"peerID"`
		Ed25519Pub string `json:"ed25519_pub"`
		X25519Pub  string `json:"x25519_pub"`
		Signature  string `json:"signature"`
	}
	if err := json.Unmarshal([]byte(m.Text), &payload); err != nil {
		log.Printf("[SERVICE] contact_hello: parse failed from %s: %v", m.Sender, err)
		return
	}
	if payload.PeerID != m.Sender {
		log.Printf("[SERVICE] contact_hello: peerID mismatch (%s != %s), dropped", payload.PeerID, m.Sender)
		return
	}
	if payload.Ed25519Pub == "" || payload.X25519Pub == "" {
		log.Printf("[SERVICE] contact_hello: missing keys from %s", m.Sender)
		return
	}

	verified := false
	if payload.Signature != "" {
		if verifyContactSignature(payload.PeerID, payload.Ed25519Pub, payload.X25519Pub, payload.Signature) {
			verified = true
		}
	}

	if _, ok := n.GetContact(m.Sender); !ok {
		n.AddTempContact(Contact{
			PeerID:      payload.PeerID,
			Ed25519Pub:  payload.Ed25519Pub,
			X25519Pub:   payload.X25519Pub,
			Signature:   payload.Signature,
			Verified:    verified,
			Confirmed:   false,
			ReadEnabled: true,
		})
	}

	log.Printf("[SERVICE] contact_hello from %s — sending ack", m.Sender)
	go func(sender string) {
		if err := n.SendContactHelloAck(sender); err != nil {
			log.Printf("[SERVICE] contact_hello_ack send failed to %s: %v", sender, err)
		}
	}(m.Sender)
}

// handleContactHelloAck — обрабатывает [CONTACT_HELLO_ACK] от B.
func (n *Node) handleContactHelloAck(m Message) {
	if m.Sender == "" {
		log.Printf("[SERVICE] contact_hello_ack: empty sender, dropped")
		return
	}
	if m.Version != 0 {
		log.Printf("[SERVICE] contact_hello_ack: unexpected version %d, dropped", m.Version)
		return
	}
	plaintext := m.Text
	if plaintext == "" {
		log.Printf("[SERVICE] contact_hello_ack: empty payload from %s", m.Sender)
		return
	}
	var payload struct {
		PeerID     string `json:"peerID"`
		Ed25519Pub string `json:"ed25519_pub"`
		X25519Pub  string `json:"x25519_pub"`
		Signature  string `json:"signature"`
	}
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		log.Printf("[SERVICE] contact_hello_ack: parse failed from %s: %v", m.Sender, err)
		return
	}
	if payload.PeerID != m.Sender {
		log.Printf("[SERVICE] contact_hello_ack: peerID mismatch (%s != %s), dropped", payload.PeerID, m.Sender)
		return
	}
	log.Printf("[SERVICE] contact_hello_ack from %s (keys: ed25519=%v, x25519=%v)",
		m.Sender, payload.Ed25519Pub != "", payload.X25519Pub != "")

	if n.messageHook != nil {
		data, _ := json.Marshal(m)
		n.messageHook(string(data))
	}
}

// SendContactRequest — отправляет запрос на добавление в контакты.
func (n *Node) SendContactRequest(recipient, displayName string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if recipient == "" {
		return "", fmt.Errorf("recipient is required")
	}

	if displayName == "" {
		displayName = n.GetMyDisplayName()
	}
	readEnabled := n.myReadEnabled

	payload := map[string]interface{}{
		"name":         displayName,
		"ed25519_pub":  base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":   base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":    n.signMyX25519(),
		"read_enabled": readEnabled,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encrypted, err := n.encryptForRecipient(recipient, string(payloadJSON))
	if err != nil {
		return "", fmt.Errorf("encrypt failed: %w", err)
	}

	id := generateMsgID(fmt.Sprintf("req:%s:%d", recipient, time.Now().UnixNano()))
	n.processMessageInternal(encrypted, n.host.ID().String(), true, time.Time{}, id, recipient, MESSAGE_VERSION_E2E, "", TypeContactRequest, true, 0, "")

	return id, nil
}

// SendContactHello — отправляет [CONTACT_HELLO] получателю.
func (n *Node) SendContactHello(recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	payload := map[string]interface{}{
		"peerID":      n.host.ID().String(),
		"ed25519_pub": base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":  base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":   n.signMyX25519(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id := generateMsgID(fmt.Sprintf("hello:%s:%d", recipient, time.Now().UnixNano()))
	msg := Message{
		ID:        id,
		Text:      string(payloadJSON),
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeContactHello,
		Version:   0,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send hello failed: %w", err)
	}
	log.Printf("[REQUESTS] contact_hello sent to %s (id=%s)", recipient, id)
	return nil
}

// SendContactHelloAck — отправляет [CONTACT_HELLO_ACK] отправителю.
func (n *Node) SendContactHelloAck(recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	payload := map[string]interface{}{
		"peerID":      n.host.ID().String(),
		"ed25519_pub": base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":  base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":   n.signMyX25519(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id := generateMsgID(fmt.Sprintf("hello_ack:%s:%d", recipient, time.Now().UnixNano()))
	msg := Message{
		ID:        id,
		Text:      string(payloadJSON),
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeContactHelloAck,
		Version:   0,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send hello_ack failed: %w", err)
	}
	log.Printf("[REQUESTS] contact_hello_ack sent to %s (id=%s)", recipient, id)
	return nil
}

// SendContactAccept — отправляет принятие запроса на контакт.
func (n *Node) SendContactAccept(requestID, recipient string) error {
	return n.sendContactControl(TypeContactAccept, requestID, recipient)
}

// SendContactReject — отправляет отклонение запроса на контакт.
func (n *Node) SendContactReject(requestID, recipient string) error {
	return n.sendContactControl(TypeContactReject, requestID, recipient)
}

// sendContactControl — общая логика accept/reject.
func (n *Node) sendContactControl(msgType MessageType, requestID, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if requestID == "" || recipient == "" {
		return fmt.Errorf("requestID and recipient are required")
	}

	readEnabled := n.myReadEnabled
	payload := map[string]interface{}{
		"request_id":   requestID,
		"name":         n.GetMyDisplayName(),
		"ed25519_pub":  base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":   base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":    n.signMyX25519(),
		"read_enabled": readEnabled,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	encrypted, err := n.encryptForRecipient(recipient, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("encrypt failed: %w", err)
	}

	id := generateMsgID(fmt.Sprintf("%s:%s:%d", msgTypeString(msgType), requestID, time.Now().UnixNano()))
	n.processMessageInternal(encrypted, n.host.ID().String(), true, time.Time{}, id, recipient, MESSAGE_VERSION_E2E, "", msgType, true, 0, "")

	return nil
}

// AddContact — добавляет или обновляет контакт.
func (n *Node) AddContact(peerID, ed25519Pub, x25519Pub, signature, localName, remoteName string, readEnabled bool) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}

	if n.host != nil && peerID == n.host.ID().String() {
		return fmt.Errorf("cannot add self as contact")
	}

	if n.deleted != nil {
		if err := n.deleted.RemoveFromDeleted(peerID); err != nil {
			log.Printf("[DELETED] removeFromDeleted failed for %s: %v", peerID, err)
		}
	}

	verified := false

	if signature != "" {
		if ed25519Pub == "" || x25519Pub == "" {
			log.Printf("[CONTACT] incomplete keys for %s (ed25519=%v, x25519=%v) — contact saved as unverified",
				peerID, ed25519Pub != "", x25519Pub != "")
		} else if verifyContactSignature(peerID, ed25519Pub, x25519Pub, signature) {
			verified = true
			log.Printf("[CONTACT] signature verified for %s", peerID)
		} else {
			log.Printf("[CONTACT] signature invalid for %s — contact saved as unverified", peerID)
		}
	}

	return n.contacts.Add(Contact{
		PeerID:      peerID,
		Ed25519Pub:  ed25519Pub,
		X25519Pub:   x25519Pub,
		Signature:   signature,
		Verified:    verified,
		Name:        localName,
		RemoteName:  remoteName,
		ReadEnabled: readEnabled,
	})
}

// SetContactReadEnabled — устанавливает read_enabled для контакта.
func (n *Node) SetContactReadEnabled(peerID string, enabled bool) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}
	return n.contacts.UpdateReadEnabled(peerID, enabled)
}

// RenameContact — устанавливает локальное имя контакта (Name).
func (n *Node) RenameContact(peerID, localName string) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}
	return n.contacts.RenameContact(peerID, localName)
}

// RemoveContact — удаляет контакт у меня. У собеседника остаётся.
func (n *Node) RemoveContact(peerID string) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}

	_ = n.contacts.Remove(peerID)

	if n.deleted != nil {
		if err := n.deleted.Add(peerID); err != nil {
			log.Printf("[DELETED] add failed for %s: %v", peerID, err)
		}
	}

	all := n.memory.GetAll()
	for _, m := range all {
		if m.Sender == peerID || m.Recipient == peerID {
			n.memory.Remove(m.ID)
		}
	}

	n.scheduleSaveState()

	log.Printf("[CONTACTS] removed %s + messages purged + added to deleted", peerID)
	return nil
}

// getPeerReadEnabled — возвращает read_enabled контакта (из isotope_contacts.json).
func (n *Node) getPeerReadEnabled(peerID string) bool {
	if n.contacts == nil {
		return true
	}
	c, ok := n.contacts.Get(peerID)
	if !ok {
		return true
	}
	return c.ReadEnabled
}

// GetContacts — возвращает все контакты.
func (n *Node) GetContacts() []Contact {
	if n.contacts == nil {
		return []Contact{}
	}
	return n.contacts.GetAll()
}

// GetContact — возвращает контакт по PeerID.
func (n *Node) GetContact(peerID string) (Contact, bool) {
	if n.contacts == nil {
		return Contact{}, false
	}
	return n.contacts.Get(peerID)
}

// nameForPeer — возвращает имя отправителя по приоритету:
//   - свой peerID → MyDisplayName;
//   - контакт → Name → RemoteName;
//   - иначе — пусто (UI возьмёт короткий PeerID).
func (n *Node) nameForPeer(peerID string) string {
	if peerID == "" {
		return ""
	}
	if n.host != nil && peerID == n.host.ID().String() {
		if n.settingsStore != nil {
			return n.settingsStore.GetMyDisplayName()
		}
		return ""
	}
	if n.contacts != nil {
		if c, ok := n.contacts.Get(peerID); ok {
			if c.Name != "" {
				return c.Name
			}
			if c.RemoteName != "" {
				return c.RemoteName
			}
		}
	}
	return ""
}

// AddTempContact — добавляет временный контакт (только в памяти).
func (n *Node) AddTempContact(c Contact) {
	if c.PeerID == "" {
		return
	}
	n.tempContactsMu.Lock()
	defer n.tempContactsMu.Unlock()
	if n.tempContacts == nil {
		n.tempContacts = make(map[string]Contact)
	}
	n.tempContacts[c.PeerID] = c
	log.Printf("[CONTACTS] temp added %s (verified=%v)", c.PeerID, c.Verified)
}

// GetContactTemp — возвращает контакт по PeerID: сначала постоянный,
// потом временный.
func (n *Node) GetContactTemp(peerID string) (Contact, bool) {
	if c, ok := n.GetContact(peerID); ok {
		return c, true
	}
	n.tempContactsMu.Lock()
	defer n.tempContactsMu.Unlock()
	c, ok := n.tempContacts[peerID]
	return c, ok
}

// RemoveTempContact — удаляет временный контакт по PeerID.
func (n *Node) RemoveTempContact(peerID string) {
	n.tempContactsMu.Lock()
	defer n.tempContactsMu.Unlock()
	if _, ok := n.tempContacts[peerID]; ok {
		delete(n.tempContacts, peerID)
		log.Printf("[CONTACTS] temp removed %s", peerID)
	}
}

// GetRequests — возвращает все pending-запросы на контакт.
func (n *Node) GetRequests() []ContactRequest {
	if n.requests == nil {
		return []ContactRequest{}
	}
	return n.requests.GetPending()
}

// AcceptRequestByID — принимает входящий запрос по ID.
func (n *Node) AcceptRequestByID(id string) error {
	if n.requests == nil {
		return fmt.Errorf("requests store not initialized")
	}
	req, ok := n.requests.Get(id)
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if err := n.AddContact(req.PeerID, req.Ed25519Pub, req.X25519Pub, req.Signature, "", req.Name, req.ReadEnabled); err != nil {
		return fmt.Errorf("add contact failed: %w", err)
	}

	if n.contacts != nil {
		_ = n.contacts.SetConfirmed(req.PeerID)
	}

	if err := n.SendContactAccept(id, req.PeerID); err != nil {
		log.Printf("[REQUESTS] accept send failed: %v", err)
	}

	if err := n.requests.Remove(id); err != nil {
		log.Printf("[REQUESTS] remove failed: %v", err)
	}
	log.Printf("[REQUESTS] accepted %s (peer=%s)", id, req.PeerID)
	return nil
}

// RejectRequestByID — отклоняет входящий запрос по ID.
func (n *Node) RejectRequestByID(id string) error {
	if n.requests == nil {
		return fmt.Errorf("requests store not initialized")
	}
	req, ok := n.requests.Get(id)
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if err := n.SendContactReject(id, req.PeerID); err != nil {
		log.Printf("[REQUESTS] reject send failed: %v", err)
	}

	if err := n.requests.Remove(id); err != nil {
		log.Printf("[REQUESTS] remove failed: %v", err)
	}
	log.Printf("[REQUESTS] rejected %s (peer=%s)", id, req.PeerID)
	return nil
}
