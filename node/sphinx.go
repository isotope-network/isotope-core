// node/sphinx.go
package core

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/nacl/box"
)

const (
	SphinxHops             = 3
	SphinxCompactIDSize    = 16
	SphinxMACSize          = 32
	SphinxEphemeralPubSize = 32
	SphinxRoutingSize      = 152
	SphinxHopLayerSize     = SphinxCompactIDSize + SphinxMACSize + SphinxRoutingSize // 200
	SphinxHeaderSize       = SphinxHops * SphinxHopLayerSize                         // 600
	SphinxPayloadTextSize  = 4096
	SphinxPayloadFileSize  = 65536
	SphinxTimeout          = 5 * time.Second
)

// sphinxRoute — построенный маршрут [relay1, relay2, recipient].
type sphinxRoute struct {
	Relays        []peer.ID
	Recipient     peer.ID
	Ephemeral     [32]byte
	EphemeralPriv [32]byte
	Keys          [][]byte
}

// sphinxCompactID — sha256(peerID)[:16].
func sphinxCompactID(peerID string) [SphinxCompactIDSize]byte {
	h := sha256.Sum256([]byte(peerID))
	var out [SphinxCompactIDSize]byte
	copy(out[:], h[:SphinxCompactIDSize])
	return out
}

// sphinxCacheID — добавляет запись в кэш compact_id → peer.ID.
func (n *Node) sphinxCacheID(pid peer.ID) {
	cid := sphinxCompactID(pid.String())
	n.sphinxIDCacheMu.Lock()
	n.sphinxIDCache[cid] = pid
	n.sphinxIDCacheMu.Unlock()
}

// sphinxLookupID — находит peer.ID по compact_id. O(1) через кэш.
func (n *Node) sphinxLookupID(cid [SphinxCompactIDSize]byte) (peer.ID, bool) {
	n.sphinxIDCacheMu.Lock()
	pid, ok := n.sphinxIDCache[cid]
	n.sphinxIDCacheMu.Unlock()
	if ok {
		return pid, true
	}
	if n.contacts == nil {
		return "", false
	}
	for _, c := range n.contacts.GetAll() {
		if sphinxCompactID(c.PeerID) == cid {
			p, err := peer.Decode(c.PeerID)
			if err != nil {
				return "", false
			}
			n.sphinxCacheID(p)
			return p, true
		}
	}
	return "", false
}

// selectSphinxRelays — выбирает 2 relay по весу > 0.5 (Verified && Confirmed).
// recipient исключается: получатель не может быть relay.
// TODO: raise weight threshold to 0.7 after test (0.5 — для теста на новых контактах).
// Возвращает nil, если < 2.
func (n *Node) selectSphinxRelays(recipient peer.ID) []peer.ID {
	if n.contacts == nil {
		return nil
	}
	all := n.contacts.GetAll()
	var trusted []peer.ID
	for _, c := range all {
		if !c.Verified || !c.Confirmed {
			continue
		}
		if c.X25519Pub == "" {
			continue
		}
		if recipient != "" && c.PeerID == recipient.String() {
			continue
		}
		if n.getPeerWeight(c.PeerID) <= 0.5 {
			continue
		}
		pid, err := peer.Decode(c.PeerID)
		if err != nil {
			continue
		}
		trusted = append(trusted, pid)
		n.sphinxCacheID(pid)
	}
	if len(trusted) < 2 {
		return nil
	}
	perm := mathrand.Perm(len(trusted))
	return []peer.ID{trusted[perm[0]], trusted[perm[1]]}
}

