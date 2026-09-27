// node/requests.go
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// ============================================================
// ХРАНИЛИЩЕ ЗАПРОСОВ НА КОНТАКТ (контакт-протокол, этап 5)
// ============================================================

// REQUEST_STATUS_* — статусы запроса.
// Сейчас используется только pending.
// accepted / rejected — заложены на будущее (история).
const (
	RequestStatusPending  = "pending"
	RequestStatusAccepted = "accepted"
	RequestStatusRejected = "rejected"
)

// ContactRequest — входящий запрос на добавление в контакты.
// Хранится в isotope_requests.json до accept/reject.
type ContactRequest struct {
	ID          string `json:"id"`           // msg_id исходного [CONTACT_REQUEST]
	PeerID      string `json:"peerID"`       // PeerID отправителя
	Name        string `json:"name"`         // имя (может быть пусто)
	Ed25519Pub  string `json:"ed25519_pub"`  // публичный ключ подписи
	X25519Pub   string `json:"x25519_pub"`   // публичный ключ шифрования
	Signature   string `json:"signature"`    // подпись peerID || x25519_pub
	ReadEnabled bool   `json:"read_enabled"` // делится ли отправитель статусом прочтения
	ReceivedAt  string `json:"received_at"`  // RFC3339
	Status      string `json:"status"`       // pending (сейчас) / accepted / rejected (потом)
}

// RequestsFile — формат файла isotope_requests.json.
// V — версия формата (открытая дверь для миграции).
type RequestsFile struct {
	V        int              `json:"v"`
	Requests []ContactRequest `json:"requests"`
}

// REQUESTS_VERSION — текущая версия формата.
const REQUESTS_VERSION = 1

// RequestsStore — потокобезопасное хранилище входящих запросов.
type RequestsStore struct {
	mu       sync.Mutex
	path     string
	requests []ContactRequest
}

// NewRequestsStore — создаёт хранилище и загружает файл (если есть).
func NewRequestsStore(path string) *RequestsStore {
	rs := &RequestsStore{
		path:     path,
		requests: []ContactRequest{},
	}
	if err := rs.Load(); err != nil {
		log.Printf("[REQUESTS] load failed: %v", err)
	}
	return rs
}

// Load — читает файл. Если файла нет — пустой список.
func (rs *RequestsStore) Load() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.path == "" {
		return nil
	}

	data, err := os.ReadFile(rs.path)
	if err != nil {
		if os.IsNotExist(err) {
			rs.requests = []ContactRequest{}
			return nil
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var file RequestsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if file.V != REQUESTS_VERSION {
		log.Printf("[REQUESTS] unexpected version %d (expected %d)", file.V, REQUESTS_VERSION)
	}

	if file.Requests == nil {
		file.Requests = []ContactRequest{}
	}
	rs.requests = file.Requests
	log.Printf("[REQUESTS] loaded %d requests", len(rs.requests))
	return nil
}

// Save — записывает файл.
func (rs *RequestsStore) Save() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.saveLocked()
}

// saveLocked — сохраняет без лока (вызывается из методов с уже взятым локом).
func (rs *RequestsStore) saveLocked() error {
	if rs.path == "" {
		return fmt.Errorf("requests path not set")
	}

	file := RequestsFile{
		V:        REQUESTS_VERSION,
		Requests: rs.requests,
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := os.WriteFile(rs.path, data, 0600); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

// Add — добавляет или обновляет запрос по ID.
// Если запрос с таким ID уже есть — обновляет поля.
// Status нового запроса — pending.
func (rs *RequestsStore) Add(req ContactRequest) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if req.ID == "" {
		return fmt.Errorf("request id is required")
	}
	if req.PeerID == "" {
		return fmt.Errorf("peerID is required")
	}

	if req.ReceivedAt == "" {
		req.ReceivedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if req.Status == "" {
		req.Status = RequestStatusPending
	}

	for i := range rs.requests {
		if rs.requests[i].ID == req.ID {
			rs.requests[i] = req
			log.Printf("[REQUESTS] updated %s (peer=%s)", req.ID, req.PeerID)
			return rs.saveLocked()
		}
	}

	rs.requests = append(rs.requests, req)
	log.Printf("[REQUESTS] added %s (peer=%s)", req.ID, req.PeerID)
	return rs.saveLocked()
}

// Get — возвращает запрос по ID.
func (rs *RequestsStore) Get(id string) (ContactRequest, bool) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	for _, r := range rs.requests {
		if r.ID == id {
			return r, true
		}
	}
	return ContactRequest{}, false
}

// GetAll — возвращает копию всех запросов.
func (rs *RequestsStore) GetAll() []ContactRequest {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	result := make([]ContactRequest, len(rs.requests))
	copy(result, rs.requests)
	return result
}

// GetPending — возвращает только запросы со статусом pending.
// Используется UI: показать «Запросы» сверху списка контактов.
func (rs *RequestsStore) GetPending() []ContactRequest {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	var result []ContactRequest
	for _, r := range rs.requests {
		if r.Status == RequestStatusPending {
			result = append(result, r)
		}
	}
	return result
}

// Count — количество запросов.
func (rs *RequestsStore) Count() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return len(rs.requests)
}

// CountPending — количество pending-запросов.
func (rs *RequestsStore) CountPending() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	count := 0
	for _, r := range rs.requests {
		if r.Status == RequestStatusPending {
			count++
		}
	}
	return count
}

// SetStatus — обновляет статус запроса по ID.
// Используется для accepted / rejected (потом — для истории).
func (rs *RequestsStore) SetStatus(id, status string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	for i := range rs.requests {
		if rs.requests[i].ID == id {
			rs.requests[i].Status = status
			log.Printf("[REQUESTS] status %s → %s", id, status)
			return rs.saveLocked()
		}
	}
	return fmt.Errorf("request %s not found", id)
}

// Remove — удаляет запрос по ID.
func (rs *RequestsStore) Remove(id string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	var kept []ContactRequest
	for _, r := range rs.requests {
		if r.ID != id {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(rs.requests) {
		return fmt.Errorf("request %s not found", id)
	}
	rs.requests = kept
	log.Printf("[REQUESTS] removed %s", id)
	return rs.saveLocked()
}
// node/requests.go