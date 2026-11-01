package core

// D:\isotope\node\node_batch.go
// node_batch.go — batch текстов, dummy-трафик, обфускация, randomDelay.
// Вынесено из node.go.

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	mathrand "math/rand"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// batchBuffer — буфер текстов для одного получателя (v2.0.4 c).
type batchBuffer struct {
	Texts   []batchText
	FirstAt time.Time
}

// batchText — один текст в батче. ID — отправителя (для [DELIVERED]).
type batchText struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// estimateBatchJSONSize — приблизительный размер JSON для лимита 60 КБ.
func estimateBatchJSONSize(texts []batchText) int {
	size := 20
	for _, t := range texts {
		size += len(t.ID) + len(t.Text) + 30
	}
	return size
}

// addToBatch — добавляет текст в буфер получателя (v2.0.4 c).
func (n *Node) addToBatch(peerID, textID, text string) {
	if peerID == "" || textID == "" {
		return
	}
	n.batchBuffersMu.Lock()
	buf := n.batchBuffers[peerID]
	if buf == nil {
		buf = &batchBuffer{}
		n.batchBuffers[peerID] = buf
	}
	if len(buf.Texts) > 0 && estimateBatchJSONSize(buf.Texts)+len(textID)+len(text)+30 > 60*1024 {
		texts := buf.Texts
		buf.Texts = nil
		n.batchBuffersMu.Unlock()
		go n.flushBatch(peerID, texts)
		n.batchBuffersMu.Lock()
		n.batchBuffers[peerID] = &batchBuffer{Texts: []batchText{{ID: textID, Text: text}}}
		n.batchBuffersMu.Unlock()
		return
	}
	if buf.FirstAt.IsZero() {
		buf.FirstAt = time.Now()
	}
	buf.Texts = append(buf.Texts, batchText{ID: textID, Text: text})
	needFlush := len(buf.Texts) >= 10
	if needFlush {
		texts := buf.Texts
		buf.Texts = nil
		n.batchBuffersMu.Unlock()
		go n.flushBatch(peerID, texts)
		return
	}
	n.batchBuffersMu.Unlock()
}

// flushBatch — собирает JSON {"texts":[{"id":"...","text":"..."}]},
// шифрует E2E, отправляет одним Sphinx-пакетом (MediaType="batch").
func (n *Node) flushBatch(peerID string, texts []batchText) {
	if peerID == "" || len(texts) == 0 {
		return
	}
	batchJSON, err := json.Marshal(map[string]interface{}{"texts": texts})
	if err != nil {
		log.Printf("[BATCH] marshal failed: %v", err)
		return
	}
	encrypted, err := n.encryptForRecipient(peerID, string(batchJSON))
	if err != nil {
		log.Printf("[BATCH] encrypt failed: %v", err)
		return
	}
	id := generateMsgID(fmt.Sprintf("batch:%s:%d", peerID, time.Now().UnixNano()))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        encrypted,
		Sender:      n.host.ID().String(),
		Recipient:   peerID,
		Version:     MESSAGE_VERSION_E2E,
		Type:        TypeMessage,
		MediaType:   "batch",
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:       true,
		Weight:      0.5,
		ReadEnabled: &readEnabled,
	}
	n.unackedChunksMu.Lock()
	for _, t := range texts {
		n.unackedChunks[t.ID] = retryInfo{
			msgID:     t.ID,
			recipient: peerID,
			lastSent:  time.Now(),
		}
	}
	n.unackedChunksMu.Unlock()
	if n.sendViaSphinx(msg) {
		log.Printf("[BATCH] sent %d texts to %s (size=%d)", len(texts), peerID, len(batchJSON))
		return
	}
	log.Printf("[BATCH] sphinx failed for %s, fallback", peerID)
	data, _ := json.Marshal(msg)
	n.sendToRecipient(msg, data, false)
}

// flushAllBatches — тикер 1 сек: flush все непустые буферы.
func (n *Node) flushAllBatches() {
	n.batchBuffersMu.Lock()
	type item struct {
		peerID string
		texts  []batchText
	}
	var items []item
	for peerID, buf := range n.batchBuffers {
		if len(buf.Texts) == 0 {
			continue
		}
		if time.Since(buf.FirstAt) < 1*time.Second {
			continue
		}
		items = append(items, item{peerID: peerID, texts: buf.Texts})
		buf.Texts = nil
		buf.FirstAt = time.Time{}
	}
	n.batchBuffersMu.Unlock()
	for _, it := range items {
		go n.flushBatch(it.peerID, it.texts)
	}
}

