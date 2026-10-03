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
	// TtlPeriod — период удаления сообщений по умолчанию.
	// "10s" | "1m" | "10m" | "1h" | "24h" | "7d" | "30d" | "forever".
	TtlPeriod string `json:"ttl_period,omitempty"`
	// TtlMode — режим удаления: "after_read" | "hard".
	// nil — при "forever" (нет режима).
	TtlMode *string `json:"ttl_mode"`
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

	// Миграция my_ttl → ttl_period + ttl_mode.
	// Если в файле есть старое my_ttl и нет нового ttl_period — конвертируем.
	if _, hasNew := raw["ttl_period"]; !hasNew {
		if oldTtl, hasOld := raw["my_ttl"]; hasOld {
			period := secondsToPeriod(fmt.Sprintf("%v", oldTtl))
			loaded.TtlPeriod = period
			if period != "forever" {
				mode := "after_read"
				loaded.TtlMode = &mode
			}
		}
	}

	// Дефолт: "forever" без режима.
	if loaded.TtlPeriod == "" {
		loaded.TtlPeriod = "forever"
	}

	s.data = loaded
	log.Printf("[SETTINGS] loaded (my_read_enabled=%v, my_display_name=%q, ttl_period=%q)",
		s.data.MyReadEnabled, s.data.MyDisplayName, s.data.TtlPeriod)
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

// GetTtl — возвращает период и режим удаления сообщений.
// period: "10s" | "1m" | ... | "forever".
// mode: nil (при forever) | &"after_read" | &"hard".
func (s *SettingsStore) GetTtl() (string, *string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	period := s.data.TtlPeriod
	if period == "" {
		period = "forever"
	}
	return period, s.data.TtlMode
}

// SetTtl — устанавливает период и режим удаления, сохраняет в файл.
func (s *SettingsStore) SetTtl(period string, mode *string) error {
	s.mu.Lock()
	s.data.TtlPeriod = period
	s.data.TtlMode = mode
	s.mu.Unlock()
	return s.Save()
}

// secondsToPeriod — конвертирует старое значение my_ttl (секунды строкой)
// в новый формат ttl_period.
func secondsToPeriod(sec string) string {
	switch sec {
	case "", "0":
		return "forever"
	case "10":
		return "10s"
	case "60":
		return "1m"
	case "600":
		return "10m"
	case "3600":
		return "1h"
	case "86400":
		return "24h"
	case "604800":
		return "7d"
	case "2592000":
		return "30d"
	default:
		return "forever"
	}
}

// parsePeriod — конвертирует ttl_period в секунды.
// "forever" или неизвестное — 0.
func parsePeriod(period string) int {
	switch period {
	case "10s":
		return 10
	case "1m":
		return 60
	case "10m":
		return 600
	case "1h":
		return 3600
	case "24h":
		return 86400
	case "7d":
		return 604800
	case "30d":
		return 2592000
	default:
		return 0
	}
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
