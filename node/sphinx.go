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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	mathrand "math/rand"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/crypto/nacl/box"
)

const (
	SphinxHops              = 3
	SphinxNextPeerIDLenSize = 1
	SphinxNextPeerIDSize    = 64
	SphinxMACSize           = 32
	SphinxFlagsSize         = 1
	SphinxRoutingSize       = 152
	SphinxHopLayerSize      = SphinxNextPeerIDLenSize + SphinxNextPeerIDSize + SphinxMACSize + SphinxFlagsSize + SphinxRoutingSize // 250
	SphinxHeaderSize        = SphinxHops * SphinxHopLayerSize                                                                      // 750
	SphinxPayloadLenSize    = 4
	SphinxPayloadTextSize   = 4096
	SphinxPayloadFileSize   = 131072
	SphinxTimeout           = 15 * time.Second

	// SphinxFlagMixing — бит 0 флагов слоя: «relay, спи».
	SphinxFlagMixing uint8 = 1 << 0
)

// sphinxRoute — построенный маршрут [relay1, relay2, recipient].
type sphinxRoute struct {
	Relays        []peer.ID
	Recipient     peer.ID
	Ephemeral     [32]byte
	EphemeralPriv [32]byte
	Keys          [][]byte
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
		if n.getPeerWeight(c.PeerID) < 0.5 {
			continue
		}
		pid, err := peer.Decode(c.PeerID)
		if err != nil {
			continue
		}
		trusted = append(trusted, pid)
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
// Payload: [length 4 байта BE][JSON][random padding].
// mixing — флаг в каждом слое (v2.0.2): relay читает и спит.
func (n *Node) buildSphinxPacket(route *sphinxRoute, payload []byte, payloadSize int, mixing bool) ([]byte, error) {
	if len(route.Keys) != SphinxHops {
		return nil, fmt.Errorf("route keys: expected %d, got %d", SphinxHops, len(route.Keys))
	}
	if payloadSize != SphinxPayloadTextSize && payloadSize != SphinxPayloadFileSize {
		return nil, fmt.Errorf("invalid payloadSize: %d", payloadSize)
	}
	if len(payload) > payloadSize-SphinxPayloadLenSize {
		return nil, fmt.Errorf("payload too large: %d > %d", len(payload), payloadSize-SphinxPayloadLenSize)
	}

	// Финальный слой: routing + nonce + ct.
	innerRouting := make([]byte, SphinxRoutingSize)
	copy(innerRouting[0:32], route.Ephemeral[:])
	if _, err := io.ReadFull(cryptorand.Reader, innerRouting[32:]); err != nil {
		return nil, fmt.Errorf("routing padding failed: %w", err)
	}

	// Payload с длиной: [length 4][JSON][padding].
	payloadPadded := make([]byte, payloadSize)
	binary.BigEndian.PutUint32(payloadPadded[0:4], uint32(len(payload)))
	copy(payloadPadded[4:], payload)
	if len(payload)+SphinxPayloadLenSize < payloadSize {
		if _, err := io.ReadFull(cryptorand.Reader, payloadPadded[len(payload)+SphinxPayloadLenSize:]); err != nil {
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
		nextPeerID := nextPeer.String()
		if len(nextPeerID) > SphinxNextPeerIDSize {
			return nil, fmt.Errorf("next_hop peerID too long: %d", len(nextPeerID))
		}

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

		// layer: [len 1][peerID 64][MAC 32][flags 1][routing 152][nonce][ct].
		var flags uint8
		if mixing {
			flags |= SphinxFlagMixing
		}
		flagsBuf := []byte{flags}

		layer := make([]byte, 0, SphinxHopLayerSize+len(nonce)+len(ct))
		layer = append(layer, byte(len(nextPeerID)))
		peerIDBuf := make([]byte, SphinxNextPeerIDSize)
		copy(peerIDBuf, nextPeerID)
		layer = append(layer, peerIDBuf...)

		mac := hmac.New(sha256.New, key)
		mac.Write(layer[0 : SphinxNextPeerIDLenSize+SphinxNextPeerIDSize])
		mac.Write(flagsBuf)
		mac.Write(routing)
		mac.Write(nonce)
		mac.Write(ct)
		macSum := mac.Sum(nil)
		layer = append(layer, macSum...)
		layer = append(layer, flagsBuf...)
		layer = append(layer, routing...)
		layer = append(layer, nonce...)
		layer = append(layer, ct...)
		current = layer
	}

	return current, nil
}

// processSphinxPacket — relay: снимает свой слой. MAC проверяется ДО decrypt.
// next_hop — полный peer.ID (base58) в layer. Relay НЕ ищет в контактах.
// Возвращает mixing (бит 0 flags) — свойство сообщения, не настройка relay.
func (n *Node) processSphinxPacket(packet []byte) (peer.ID, bool, []byte, error) {
	if len(packet) < SphinxHopLayerSize+12 {
		return "", false, nil, fmt.Errorf("packet too short: %d", len(packet))
	}

	// layer: [len 1][peerID 64][MAC 32][flags 1][routing 152][nonce][ct].
	nextLen := int(packet[0])
	if nextLen <= 0 || nextLen > SphinxNextPeerIDSize {
		return "", false, nil, fmt.Errorf("invalid next_hop len: %d", nextLen)
	}
	nextPeerIDStr := string(packet[SphinxNextPeerIDLenSize : SphinxNextPeerIDLenSize+nextLen])
	nextPeer, err := peer.Decode(nextPeerIDStr)
	if err != nil {
		return "", false, nil, fmt.Errorf("invalid next_hop peerID: %w", err)
	}
	log.Printf("[SPHINX] process next=%s", nextPeer)

	macOff := SphinxNextPeerIDLenSize + SphinxNextPeerIDSize
	macSum := packet[macOff : macOff+SphinxMACSize]
	flagsOff := macOff + SphinxMACSize
	flags := packet[flagsOff]
	mixing := flags&SphinxFlagMixing != 0
	routing := packet[flagsOff+SphinxFlagsSize : SphinxHopLayerSize]

	var ephPub [32]byte
	copy(ephPub[:], routing[0:32])
	log.Printf("[SPHINX] process eph_pub=%x", ephPub[:8])

	var shared [32]byte
	box.Precompute(&shared, &ephPub, &n.x25519Priv)
	key := shared[:]
	log.Printf("[SPHINX] process DH=%x", key[:8])

	rest := packet[SphinxHopLayerSize:]
	nonce := rest[0:12]
	ct := rest[12:]

	mac := hmac.New(sha256.New, key)
	mac.Write(packet[0:macOff])
	mac.Write(packet[flagsOff : flagsOff+SphinxFlagsSize])
	mac.Write(routing)
	mac.Write(nonce)
	mac.Write(ct)
	if !hmac.Equal(mac.Sum(nil), macSum) {
		log.Printf("[SPHINX] process MAC mismatch")
		return "", false, nil, fmt.Errorf("MAC mismatch")
	}
	log.Printf("[SPHINX] process MAC ok mixing=%v", mixing)

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		log.Printf("[SPHINX] process decrypt failed: %v", err)
		return "", false, nil, fmt.Errorf("decrypt failed: %w", err)
	}
	log.Printf("[SPHINX] process decrypt ok len=%d", len(plain))

	return nextPeer, mixing, plain, nil
}

// extractSphinxPayload — получатель: извлекает payload.
// Вход: [routing 152][nonce 12][ct] (финальный слой).
// Payload: [length 4 BE][JSON][padding]. Возвращает чистый JSON.
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
	if len(plain) < SphinxPayloadLenSize {
		return nil, fmt.Errorf("payload too short: %d", len(plain))
	}
	length := int(binary.BigEndian.Uint32(plain[0:4]))
	if length < 0 || SphinxPayloadLenSize+length > len(plain) {
		return nil, fmt.Errorf("invalid payload length: %d", length)
	}
	return plain[SphinxPayloadLenSize : SphinxPayloadLenSize+length], nil
}