// sendDummy — отправитель шлёт пустой Sphinx-пакет (v2.0.4 d, d.1.1).
func (n *Node) sendDummy() {
	if n.settingsStore == nil || !n.settingsStore.GetMaskingEnabled() {
		return
	}
	if !n.useOnion || n.isRelay || n.host == nil || n.contacts == nil {
		return
	}
	all := n.contacts.GetAll()
	if len(all) == 0 {
		return
	}
	c := all[mathrand.Intn(len(all))]
	if c.X25519Pub == "" || c.PeerID == "" {
		return
	}
	recipient, err := peer.Decode(c.PeerID)
	if err != nil {
		return
	}
	route, err := n.buildSphinxRoute(recipient)
	if err != nil {
		return
	}
	payload := []byte(generateRandomText(50, 200))
	packet, err := n.buildSphinxPacket(route, payload, false, true)
	if err != nil {
		return
	}
	delay := time.Duration(1+mathrand.Intn(60)) * time.Second
	go func() {
		time.Sleep(delay)
		n.sendSphinxPacketToRelay1(route, packet)
	}()
}

// SetDummyScreenOn — передаёт состояние экрана (v2.0.4 d.1.1).
func (n *Node) SetDummyScreenOn(screenOn bool) {
	n.mu.Lock()
	n.dummyScreenOn = screenOn
	n.mu.Unlock()
	log.Printf("[DUMMY] screenOn=%v", screenOn)
}

// dummyInterval — интервал dummy с учётом screenOn (v2.0.4 d.1.1).
func (n *Node) dummyInterval() time.Duration {
	n.mu.Lock()
	screenOn := n.dummyScreenOn
	n.mu.Unlock()
	base := 10 + mathrand.Intn(111)
	if !screenOn {
		base = base * 5
	}
	return time.Duration(base) * time.Second
}

// generateRandomText — случайный текст 50–200 символов (dummy, v2.0.4 d.1).
func generateRandomText(minLen, maxLen int) string {
	n := minLen + mathrand.Intn(maxLen-minLen+1)
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,!?"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[mathrand.Intn(len(chars))]
	}
	return string(b)
}

// sendSphinxPacketToRelay1 — отправка готового Sphinx-пакета relay1.
func (n *Node) sendSphinxPacketToRelay1(route *sphinxRoute, packet []byte) {
	if len(route.Relays) == 0 || n.host == nil {
		return
	}
	peerAddrs := n.host.Peerstore().Addrs(route.Relays[0])
	var direct []ma.Multiaddr
	if len(peerAddrs) == 0 {
		addrs, ferr := n.FindPeerByID(route.Relays[0].String())
		if ferr != nil {
			return
		}
		for _, a := range addrs {
			maAddr, perr := ma.NewMultiaddr(a)
			if perr != nil {
				continue
			}
			if strings.Contains(maAddr.String(), "/p2p-circuit") {
				continue
			}
			direct = append(direct, maAddr)
		}
		if len(direct) == 0 {
			return
		}
		n.host.Peerstore().AddAddrs(route.Relays[0], direct, peerstore.TempAddrTTL)
	} else {
		for _, a := range peerAddrs {
			if strings.Contains(a.String(), "/p2p-circuit") {
				continue
			}
			direct = append(direct, a)
		}
		if len(direct) == 0 {
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), SphinxTimeout)
	defer cancel()
	s, err := n.host.NewStream(ctx, route.Relays[0], protocolID)
	if err != nil {
		return
	}
	defer s.Close()
	s.SetWriteDeadline(time.Now().Add(30 * time.Second))
	fmt.Fprintf(s, "%s%s\n", SPHINX_PREFIX, base64.StdEncoding.EncodeToString(packet))
	log.Printf("[DUMMY] sent to %s (relay1=%s)", route.Recipient, route.Relays[0])
}
func (n *Node) getObfuscationKey() []byte {
	hash := sha256.Sum256([]byte(n.ethHash))
	return hash[:]
}

func (n *Node) obfuscate(msg string) string {
	key := n.getObfuscationKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(cryptorand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(msg), nil)
	return OBFUSCATION_PREFIX + base64.StdEncoding.EncodeToString(ciphertext)
}

func (n *Node) deobfuscate(data string) (string, bool) {
	if !strings.HasPrefix(data, OBFUSCATION_PREFIX) {
		return data, false
	}
	encoded := strings.TrimPrefix(data, OBFUSCATION_PREFIX)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return data, false
	}
	key := n.getObfuscationKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return data, false
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return data, false
	}
	return string(plaintext), true
}

func randomDelay(minMs, maxMs int) {
	n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(maxMs-minMs)))
	delay := time.Duration(minMs+int(n.Int64())) * time.Millisecond
	time.Sleep(delay)
}
