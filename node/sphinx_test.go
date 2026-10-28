// node/sphinx_test.go
package core

import (
	"bytes"
	cryptorand "crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/nacl/box"
)

func newTestSphinxNode(t *testing.T) *Node {
	t.Helper()
	pub, priv, err := box.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	return &Node{
		x25519Priv: *priv,
		x25519Pub:  *pub,
	}
}

func mustPeerID(t *testing.T, s string) peer.ID {
	t.Helper()
	id, err := peer.Decode(s)
	if err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return id
}

func TestSphinxRoundTrip(t *testing.T) {
	relay1 := newTestSphinxNode(t)
	relay2 := newTestSphinxNode(t)
	recipient := newTestSphinxNode(t)

	relay1ID := mustPeerID(t, "QmY7MefUnu3GG2VLEUQNCQXTMiGTLhB6HRR3YpFHUXWHkz")
	relay2ID := mustPeerID(t, "QmYJmBx8azsjij92qePsfGrUA4Md81NoavW9HsnY77iECE")
	recipientID := mustPeerID(t, "QmSsz5Kd8morv19LVCk8L4FHdGZnFyMJGh5BD9k1rqMGpT")

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
	packet, err := relay1.buildSphinxPacket(route, payload, true, false)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	next1, mixing1, _, rem1, err := relay1.processSphinxPacket(packet)
	if err != nil {
		t.Fatalf("relay1 process: %v", err)
	}
	if next1 != relay2ID {
		t.Fatalf("relay1 next_hop: got %q, want %q", next1, relay2ID)
	}
	if !mixing1 {
		t.Fatalf("relay1 mixing: got false, want true")
	}

	next2, mixing2, _, rem2, err := relay2.processSphinxPacket(rem1)
	if err != nil {
		t.Fatalf("relay2 process: %v", err)
	}
	if next2 != recipientID {
		t.Fatalf("relay2 next_hop: got %q, want %q", next2, recipientID)
	}
	if !mixing2 {
		t.Fatalf("relay2 mixing: got false, want true")
	}

	plain, err := recipient.extractSphinxPayload(rem2)
	if err != nil {
		t.Fatalf("recipient extract: %v", err)
	}
	if !bytes.Equal(plain, payload) {
		t.Fatalf("payload mismatch: got %q, want %q", plain, payload)
	}
}

func TestSphinxNoMixingFlag(t *testing.T) {
	relay1 := newTestSphinxNode(t)
	relay2 := newTestSphinxNode(t)
	recipient := newTestSphinxNode(t)

	relay1ID := mustPeerID(t, "QmY7MefUnu3GG2VLEUQNCQXTMiGTLhB6HRR3YpFHUXWHkz")
	relay2ID := mustPeerID(t, "QmYJmBx8azsjij92qePsfGrUA4Md81NoavW9HsnY77iECE")
	recipientID := mustPeerID(t, "QmSsz5Kd8morv19LVCk8L4FHdGZnFyMJGh5BD9k1rqMGpT")

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

	packet, err := relay1.buildSphinxPacket(route, []byte("x"), false, false)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	_, mixing, _, _, err := relay1.processSphinxPacket(packet)
	if err != nil {
		t.Fatalf("relay1 process: %v", err)
	}
	if mixing {
		t.Fatalf("mixing flag: got true, want false")
	}
}

func TestSphinxWrongMAC(t *testing.T) {
	relay1 := newTestSphinxNode(t)
	relay2 := newTestSphinxNode(t)
	recipient := newTestSphinxNode(t)

	relay1ID := mustPeerID(t, "QmY7MefUnu3GG2VLEUQNCQXTMiGTLhB6HRR3YpFHUXWHkz")
	relay2ID := mustPeerID(t, "QmYJmBx8azsjij92qePsfGrUA4Md81NoavW9HsnY77iECE")
	recipientID := mustPeerID(t, "QmSsz5Kd8morv19LVCk8L4FHdGZnFyMJGh5BD9k1rqMGpT")

	ephPub, ephPriv, _ := box.GenerateKey(cryptorand.Reader)
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

	packet, err := relay1.buildSphinxPacket(route, []byte("x"), false, false)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	// Портим MAC (offset 65..97).
	packet[70] ^= 0xff

	_, _, _, _, err = relay1.processSphinxPacket(packet)
	if err == nil {
		t.Fatalf("expected MAC mismatch, got nil")
	}
}

func TestSphinxTamperedNextHop(t *testing.T) {
	relay1 := newTestSphinxNode(t)
	relay2 := newTestSphinxNode(t)
	recipient := newTestSphinxNode(t)

	relay1ID := mustPeerID(t, "QmY7MefUnu3GG2VLEUQNCQXTMiGTLhB6HRR3YpFHUXWHkz")
	relay2ID := mustPeerID(t, "QmYJmBx8azsjij92qePsfGrUA4Md81NoavW9HsnY77iECE")
	recipientID := mustPeerID(t, "QmSsz5Kd8morv19LVCk8L4FHdGZnFyMJGh5BD9k1rqMGpT")

	ephPub, ephPriv, _ := box.GenerateKey(cryptorand.Reader)
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

	packet, err := relay1.buildSphinxPacket(route, []byte("x"), false, false)
	if err != nil {
		t.Fatalf("build packet: %v", err)
	}

	// Портим next_hop len (первый байт) — невалидно.
	packet[0] = 0xff

	_, _, _, _, err = relay1.processSphinxPacket(packet)
	if err == nil {
		t.Fatalf("expected invalid next_hop error, got nil")
	}
}

// node/sphinx_test.go