// buildSphinxRoute — строит маршрут: 2 relay + получатель. Генерирует ephemeral.
// Для всех слоёв (relay'и + получатель) — DH(ephemeral_priv, X25519Pub).
// n.x25519Priv НЕ используется — relay не знает отправителя.
func (n *Node) buildSphinxRoute(recipient peer.ID) (*sphinxRoute, error) {
	relays := n.selectSphinxRelays(recipient)
	if len(relays) < 2 {
		return nil, fmt.Errorf("not enough trusted relays (< 2)")
	}

	ephPub, ephPriv, err := box.GenerateKey(cryptorand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ephemeral keygen failed: %w", err)
	}

	keys := make([][]byte, 0, SphinxHops)

	// relay'и: DH(ephPriv, relay.X25519Pub).
	for _, relay := range relays {
		c, ok := n.contacts.Get(relay.String())
		if !ok || c.X25519Pub == "" {
			return nil, fmt.Errorf("relay %s has no X25519Pub", relay)
		}
		pubBytes, err := base64.StdEncoding.DecodeString(c.X25519Pub)
		if err != nil || len(pubBytes) != 32 {
			return nil, fmt.Errorf("relay %s invalid X25519Pub", relay)
		}
		var pub [32]byte
		copy(pub[:], pubBytes)
		var shared [32]byte
		box.Precompute(&shared, &pub, ephPriv)
		keys = append(keys, shared[:])
	}

	// получатель: DH(ephPriv, recipient.X25519Pub).
	rc, ok := n.contacts.Get(recipient.String())
	if !ok || rc.X25519Pub == "" {
		return nil, fmt.Errorf("recipient %s has no X25519Pub", recipient)
	}
	rpubBytes, err := base64.StdEncoding.DecodeString(rc.X25519Pub)
	if err != nil || len(rpubBytes) != 32 {
		return nil, fmt.Errorf("recipient invalid X25519Pub")
	}
	var rpub [32]byte
	copy(rpub[:], rpubBytes)
	var rshared [32]byte
	box.Precompute(&rshared, &rpub, ephPriv)
	keys = append(keys, rshared[:])

	n.sphinxCacheID(recipient)
	for _, r := range relays {
		n.sphinxCacheID(r)
	}

	return &sphinxRoute{
		Relays:        relays,
		Recipient:     recipient,
		Ephemeral:     *ephPub,
		EphemeralPriv: *ephPriv,
		Keys:          keys,
	}, nil
}

// buildSphinxPacket — строит пакет. payloadSize — 4K или 64K.
// MAC считается по ciphertext (nonce || ct) — проверяется до decrypt.
func (n *Node) buildSphinxPacket(route *sphinxRoute, payload []byte, payloadSize int) ([]byte, error) {
	if len(route.Keys) != SphinxHops {
		return nil, fmt.Errorf("route keys: expected %d, got %d", SphinxHops, len(route.Keys))
	}
	if payloadSize != SphinxPayloadTextSize && payloadSize != SphinxPayloadFileSize {
		return nil, fmt.Errorf("invalid payloadSize: %d", payloadSize)
	}
	if len(payload) > payloadSize {
		return nil, fmt.Errorf("payload too large: %d > %d", len(payload), payloadSize)
	}

	innerRouting := make([]byte, SphinxRoutingSize)
	copy(innerRouting[0:32], route.Ephemeral[:])
	if _, err := io.ReadFull(cryptorand.Reader, innerRouting[32:]); err != nil {
		return nil, fmt.Errorf("routing padding failed: %w", err)
	}

	payloadPadded := make([]byte, payloadSize)
	copy(payloadPadded, payload)
	if len(payload) < payloadSize {
		if _, err := io.ReadFull(cryptorand.Reader, payloadPadded[len(payload):]); err != nil {
			return nil, fmt.Errorf("payload padding failed: %w", err)
		}
	}

	key := route.Keys[2]
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce failed: %w", err)
	}
	ct := gcm.Seal(nil, nonce, payloadPadded, nil)

	current := make([]byte, 0, SphinxRoutingSize+len(nonce)+len(ct))
	current = append(current, innerRouting...)
	current = append(current, nonce...)
	current = append(current, ct...)

	for i := len(route.Relays) - 1; i >= 0; i-- {
		key := route.Keys[i]

		var nextPeer peer.ID
		if i+1 < len(route.Relays) {
			nextPeer = route.Relays[i+1]
		} else {
			nextPeer = route.Recipient
		}
		nextCID := sphinxCompactID(nextPeer.String())

		routing := make([]byte, SphinxRoutingSize)
		copy(routing[0:32], route.Ephemeral[:])
		if _, err := io.ReadFull(cryptorand.Reader, routing[32:]); err != nil {
			return nil, fmt.Errorf("routing fill failed: %w", err)
		}

		block, _ := aes.NewCipher(key)
		gcm, _ := cipher.NewGCM(block)
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
			return nil, fmt.Errorf("nonce failed: %w", err)
		}
		ct := gcm.Seal(nil, nonce, current, nil)

		mac := hmac.New(sha256.New, key)
		mac.Write(nextCID[:])
		mac.Write(routing)
		mac.Write(nonce)
		mac.Write(ct)
		macSum := mac.Sum(nil)

		layer := make([]byte, 0, SphinxHopLayerSize+len(nonce)+len(ct))
		layer = append(layer, nextCID[:]...)
		layer = append(layer, macSum...)
		layer = append(layer, routing...)
		layer = append(layer, nonce...)
		layer = append(layer, ct...)
		current = layer
	}

	return current, nil
}

