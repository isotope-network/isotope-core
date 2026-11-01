package core

// D:\isotope\node\node_e2e.go
// node_e2e.go — E2E-ключи и шифрование (вынесено из node.go).
// Ed25519 (подпись) + X25519 (шифрование). QR v1. Подпись peerID||x25519_pub.

import (
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// ============================================================
// E2E — два ключа: Ed25519 (подпись) + X25519 (шифрование).
// Этап 4.3.1. Публичные ключи не хранятся — вычисляются из приватных.
// ============================================================

// loadOrGenerateE2EKeys — загружает / генерирует Ed25519 и X25519 ключи.
func (n *Node) loadOrGenerateE2EKeys() {
	if n.stateFile == "" {
		log.Printf("[KEY] stateFile not set, E2E keys skipped")
		return
	}
	if n.ed25519KeyFile == "" {
		n.ed25519KeyFile = n.stateFile + ".ed25519.key"
	}
	if n.x25519KeyFile == "" {
		n.x25519KeyFile = n.stateFile + ".x25519.key"
	}

	if err := n.loadOrGenerateEd25519Key(); err != nil {
		log.Printf("[KEY] ed25519 failed: %v", err)
	}
	if err := n.loadOrGenerateX25519Key(); err != nil {
		log.Printf("[KEY] x25519 failed: %v", err)
	}
}

// loadOrGenerateEd25519Key — загружает или генерирует Ed25519-ключ (подпись).
func (n *Node) loadOrGenerateEd25519Key() error {
	if data, err := os.ReadFile(n.ed25519KeyFile); err == nil {
		if len(data) == ed25519.PrivateKeySize {
			n.ed25519Priv = ed25519.PrivateKey(data)
			n.ed25519Pub = n.ed25519Priv.Public().(ed25519.PublicKey)
			log.Printf("[KEY] loaded ed25519 (pub=%s...)",
				truncate(base64.StdEncoding.EncodeToString(n.ed25519Pub), 16))
			return nil
		}
		log.Printf("[KEY] regenerated ed25519 key (wrong size: %d)", len(data))
	}

	pub, priv, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		return fmt.Errorf("ed25519 keygen failed: %w", err)
	}
	n.ed25519Priv = priv
	n.ed25519Pub = pub

	if err := os.WriteFile(n.ed25519KeyFile, priv, 0600); err != nil {
		return fmt.Errorf("ed25519 save failed: %w", err)
	}

	log.Printf("[KEY] generated ed25519 (pub=%s...)",
		truncate(base64.StdEncoding.EncodeToString(pub), 16))
	return nil
}

// loadOrGenerateX25519Key — загружает или генерирует X25519-ключ (шифрование).
func (n *Node) loadOrGenerateX25519Key() error {
	if data, err := os.ReadFile(n.x25519KeyFile); err == nil {
		if len(data) == 32 {
			copy(n.x25519Priv[:], data)
			pub, err := deriveX25519Public(n.x25519Priv)
			if err != nil {
				log.Printf("[KEY] x25519 derive failed: %v", err)
			} else {
				n.x25519Pub = pub
			}
			log.Printf("[KEY] loaded x25519 (pub=%s...)",
				truncate(base64.StdEncoding.EncodeToString(n.x25519Pub[:]), 16))
			return nil
		}
		log.Printf("[KEY] regenerated x25519 key (wrong size: %d)", len(data))
	}

	pub, priv, err := box.GenerateKey(cryptorand.Reader)
	if err != nil {
		return fmt.Errorf("x25519 keygen failed: %w", err)
	}
	n.x25519Priv = *priv
	n.x25519Pub = *pub

	if err := os.WriteFile(n.x25519KeyFile, n.x25519Priv[:], 0600); err != nil {
		return fmt.Errorf("x25519 save failed: %w", err)
	}

	log.Printf("[KEY] generated x25519 (pub=%s...)",
		truncate(base64.StdEncoding.EncodeToString(n.x25519Pub[:]), 16))
	return nil
}

