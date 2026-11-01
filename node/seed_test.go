package core

// D:\isotope\node\seed_test.go
// seed_test.go — юнит-тесты для seed.go. Без телефонов, без сети.
// go test ./...

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/curve25519"
)

func TestGenerateSeedPhrase(t *testing.T) {
	for _, words := range []int{12, 24} {
		phrase, err := GenerateSeedPhrase(words)
		if err != nil {
			t.Fatalf("GenerateSeedPhrase(%d): %v", words, err)
		}
		if err := ValidateSeedPhrase(phrase); err != nil {
			t.Fatalf("ValidateSeedPhrase(%d words): %v", words, err)
		}
	}
}

func TestGenerateSeedPhrase_BadWords(t *testing.T) {
	if _, err := GenerateSeedPhrase(15); err == nil {
		t.Fatal("GenerateSeedPhrase(15) should fail")
	}
}

func TestValidateSeedPhrase_Empty(t *testing.T) {
	if err := ValidateSeedPhrase(""); err == nil {
		t.Fatal("empty phrase should fail")
	}
}

func TestValidateSeedPhrase_Bad(t *testing.T) {
	if err := ValidateSeedPhrase("not a valid mnemonic phrase at all"); err == nil {
		t.Fatal("invalid phrase should fail")
	}
}

func TestSeedToEd25519_Deterministic(t *testing.T) {
	phrase, err := GenerateSeedPhrase(12)
	if err != nil {
		t.Fatalf("GenerateSeedPhrase: %v", err)
	}

	priv1, pid1, err := SeedToEd25519(phrase)
	if err != nil {
		t.Fatalf("SeedToEd25519 #1: %v", err)
	}
	priv2, pid2, err := SeedToEd25519(phrase)
	if err != nil {
		t.Fatalf("SeedToEd25519 #2: %v", err)
	}

	if pid1 != pid2 {
		t.Fatalf("PeerID not deterministic: %s vs %s", pid1, pid2)
	}
	if !priv1.Equals(priv2) {
		t.Fatal("Ed25519 priv not deterministic")
	}
}

func TestSeedToEd25519_PeerIDMatchesPub(t *testing.T) {
	phrase, err := GenerateSeedPhrase(12)
	if err != nil {
		t.Fatalf("GenerateSeedPhrase: %v", err)
	}
	priv, pid, err := SeedToEd25519(phrase)
	if err != nil {
		t.Fatalf("SeedToEd25519: %v", err)
	}
	want, err := peer.IDFromPublicKey(priv.GetPublic())
	if err != nil {
		t.Fatalf("IDFromPublicKey: %v", err)
	}
	if pid != want {
		t.Fatalf("PeerID mismatch: %s vs %s", pid, want)
	}
}

func TestSeedToX25519_Deterministic(t *testing.T) {
	phrase, err := GenerateSeedPhrase(12)
	if err != nil {
		t.Fatalf("GenerateSeedPhrase: %v", err)
	}
	priv1, pub1, err := SeedToX25519(phrase)
	if err != nil {
		t.Fatalf("SeedToX25519 #1: %v", err)
	}
	priv2, pub2, err := SeedToX25519(phrase)
	if err != nil {
		t.Fatalf("SeedToX25519 #2: %v", err)
	}
	if priv1 != priv2 {
		t.Fatal("X25519 priv not deterministic")
	}
	if pub1 != pub2 {
		t.Fatal("X25519 pub not deterministic")
	}
}

func TestSeedToX25519_PubFromPriv(t *testing.T) {
	phrase, err := GenerateSeedPhrase(12)
	if err != nil {
		t.Fatalf("GenerateSeedPhrase: %v", err)
	}
	priv, pub, err := SeedToX25519(phrase)
	if err != nil {
		t.Fatalf("SeedToX25519: %v", err)
	}
	want, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		t.Fatalf("curve25519.X25519: %v", err)
	}
	for i := 0; i < 32; i++ {
		if pub[i] != want[i] {
			t.Fatalf("X25519 pub mismatch at byte %d", i)
		}
	}
}

func TestSeedToEd25519_X25519_DifferentDomains(t *testing.T) {
	phrase, err := GenerateSeedPhrase(12)
	if err != nil {
		t.Fatalf("GenerateSeedPhrase: %v", err)
	}
	// Один и тот же seed — разные HMAC-ключи — разные байты.
	seed, err := seedBytes(phrase)
	if err != nil {
		t.Fatalf("seedBytes: %v", err)
	}
	ed := hmacSHA256(seedHMACEd25519, seed)
	x := hmacSHA256(seedHMACX25519, seed)
	if string(ed) == string(x) {
		t.Fatal("domain separation broken: Ed25519 and X25519 derived bytes identical")
	}
}