// sendViaSphinx — обёртка. Строит маршрут, пакет, шлёт relay1.
// Только прямой (Wi-Fi) путь. Circuit (через VPS) — не Sphinx.
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

	packet, err := n.buildSphinxPacket(route, payload, payloadSize, msg.MixingEnabled)
	if err != nil {
		log.Printf("[SPHINX] build packet failed: %v", err)
		return false
	}

	// Только прямой (Wi-Fi) путь. Circuit (через VPS) — не Sphinx.
	peerAddrs := n.host.Peerstore().Addrs(route.Relays[0])
	if len(peerAddrs) == 0 {
		if addrs, ferr := n.FindPeerByID(route.Relays[0].String()); ferr == nil {
			var direct []ma.Multiaddr
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
				log.Printf("[SPHINX] no direct addrs for relay1 %s, fallback", route.Relays[0])
				return false
			}
			n.host.Peerstore().AddAddrs(route.Relays[0], direct, peerstore.TempAddrTTL)
			log.Printf("[SPHINX] relay1=%s direct_addrs=%d", route.Relays[0], len(direct))
		} else {
			log.Printf("[SPHINX] FindPeerByID(%s) failed: %v", route.Relays[0], ferr)
			return false
		}
	} else {
		// В peerstore есть — проверить, что не circuit.
		var direct []ma.Multiaddr
		for _, a := range peerAddrs {
			if strings.Contains(a.String(), "/p2p-circuit") {
				continue
			}
			direct = append(direct, a)
		}
		if len(direct) == 0 {
			log.Printf("[SPHINX] no direct addrs in peerstore for relay1 %s, fallback", route.Relays[0])
			return false
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), SphinxTimeout)
	defer cancel()
	s, err := n.host.NewStream(ctx, route.Relays[0], protocolID)
	if err != nil {
		log.Printf("[SPHINX] NewStream to relay1 failed: %v", err)
		return false
	}
	defer s.Close()
	// Write deadline (v2.0.3): 128 КБ write — дольше 15 сек (Samsung нагрузка).
	// 30 сек — не висит бесконечно.
	s.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if _, err := fmt.Fprintf(s, "%s%s\n", SPHINX_PREFIX, base64.StdEncoding.EncodeToString(packet)); err != nil {
		log.Printf("[SPHINX] write failed: %v", err)
		return false
	}

	log.Printf("[SPHINX] sent via relay1=%s relay2=%s recipient=%s size=%d",
		route.Relays[0], route.Relays[1], route.Recipient, len(packet))
	return true
}

// sleepExponential — задержка с экспоненциальным распределением (mixing).
// mean — среднее, max — потолок. math.Log(1-u) — обратное преобразование.
// u ∈ (0, 1), чтобы log не ушёл в -Inf.
func sleepExponential(mean, max time.Duration) {
	u := mathrand.Float64()
	if u >= 1.0 {
		u = 0.999999
	}
	// Обратное преобразование exponential: -mean * ln(1-u).
	delay := time.Duration(float64(mean) * -math.Log(1.0-u))
	if delay > max {
		delay = max
	}
	if delay < 0 {
		delay = 0
	}
	log.Printf("[SPHINX] mixing: delay=%s", delay)
	time.Sleep(delay)
}

// node/sphinx.go