// deriveX25519Public — вычисляет публичный ключ из приватного X25519.
func deriveX25519Public(priv [32]byte) ([32]byte, error) {
	pubBytes, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return [32]byte{}, err
	}
	var result [32]byte
	copy(result[:], pubBytes)
	return result, nil
}

// GetEd25519PublicKey — возвращает Ed25519-публичный ключ в base64.
func (n *Node) GetEd25519PublicKey() string {
	if len(n.ed25519Pub) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(n.ed25519Pub)
}

// GetX25519PublicKey — возвращает X25519-публичный ключ в base64.
func (n *Node) GetX25519PublicKey() string {
	return base64.StdEncoding.EncodeToString(n.x25519Pub[:])
}

// signMyX25519 — подпись peerID || x25519_pub приватным Ed25519-ключом.
// Возвращает base64-подпись или "" при ошибке / отсутствии ключей.
func (n *Node) signMyX25519() string {
	if n.host == nil || len(n.ed25519Priv) == 0 {
		return ""
	}
	peerID := n.host.ID().String()
	msg := make([]byte, 0, len(peerID)+32)
	msg = append(msg, []byte(peerID)...)
	msg = append(msg, n.x25519Pub[:]...)

	sig := ed25519.Sign(n.ed25519Priv, msg)
	return base64.StdEncoding.EncodeToString(sig)
}

// verifyContactSignature — проверяет подпись контакта.
// Подпись — над peerID || x25519_pub.
func verifyContactSignature(peerID, ed25519PubB64, x25519PubB64, signatureB64 string) bool {
	if peerID == "" || ed25519PubB64 == "" || x25519PubB64 == "" || signatureB64 == "" {
		return false
	}

	edPub, err := base64.StdEncoding.DecodeString(ed25519PubB64)
	if err != nil || len(edPub) != ed25519.PublicKeySize {
		return false
	}
	xPub, err := base64.StdEncoding.DecodeString(x25519PubB64)
	if err != nil || len(xPub) != 32 {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}

	msg := make([]byte, 0, len(peerID)+32)
	msg = append(msg, []byte(peerID)...)
	msg = append(msg, xPub...)

	return ed25519.Verify(ed25519.PublicKey(edPub), msg, sig)
}

// GetMyQRData — возвращает JSON для QR-кода версии 1.
func (n *Node) GetMyQRData() string {
	if n.host == nil {
		return ""
	}
	readEnabled := n.myReadEnabled
	displayName := n.GetMyDisplayName()
	data, err := json.Marshal(qrDataV1{
		V:           E2E_VERSION,
		PeerID:      n.host.ID().String(),
		Ed25519Pub:  base64.StdEncoding.EncodeToString(n.ed25519Pub),
		X25519Pub:   base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		Signature:   n.signMyX25519(),
		DisplayName: displayName,
		ReadEnabled: &readEnabled,
	})
	if err != nil {
		log.Printf("[KEY] QR marshal failed: %v", err)
		return ""
	}
	return string(data)
}

