// node/settings.go
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// Settings — пользовательские настройки узла.
// Хранятся отдельно от state (это предпочтения пользователя, не состояние сети).
// Файл: isotope_settings.json — рядом со state.
type Settings struct {
	V             int  `json:"v"`
	MyReadEnabled bool `json:"my_read_enabled"`
	// MyDisplayName — представление по умолчанию (как меня видеть другим).
	// Используется в QR и [CONTACT_REQUEST], если не переопределено.
	MyDisplayName string `json:"my_display_name,omitempty"`
	// MyTtl — время жизни сообщения по умолчанию (в секундах).
	// "0" — Вечно (не удалять). Применяется к новым сообщениям.
	// Хранится как строка (гибко: "10", "60", "3600", "0").
	MyTtl string `json:"my_ttl,omitempty"`
}

// SETTINGS_VERSION — текущая версия формата.
const SETTINGS_VERSION = 1

// SettingsStore — потокобезопасное хранилище настроек.
type SettingsStore struct {
	mu   sync.RWMutex
	file string
	data Settings
}

// NewSettingsStore — создаёт store и загружает файл (если есть).
// Если файла нет — создаёт с дефолтами (MyReadEnabled: true).
func NewSettingsStore(file string) *SettingsStore {
	s := &SettingsStore{
		file: file,
		data: Settings{
			V:             SETTINGS_VERSION,
			MyReadEnabled: true, // дефолт: делюсь статусом прочтения
		},
	}
	if err := s.Load(); err != nil {
		log.Printf("[SETTINGS] load failed: %v", err)
	}
	return s
}

// Load — читает файл. Если файла нет — дефолты (без ошибки).
// При несовпадении версии — логирует, но не падает (обратная совместимость).
// Если поля my_read_enabled нет в JSON — дефолт true (обратная совместимость).
func (s *SettingsStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == "" {
		return nil
	}

	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			// Файла нет — дефолты в памяти. Save при первом Set.
			return nil
		}
		return fmt.Errorf("read failed: %w", err)
	}

	// Основной разбор.
	var loaded Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if loaded.V == 0 {
		loaded.V = SETTINGS_VERSION
	}
	if loaded.V != SETTINGS_VERSION {
		log.Printf("[SETTINGS] unexpected version %d (expected %d)", loaded.V, SETTINGS_VERSION)
	}

	// Обратная совместимость: старый файл без my_read_enabled
	// (поле отсутствует → false) считаем как true.
	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)
	if _, ok := raw["my_read_enabled"]; !ok {
		loaded.MyReadEnabled = true
	}

	s.data = loaded
	log.Printf("[SETTINGS] loaded (my_read_enabled=%v, my_display_name=%q)",
		s.data.MyReadEnabled, s.data.MyDisplayName)
	return nil
}

// Save — записывает файл.
func (s *SettingsStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *SettingsStore) saveLocked() error {
	if s.file == "" {
		return fmt.Errorf("settings file not set")
	}
	s.data.V = SETTINGS_VERSION
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}
	if err := os.WriteFile(s.file, data, 0600); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

// GetMyReadEnabled — возвращает настройку "делюсь ли статусом прочтения".
func (s *SettingsStore) GetMyReadEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.MyReadEnabled
}

// SetMyReadEnabled — устанавливает настройку и сохраняет в файл.
func (s *SettingsStore) SetMyReadEnabled(enabled bool) error {
	s.mu.Lock()
	if s.data.MyReadEnabled == enabled {
		s.mu.Unlock()
		return nil // ничего не изменилось
	}
	s.data.MyReadEnabled = enabled
	s.mu.Unlock()
	return s.Save()
}

// GetMyDisplayName — возвращает представление по умолчанию.
func (s *SettingsStore) GetMyDisplayName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.MyDisplayName
}

// GetMyTtl — возвращает TTL по умолчанию (в секундах, строкой).
// Пусто или "0" — Вечно.
func (s *SettingsStore) GetMyTtl() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data.MyTtl == "" {
		return "0"
	}
	return s.data.MyTtl
}

// SetMyTtl — устанавливает TTL по умолчанию и сохраняет в файл.
func (s *SettingsStore) SetMyTtl(ttl string) error {
	s.mu.Lock()
	if s.data.MyTtl == ttl {
		s.mu.Unlock()
		return nil
	}
	s.data.MyTtl = ttl
	s.mu.Unlock()
	return s.Save()
}

// SetMyDisplayName — устанавливает представление по умолчанию и сохраняет.
func (s *SettingsStore) SetMyDisplayName(name string) error {
	s.mu.Lock()
	if s.data.MyDisplayName == name {
		s.mu.Unlock()
		return nil
	}
	s.data.MyDisplayName = name
	s.mu.Unlock()
	return s.Save()
}

// node/settings.go
