// node/deleted.go
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// ============================================================
// СПИСОК УДАЛЁННЫХ КОНТАКТОВ
// ============================================================
//
// Пользователь удалил контакт — он не должен возвращаться
// после перезапуска приложения. Но «память приложения» — не одно
// место (контакты, state, P2PService history, UI). Удалить отовсюду
// нельзя. Значит — фильтр: «этот peerID не показывать».
//
// Файл: isotope_deleted.json — рядом с contacts / requests / settings.
// «Разные ключи — разные файлы» (принцип Публикатора).
//
// Логика:
//   - RemoveContact(peerID) → Add(peerID).
//   - При загрузке контактов — не добавлять, если IsDeleted.
//   - P2PService._loadNodes / addDiscoveredPeer — фильтровать.
//   - При QR-возврате: RemoveFromDeleted(peerID) → AddContact.
//
// Свобода человека: удалил — удалил. Даже если пишет, даже если в сети.
// Сеть работает. UI — фильтрует.

// DeletedFile — формат файла isotope_deleted.json.
type DeletedFile struct {
	V       int      `json:"v"`
	Deleted []string `json:"deleted"`
}

// DELETED_VERSION — текущая версия формата.
const DELETED_VERSION = 1

// DeletedStore — потокобезопасное хранилище удалённых peerID.
type DeletedStore struct {
	mu      sync.Mutex
	path    string
	deleted map[string]bool
}

// NewDeletedStore — создаёт хранилище и загружает файл (если есть).
func NewDeletedStore(path string) *DeletedStore {
	ds := &DeletedStore{
		path:    path,
		deleted: make(map[string]bool),
	}
	if err := ds.Load(); err != nil {
		log.Printf("[DELETED] load failed: %v", err)
	}
	return ds
}

// Load — читает файл. Если файла нет — пусто.
func (ds *DeletedStore) Load() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.path == "" {
		return nil
	}

	data, err := os.ReadFile(ds.path)
	if err != nil {
		if os.IsNotExist(err) {
			ds.deleted = make(map[string]bool)
			return nil
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var file DeletedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if file.V != DELETED_VERSION {
		log.Printf("[DELETED] unexpected version %d (expected %d)", file.V, DELETED_VERSION)
	}

	ds.deleted = make(map[string]bool)
	for _, id := range file.Deleted {
		if id != "" {
			ds.deleted[id] = true
		}
	}
	log.Printf("[DELETED] loaded %d deleted peerIDs", len(ds.deleted))
	return nil
}

// Save — записывает файл.
func (ds *DeletedStore) Save() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.saveLocked()
}

// saveLocked — сохраняет без лока (вызывается из методов с уже взятым локом).
func (ds *DeletedStore) saveLocked() error {
	if ds.path == "" {
		return fmt.Errorf("deleted path not set")
	}

	list := make([]string, 0, len(ds.deleted))
	for id := range ds.deleted {
		list = append(list, id)
	}
	file := DeletedFile{
		V:       DELETED_VERSION,
		Deleted: list,
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := os.WriteFile(ds.path, data, 0600); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

// Add — добавляет peerID в список удалённых.
// Идемпотентно. Если уже есть — просто nil.
func (ds *DeletedStore) Add(peerID string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if peerID == "" {
		return fmt.Errorf("peerID is required")
	}
	if ds.deleted[peerID] {
		return nil
	}
	ds.deleted[peerID] = true
	log.Printf("[DELETED] added %s", peerID)
	return ds.saveLocked()
}

// RemoveFromDeleted — убирает peerID из списка удалённых.
// Используется при QR-возврате контакта.
func (ds *DeletedStore) RemoveFromDeleted(peerID string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if peerID == "" {
		return fmt.Errorf("peerID is required")
	}
	if !ds.deleted[peerID] {
		return nil
	}
	delete(ds.deleted, peerID)
	log.Printf("[DELETED] removed %s", peerID)
	return ds.saveLocked()
}

// IsDeleted — true, если peerID в списке удалённых.
func (ds *DeletedStore) IsDeleted(peerID string) bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.deleted[peerID]
}

// GetAll — возвращает копию списка удалённых peerID.
func (ds *DeletedStore) GetAll() []string {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	result := make([]string, 0, len(ds.deleted))
	for id := range ds.deleted {
		result = append(result, id)
	}
	return result
}

// node/deleted.go