// encryptForRecipient — шифрует text для получателя.
// recipient — PeerID получателя. Требуется контакт с x25519_pub.
// Возвращает payload: base64(nonce(24) || ciphertext).
func (n *Node) encryptForRecipient(recipient, text string) (string, error) {
	if n.contacts == nil {
		return "", fmt.Errorf("contacts store not initialized")
	}

	contact, ok := n.contacts.Get(recipient)
	if !ok {
		return "", fmt.Errorf("contact not found: %s", recipient)
	}
	if contact.X25519Pub == "" {
		return "", fmt.Errorf("contact has no E2E key: %s", recipient)
	}

	recipientPubBytes, err := base64.StdEncoding.DecodeString(contact.X25519Pub)
	if err != nil || len(recipientPubBytes) != 32 {
		return "", fmt.Errorf("invalid x25519_pub for %s", recipient)
	}
	var recipientPub [32]byte
	copy(recipientPub[:], recipientPubBytes)

	var nonce [24]byte
	if _, err := cryptorand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext := box.Seal(nil, []byte(text), &nonce, &recipientPub, &n.x25519Priv)
	payload := append(nonce[:], ciphertext...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

// encryptBytesForRecipient — шифрует raw bytes для получателя.
func (n *Node) encryptBytesForRecipient(recipient string, data []byte) (string, error) {
	if n.contacts == nil {
		return "", fmt.Errorf("contacts store not initialized")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}
	contact, ok := n.contacts.Get(recipient)
	if !ok {
		return "", fmt.Errorf("contact not found: %s", recipient)
	}
	if contact.X25519Pub == "" {
		return "", fmt.Errorf("contact has no E2E key: %s", recipient)
	}
	recipientPubBytes, err := base64.StdEncoding.DecodeString(contact.X25519Pub)
	if err != nil || len(recipientPubBytes) != 32 {
		return "", fmt.Errorf("invalid x25519_pub for %s", recipient)
	}
	var recipientPub [32]byte
	copy(recipientPub[:], recipientPubBytes)
	var nonce [24]byte
	if _, err := cryptorand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}
	ciphertext := box.Seal(nil, data, &nonce, &recipientPub, &n.x25519Priv)
	payload := append(nonce[:], ciphertext...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

// decryptBytesFromSender — расшифровывает payload в raw bytes.
func (n *Node) decryptBytesFromSender(sender, payload string) ([]byte, error) {
	if n.contacts == nil {
		return nil, fmt.Errorf("contacts store not initialized")
	}
	if payload == "" {
		return nil, fmt.Errorf("empty payload from %s", sender)
	}
	contact, ok := n.GetContactTemp(sender)
	if !ok {
		return nil, fmt.Errorf("contact not found: %s", sender)
	}
	if contact.X25519Pub == "" {
		return nil, fmt.Errorf("contact has no E2E key: %s", sender)
	}
	senderPubBytes, err := base64.StdEncoding.DecodeString(contact.X25519Pub)
	if err != nil || len(senderPubBytes) != 32 {
		return nil, fmt.Errorf("invalid x25519_pub for %s", sender)
	}
	var senderPub [32]byte
	copy(senderPub[:], senderPubBytes)
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(raw) < 24 {
		return nil, fmt.Errorf("invalid payload from %s", sender)
	}
	var nonce [24]byte
	copy(nonce[:], raw[:24])
	ciphertext := raw[24:]
	plaintext, ok := box.Open(nil, ciphertext, &nonce, &senderPub, &n.x25519Priv)
	if !ok {
		log.Printf("[E2E] decrypt failed for %s", sender)
		return nil, fmt.Errorf("decryption failed for %s", sender)
	}
	return plaintext, nil
}

// decryptFromSender — расшифровывает payload от отправителя.
// payload — base64(nonce(24) || ciphertext).
func (n *Node) decryptFromSender(sender, payload string) (string, error) {
	if n.contacts == nil {
		return "", fmt.Errorf("contacts store not initialized")
	}

	contact, ok := n.GetContactTemp(sender)
	if !ok {
		return "", fmt.Errorf("contact not found: %s", sender)
	}
	if contact.X25519Pub == "" {
		return "", fmt.Errorf("contact has no E2E key: %s", sender)
	}

	senderPubBytes, err := base64.StdEncoding.DecodeString(contact.X25519Pub)
	if err != nil || len(senderPubBytes) != 32 {
		return "", fmt.Errorf("invalid x25519_pub for %s", sender)
	}
	var senderPub [32]byte
	copy(senderPub[:], senderPubBytes)

	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(raw) < 24 {
		return "", fmt.Errorf("invalid payload from %s", sender)
	}

	var nonce [24]byte
	copy(nonce[:], raw[:24])
	ciphertext := raw[24:]

	plaintext, ok := box.Open(nil, ciphertext, &nonce, &senderPub, &n.x25519Priv)
	if !ok {
		return "", fmt.Errorf("decryption failed for %s", sender)
	}
	return string(plaintext), nil
}
