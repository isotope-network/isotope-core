// node/sphinx_test.go
package core

import (
	"bytes"
	cryptorand "crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/nacl/box"
)

// newTestSphinxNode — минимальный Node для Sphinx-теста.
// Без host, без contacts. Только X25519-ключи и кэш compact_id.
func newTestSphinxNode(t *testing.T, pid peer.ID, known ...peer.ID) *Node {
	t.Helper()
	pub, priv, err := box.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	n := &Node{
		x25519Priv:    *priv,
		x25519Pub:     *pub,
		sphinxIDCache: make(map[[SphinxCompactIDSize]byte]peer.ID),
	}
	n.sphinxIDCache[sphinxCompactID(pid.String())] = pid
	for _, k := range known {
		n.sphinxIDCache[sphinxCompactID(k.String())] = k
	}
	return n
}

func testPeerID(s string) peer.ID {
	return peer.ID(s)
}

func TestSphinxRoundTrip(t *testing.T) {
	relay1ID := testPeerID("relay1")
	relay2ID := testPeerID("relay2")
	recipientID := testPeerID("recipient")

	relay1 := newTestSphinxNode(t, relay1ID, relay2ID)
	relay2 := newTestSphinxNode(t, relay2ID, recipientID)
	recipient := newTestSphinxNode(t, recipientID)

	ephPub, ephPriv, err := box.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("ephemeral keygen: %v", err)
	}

	var k1, k2, k3 [32]byte
	box.Precompute(&k1, &relay1.x25519Pub, ephPriv)
	box.Precompute(&k2, &relay2.x25519Pub, ephPriv)
	box.Precompute(&k3, &recipient.x25519Pub, ephPriv)

	route := &sphinxRoute{
		Relays:        []peer.ID{relay1ID, relay2ID},
		Recipient:     recipientID,
		Ephemeral:     *ephPub,
		EphemeralPriv: *ephPriv,
		Keys:          [][]byte{k1[:], k2[:], k3[:]},
	}

	payload := []byte(`{"id":"msg1","text":"hello sphinx"}`)
	packet, err := relay1.buildSphinxPacket(route, payload, SphinxPayloadTextSize)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	next1, rem1, err := relay1.processSphinxPacket(packet)
	if err != nil {
		t.Fatalf("relay1 process: %v", err)
	}
	if next1 != relay2ID {
		t.Fatalf("relay1 next_hop: got %q, want %q", next1, relay2ID)
	}

	next2, rem2, err := relay2.processSphinxPacket(rem1)
	if err != nil {
		t.Fatalf("relay2 process: %v", err)
	}
	if next2 != recipientID {
		t.Fatalf("relay2 next_hop: got %q, want %q", next2, recipientID)
	}

	plain, err := recipient.extractSphinxPayload(rem2)
	if err != nil {
		t.Fatalf("recipient extract: %v", err)
	}
	if len(plain) < len(payload) {
		t.Fatalf("payload too short: got %d, want >= %d", len(plain), len(payload))
	}
	if !bytes.Equal(plain[:len(payload)], payload) {
		t.Fatalf("payload mismatch: got %q, want %q", plain[:len(payload)], payload)
	}
}

func TestSphinxWrongMAC(t *testing.T) {
	relay1ID := testPeerID("relay1")
	relay2ID := testPeerID("relay2")
	recipientID := testPeerID("recipient")

	relay1 := newTestSphinxNode(t, relay1ID)

	ephPub, ephPriv, _ := box.GenerateKey(cryptorand.Reader)
	var k1, k2, k3 [32]byte
	box.Precompute(&k1, &relay1.x25519Pub, ephPriv)
	box.Precompute(&k2, &relay1.x25519Pub, ephPriv)
	box.Precompute(&k3, &relay1.x25519Pub, ephPriv)

	route := &sphinxRoute{
		Relays:        []peer.ID{relay1ID, relay2ID},
		Recipient:     recipientID,
		Ephemeral:     *ephPub,
		EphemeralPriv: *ephPriv,
		Keys:          [][]byte{k1[:], k2[:], k3[:]},
	}

	packet, err := relay1.buildSphinxPacket(route, []byte("x"), SphinxPayloadTextSize)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	packet[20] ^= 0xff

	_, _, err = relay1.processSphinxPacket(packet)
	if err == nil {
		t.Fatalf("expected MAC mismatch, got nil")
	}
}

func TestSphinxTamperedNextHop(t *testing.T) {
	relay1ID := testPeerID("relay1")
	relay2ID := testPeerID("relay2")
	recipientID := testPeerID("recipient")

	relay1 := newTestSphinxNode(t, relay1ID, relay2ID)

	ephPub, ephPriv, _ := box.GenerateKey(cryptorand.Reader)
	var k1, k2, k3 [32]byte
	box.Precompute(&k1, &relay1.x25519Pub, ephPriv)
	box.Precompute(&k2, &relay1.x25519Pub, ephPriv)
	box.Precompute(&k3, &relay1.x25519Pub, ephPriv)

	route := &sphinxRoute{
		Relays:        []peer.ID{relay1ID, relay2ID},
		Recipient:     recipientID,
		Ephemeral:     *ephPub,
		EphemeralPriv: *ephPriv,
		Keys:          [][]byte{k1[:], k2[:], k3[:]},
	}

	packet, err := relay1.buildSphinxPacket(route, []byte("x"), SphinxPayloadTextSize)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	packet[0] ^= 0xff

	_, _, err = relay1.processSphinxPacket(packet)
	if err == nil {
		t.Fatalf("expected unknown next_hop error, got nil")
	}
}

// node/sphinx_test.go
