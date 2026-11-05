// node/aborted.go
package core

// node/aborted.go — v2.0.7: список отменённых (aborted) mediaID.
// Если отправитель прервал отправку — получатель не должен «догружать»
// файл при перезапуске (resumeFile / getMessagesViaLibP2P).
// Файл: isotope_aborted.json — рядом с contacts / requests / settings / deleted.
// Очистка: 24 часа (cleanupLoop) или [FILE_COMPLETE] (файл принят).

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// AbortedFile — формат файла isotope_aborted.json.
type AbortedFile struct {
	V       int                  `json:"v"`
	Aborted map[string]time.Time `json:"aborted"`
}

// ABORTED_VERSION — текущая версия формата.
const ABORTED_VERSION = 1

// ABORTED_TTL — сколько хранить mediaID в списке (24 часа).
const ABORTED_TTL = 24 * time.Hour

// AbortedStore — потокобезопасное хранилище aborted mediaID.
type AbortedStore struct {
	mu      sync.Mutex
	path    string
	aborted map[string]time.Time
}

// NewAbortedStore — создаёт хранилище и загружает файл (если есть).
func NewAbortedStore(path string) *AbortedStore {
	as := &AbortedStore{
		path:    path,
		aborted: make(map[string]time.Time),
	}
	if err := as.Load(); err != nil {
		log.Printf("[ABORTED] load failed: %v", err)
	}
	return as
}

// Load — читает файл. Если файла нет — пусто.
func (as *AbortedStore) Load() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.path == "" {
		return nil
	}

	data, err := os.ReadFile(as.path)
	if err != nil {
		if os.IsNotExist(err) {
			as.aborted = make(map[string]time.Time)
			return nil
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var file AbortedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if file.V != ABORTED_VERSION {
		log.Printf("[ABORTED] unexpected version %d (expected %d)", file.V, ABORTED_VERSION)
	}

	as.aborted = make(map[string]time.Time)
	if file.Aborted != nil {
		for id, t := range file.Aborted {
			if id != "" {
				as.aborted[id] = t
			}
		}
	}
	log.Printf("[ABORTED] loaded %d aborted mediaIDs", len(as.aborted))
	return nil
}

// Save — записывает файл.
func (as *AbortedStore) Save() error {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.saveLocked()
}

// saveLocked — сохраняет без лока.
func (as *AbortedStore) saveLocked() error {
	if as.path == "" {
		return fmt.Errorf("aborted path not set")
	}

	file := AbortedFile{
		V:       ABORTED_VERSION,
		Aborted: as.aborted,
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := os.WriteFile(as.path, data, 0600); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

// Add — добавляет mediaID в список отменённых.
// Идемпотентно.
func (as *AbortedStore) Add(mediaID string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if mediaID == "" {
		return fmt.Errorf("mediaID is required")
	}
	as.aborted[mediaID] = time.Now()
	log.Printf("[ABORTED] added %s", mediaID)
	return as.saveLocked()
}

// Remove — убирает mediaID из списка отменённых.
// Вызывается при [FILE_COMPLETE] — файл всё-таки принят.
func (as *AbortedStore) Remove(mediaID string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if mediaID == "" {
		return fmt.Errorf("mediaID is required")
	}
	if _, ok := as.aborted[mediaID]; !ok {
		return nil
	}
	delete(as.aborted, mediaID)
	log.Printf("[ABORTED] removed %s", mediaID)
	return as.saveLocked()
}

// IsAborted — true, если mediaID в списке отменённых.
func (as *AbortedStore) IsAborted(mediaID string) bool {
	as.mu.Lock()
	defer as.mu.Unlock()
	_, ok := as.aborted[mediaID]
	return ok
}

// GetAll — возвращает копию списка отменённых mediaID.
func (as *AbortedStore) GetAll() []string {
	as.mu.Lock()
	defer as.mu.Unlock()
	result := make([]string, 0, len(as.aborted))
	for id := range as.aborted {
		result = append(result, id)
	}
	return result
}

// CleanupExpired — удаляет mediaID старше ABORTED_TTL.
// Возвращает количество удалённых.
func (as *AbortedStore) CleanupExpired() int {
	as.mu.Lock()
	defer as.mu.Unlock()

	now := time.Now()
	removed := 0
	for id, t := range as.aborted {
		if now.Sub(t) > ABORTED_TTL {
			delete(as.aborted, id)
			removed++
		}
	}
	if removed > 0 {
		_ = as.saveLocked()
		log.Printf("[ABORTED] cleaned up %d expired", removed)
	}
	return removed
}

// node/aborted.go
