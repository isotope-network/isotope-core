// node/state.go
package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ============================================================
// СОСТОЯНИЕ УЗЛА (СОХРАНЕНИЕ И ЗАГРУЗКА С ШИФРОВАНИЕМ)
// ============================================================

// State — структура, которая сохраняется на диск.
type State struct {
	Layers       [][]float64     `json:"layers"`
	MsgCount     int             `json:"msgCount"`
	Messages     []Message       `json:"messages"`
	Seen         map[string]bool `json:"seen"`
	PreHash      string          `json:"preHash"`
	AntiHash     string          `json:"antiHash"`
	RoutingTable []string        `json:"routingTable"`
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

// saveState — сохраняет состояние узла в файл
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

	var routingTable []string
	if n.dhtNode != nil {
		data := n.dhtNode.SaveRoutingTable()
		_ = json.Unmarshal(data, &routingTable)
	}

	// Не сохраняем истёкшие — они не должны возродиться после перезапуска.
	// Файловые чанки: не пишем Text/PlainText (base64) — только метаданные.
	// Иначе state-файл пухнет на десятки МБ (440 чанков × ~170 КБ).
	now := time.Now()
	allMsgs := n.memory.GetAll()
	aliveMsgs := make([]Message, 0, len(allMsgs))
	for _, m := range allMsgs {
		if !m.ExpiresAt.IsZero() && now.After(m.ExpiresAt) {
			continue
		}
		if m.MediaType == "file" && m.ChunkTotal > 0 {
			m.Text = ""
			m.PlainText = ""
		}
		aliveMsgs = append(aliveMsgs, m)
	}

	state := State{
		Layers:       n.layers,
		MsgCount:     n.msgCount,
		Messages:     aliveMsgs,
		Seen:         seen,
		PreHash:      n.preHash,
		AntiHash:     n.antiHash,
		RoutingTable: routingTable,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	password := os.Getenv("ISOTOPE_STATE_PASSWORD")
	if password != "" {
		data, err = encryptData(data, password)
		if err != nil {
			return err
		}
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

// sentPath — путь к копии отправленного файла.
// Детерминирован: <stateFile dir>/isotope_media/sent/<MediaID>.bin.
// Используется для flushPending: перечитать чанк с диска, переслать.
func (n *Node) sentPath(mediaID string) string {
	dir := filepath.Dir(n.stateFile)
	return filepath.Join(dir, "isotope_media", "sent", mediaID+".bin")
}

// sentDirPath — путь к директории sent/.
func (n *Node) sentDirPath() string {
	dir := filepath.Dir(n.stateFile)
	return filepath.Join(dir, "isotope_media", "sent")
}

// savePrivateKey — сохраняет приватный ключ рядом с stateFile
func (n *Node) savePrivateKey(key []byte) error {
	keyFile := n.stateFile + ".key"

	// Создаём директорию, если не существует
	dir := filepath.Dir(keyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(keyFile, key, 0600)
}

// loadPrivateKey — загружает приватный ключ
func (n *Node) loadPrivateKey() ([]byte, error) {
	keyFile := n.stateFile + ".key"
	return os.ReadFile(keyFile)
}

// node/state.go