// processSphinxPacket — relay: снимает свой слой. MAC проверяется ДО decrypt.
func (n *Node) processSphinxPacket(packet []byte) (peer.ID, []byte, error) {
	if len(packet) < SphinxHopLayerSize+12 {
		return "", nil, fmt.Errorf("packet too short: %d", len(packet))
	}

	var cid [SphinxCompactIDSize]byte
	copy(cid[:], packet[0:SphinxCompactIDSize])
	nextPeer, ok := n.sphinxLookupID(cid)
	if !ok {
		return "", nil, fmt.Errorf("unknown next_hop compact_id")
	}

	var macSum [SphinxMACSize]byte
	copy(macSum[:], packet[SphinxCompactIDSize:SphinxCompactIDSize+SphinxMACSize])

	routing := packet[SphinxCompactIDSize+SphinxMACSize : SphinxHopLayerSize]
	var ephPub [32]byte
	copy(ephPub[:], routing[0:32])

	var shared [32]byte
	box.Precompute(&shared, &ephPub, &n.x25519Priv)
	key := shared[:]

	rest := packet[SphinxHopLayerSize:]
	nonce := rest[0:12]
	ct := rest[12:]

	mac := hmac.New(sha256.New, key)
	mac.Write(cid[:])
	mac.Write(routing)
	mac.Write(nonce)
	mac.Write(ct)
	if !hmac.Equal(mac.Sum(nil), macSum[:]) {
		return "", nil, fmt.Errorf("MAC mismatch")
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt failed: %w", err)
	}

	return nextPeer, plain, nil
}

// extractSphinxPayload — получатель: извлекает payload.
// Вход: [routing 152][nonce 12][ct] (финальный слой).
func (n *Node) extractSphinxPayload(packet []byte) ([]byte, error) {
	if len(packet) < SphinxRoutingSize+12 {
		return nil, fmt.Errorf("packet too short: %d", len(packet))
	}
	routing := packet[0:SphinxRoutingSize]
	var ephPub [32]byte
	copy(ephPub[:], routing[0:32])

	var shared [32]byte
	box.Precompute(&shared, &ephPub, &n.x25519Priv)
	key := shared[:]

	rest := packet[SphinxRoutingSize:]
	nonce := rest[0:12]
	ct := rest[12:]

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}
	return plain, nil
}

// sendViaSphinx — обёртка. Строит маршрут, пакет, шлёт relay1.
// Fallback на sendToRecipient при любой ошибке.
func (n *Node) sendViaSphinx(msg Message) bool {
	recipient, err := peer.Decode(msg.Recipient)
	if err != nil {
		log.Printf("[SPHINX] invalid recipient %q: %v", msg.Recipient, err)
		return false
	}

	route, err := n.buildSphinxRoute(recipient)
	if err != nil {
		log.Printf("[SPHINX] build route failed: %v", err)
		return false
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[SPHINX] marshal failed: %v", err)
		return false
	}

	payloadSize := SphinxPayloadTextSize
	switch msg.MediaType {
	case "file", "voice", "photo":
		payloadSize = SphinxPayloadFileSize
	}

	packet, err := n.buildSphinxPacket(route, payload, payloadSize)
	if err != nil {
		log.Printf("[SPHINX] build packet failed: %v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), SphinxTimeout)
	defer cancel()
	s, err := n.host.NewStream(ctx, route.Relays[0], protocolID)
	if err != nil {
		log.Printf("[SPHINX] NewStream to relay1 failed: %v", err)
		return false
	}
	defer s.Close()
	if _, err := fmt.Fprintf(s, "%s%s\n", SPHINX_PREFIX, base64.StdEncoding.EncodeToString(packet)); err != nil {
		log.Printf("[SPHINX] write failed: %v", err)
		return false
	}

	log.Printf("[SPHINX] sent via relay1=%s relay2=%s recipient=%s size=%d",
		route.Relays[0], route.Relays[1], route.Recipient, len(packet))
	return true
}

// node/sphinx.go
