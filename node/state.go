package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
)

// ============================================================
// СОСТОЯНИЕ УЗЛА (СОХРАНЕНИЕ И ЗАГРУЗКА С ШИФРОВАНИЕМ)
// ============================================================

// State — структура, которая сохраняется на диск.
type State struct {
	Layers   [][]float64     `json:"layers"`
	MsgCount int             `json:"msgCount"`
	Messages []Message       `json:"messages"`
	Seen     map[string]bool `json:"seen"`
	PreHash  string          `json:"preHash"`
	AntiHash string          `json:"antiHash"`
}

// getEncryptionKey — возвращает 32-байтный ключ из пароля
func getEncryptionKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// encryptData — шифрует данные AES-GCM
func encryptData(data []byte, password string) ([]byte, error) {
	key := getEncryptionKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

// decryptData — расшифровывает данные AES-GCM
func decryptData(data []byte, password string) ([]byte, error) {
	key := getEncryptionKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, os.ErrInvalid
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// saveState — сохраняет состояние узла в файл (с шифрованием, если задан пароль)
func (n *Node) saveState() error {
	if n.stateFile == "" {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	seen := n.memory.seen
	if seen == nil {
		seen = make(map[string]bool)
	}

	state := State{
		Layers:   n.layers,
		MsgCount: n.msgCount,
		Messages: n.memory.GetAll(),
		Seen:     seen,
		PreHash:  n.preHash,
		AntiHash: n.antiHash,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Шифруем, если задан пароль
	password := os.Getenv("ISOTOPE_STATE_PASSWORD")
	if password != "" {
		data, err = encryptData(data, password)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll("state", 0755); err != nil {
		return err
	}

	return os.WriteFile(n.stateFile, data, 0644)
}

// loadStateData — загружает и расшифровывает данные состояния
func (n *Node) loadStateData() ([]byte, error) {
	data, err := os.ReadFile(n.stateFile)
	if err != nil {
		return nil, err
	}

	password := os.Getenv("ISOTOPE_STATE_PASSWORD")
	if password != "" {
		data, err = decryptData(data, password)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}