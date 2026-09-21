// node/contacts.go
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Contact — запись о контакте.
// Публичные ключи: Ed25519 (подпись) и X25519 (шифрование).
// Signature — подпись ed25519_pub от PeerID-ключа владельца (заполняется на 4.4).
// Verified — true после проверки подписи (4.4).
// Name — локальное имя, пустое при добавлении.
type Contact struct {
	PeerID     string `json:"peerID"`
	Ed25519Pub string `json:"ed25519_pub"`
	X25519Pub  string `json:"x25519_pub"`
	Signature  string `json:"signature"`
	Verified   bool   `json:"verified"`
	Name       string `json:"name"`
	AddedAt    string `json:"added_at"`
}

// ContactsFile — формат файла isotope_contacts.json.
// V — версия формата (открытая дверь для миграции).
type ContactsFile struct {
	V        int       `json:"v"`
	Contacts []Contact `json:"contacts"`
}

// CONTACTS_VERSION — текущая версия формата.
const CONTACTS_VERSION = 1

// ContactsStore — потокобезопасное хранилище контактов.
type ContactsStore struct {
	mu       sync.Mutex
	path     string
	contacts []Contact
}

// NewContactsStore — создаёт хранилище и загружает файл (если есть).
func NewContactsStore(path string) *ContactsStore {
	cs := &ContactsStore{
		path:     path,
		contacts: []Contact{},
	}
	if err := cs.Load(); err != nil {
		log.Printf("[CONTACTS] load failed: %v", err)
	}
	return cs
}

// Load — читает файл. Если файла нет — пустой список.
func (cs *ContactsStore) Load() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.path == "" {
		return nil
	}

	data, err := os.ReadFile(cs.path)
	if err != nil {
		if os.IsNotExist(err) {
			cs.contacts = []Contact{}
			return nil
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var file ContactsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if file.V != CONTACTS_VERSION {
		log.Printf("[CONTACTS] unexpected version %d (expected %d)", file.V, CONTACTS_VERSION)
	}

	if file.Contacts == nil {
		file.Contacts = []Contact{}
	}
	cs.contacts = file.Contacts
	log.Printf("[CONTACTS] loaded %d contacts", len(cs.contacts))
	return nil
}

// Save — записывает файл.
func (cs *ContactsStore) Save() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.saveLocked()
}

// saveLocked — сохраняет без лока (вызывается из методов с уже взятым локом).
func (cs *ContactsStore) saveLocked() error {
	if cs.path == "" {
		return fmt.Errorf("contacts path not set")
	}

	file := ContactsFile{
		V:        CONTACTS_VERSION,
		Contacts: cs.contacts,
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := os.WriteFile(cs.path, data, 0600); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

// Add — добавляет или обновляет контакт по PeerID.
// Если контакт уже есть — обновляет поля, кроме Name (сохраняется пользовательское).
// Verified — сбрасывается в false (пока подпись не проверена на 4.4).
func (cs *ContactsStore) Add(c Contact) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if c.PeerID == "" {
		return fmt.Errorf("peerID is required")
	}

	// Если AddedAt не задан — ставим сейчас.
	if c.AddedAt == "" {
		c.AddedAt = time.Now().UTC().Format(time.RFC3339)
	}

	// Ищем существующий.
	for i := range cs.contacts {
		if cs.contacts[i].PeerID == c.PeerID {
			// Обновляем ключи и подпись, имя — сохраняем.
			existingName := cs.contacts[i].Name
			cs.contacts[i] = c
			if existingName != "" && c.Name == "" {
				cs.contacts[i].Name = existingName
			}
			cs.contacts[i].Verified = false
			if cs.contacts[i].AddedAt == "" {
				cs.contacts[i].AddedAt = time.Now().UTC().Format(time.RFC3339)
			}
			log.Printf("[CONTACTS] updated %s", c.PeerID)
			return cs.saveLocked()
		}
	}

	// Новый.
	c.Verified = false
	cs.contacts = append(cs.contacts, c)
	log.Printf("[CONTACTS] added %s", c.PeerID)
	return cs.saveLocked()
}

// Get — возвращает контакт по PeerID.
func (cs *ContactsStore) Get(peerID string) (Contact, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for _, c := range cs.contacts {
		if c.PeerID == peerID {
			return c, true
		}
	}
	return Contact{}, false
}

// GetAll — возвращает копию списка контактов.
func (cs *ContactsStore) GetAll() []Contact {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	result := make([]Contact, len(cs.contacts))
	copy(result, cs.contacts)
	return result
}

// Count — количество контактов.
func (cs *ContactsStore) Count() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.contacts)
}

// Remove — удаляет контакт по PeerID.
func (cs *ContactsStore) Remove(peerID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var kept []Contact
	for _, c := range cs.contacts {
		if c.PeerID != peerID {
			kept = append(kept, c)
		}
	}
	if len(kept) == len(cs.contacts) {
		return fmt.Errorf("contact %s not found", peerID)
	}
	cs.contacts = kept
	log.Printf("[CONTACTS] removed %s", peerID)
	return cs.saveLocked()
}