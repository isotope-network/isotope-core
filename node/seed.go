package core

// D:\isotope\node\seed.go
// seed.go — BIP39 seed-фраза → Ed25519 (PeerID + подпись) + X25519 (шифрование).
// Восстановление PeerID после сноса приложения.
// Ed25519: HMAC-SHA256("OpenBazaar seed") — путь Mobazha (проверенный).
// X25519: HMAC-SHA256("OpenBazaar x25519") — domain separation (Публикатор).
// Seed — только в памяти. Не логировать. Не в plaintext на диске. Не в сеть.
// BIP39: github.com/kslamph/bip39-hdwallet/bip39 — drop-in замена tyler-smith v1.1.0
// (tyler-smith удалён с GitHub; luxfi-форк имеет битые внутренние ссылки на gojuukaze).

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/kslamph/bip39-hdwallet/bip39"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/curve25519"
)

const (
	seedHMACEd25519 = "OpenBazaar seed"
	seedHMACX25519  = "OpenBazaar x25519"
	ed25519Bits     = 256
)

// GenerateSeedPhrase — новая BIP39-фраза на 12 (128 бит) или 24 (256 бит) слова.
func GenerateSeedPhrase(words int) (string, error) {
	var entropyBits int
	switch words {
	case 12:
		entropyBits = 128
	case 24:
		entropyBits = 256
	default:
		return "", fmt.Errorf("seed: unsupported words count %d (want 12 or 24)", words)
	}
	entropy, err := bip39.NewEntropy(entropyBits)
	if err != nil {
		return "", fmt.Errorf("seed: NewEntropy: %w", err)
	}
	phrase, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("seed: NewMnemonic: %w", err)
	}
	return phrase, nil
}

// ValidateSeedPhrase — BIP39 checksum. Пустая/битая фраза → ошибка.
func ValidateSeedPhrase(phrase string) error {
	if phrase == "" {
		return fmt.Errorf("seed: empty phrase")
	}
	if !bip39.IsMnemonicValid(phrase) {
		return fmt.Errorf("seed: invalid mnemonic (checksum or word)")
	}
	return nil
}

// seedBytes — фраза → 64-байтный BIP39 seed. Общая часть для Ed25519 и X25519.
func seedBytes(phrase string) ([]byte, error) {
	if err := ValidateSeedPhrase(phrase); err != nil {
		return nil, err
	}
	seed, err := bip39.NewSeedWithErrorChecking(phrase, "")
	if err != nil {
		return nil, fmt.Errorf("seed: NewSeedWithErrorChecking: %w", err)
	}
	if len(seed) != 64 {
		return nil, fmt.Errorf("seed: unexpected seed length %d", len(seed))
	}
	return seed, nil
}

// hmacSHA256 — 32 байта из seed с domain-separation ключом.
func hmacSHA256(key string, seed []byte) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(seed)
	return h.Sum(nil)
}

// SeedToEd25519 — фраза → Ed25519 (PeerID + подпись). Детерминированно.
func SeedToEd25519(phrase string) (libp2pcrypto.PrivKey, peer.ID, error) {
	seed, err := seedBytes(phrase)
	if err != nil {
		return nil, "", err
	}
	derived := hmacSHA256(seedHMACEd25519, seed)
	reader := bytes.NewReader(derived)
	priv, _, err := libp2pcrypto.GenerateKeyPairWithReader(libp2pcrypto.Ed25519, ed25519Bits, reader)
	if err != nil {
		return nil, "", fmt.Errorf("seed: GenerateKeyPairWithReader(Ed25519): %w", err)
	}
	pid, err := peer.IDFromPublicKey(priv.GetPublic())
	if err != nil {
		return nil, "", fmt.Errorf("seed: IDFromPublicKey: %w", err)
	}
	return priv, pid, nil
}

// SeedToX25519 — фраза → X25519 (шифрование). Детерминированно.
// priv: HMAC-SHA256("OpenBazaar x25519"). pub: curve25519.X25519(priv, Basepoint).
func SeedToX25519(phrase string) (priv [32]byte, pub [32]byte, err error) {
	seed, err := seedBytes(phrase)
	if err != nil {
		return priv, pub, err
	}
	derived := hmacSHA256(seedHMACX25519, seed)
	if len(derived) != 32 {
		return priv, pub, fmt.Errorf("seed: unexpected X25519 derived length %d", len(derived))
	}
	copy(priv[:], derived)
	pubBytes, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return priv, pub, fmt.Errorf("seed: X25519 pub: %w", err)
	}
	copy(pub[:], pubBytes)
	return priv, pub, nil
}
