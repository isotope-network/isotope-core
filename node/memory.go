package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

const maxMessages = 10000

// ============================================================
// ПАМЯТЬ СООБЩЕНИЙ (ВЗВЕШЕННАЯ, С АРХИВОМ)
// ============================================================

// Message — структура одного сообщения
type Message struct {
	ID             string    `json:"id"`
	Text           string    `json:"text"`
	Sender         string    `json:"sender"`
	Time           string    `json:"time"`
	IsOwn          bool      `json:"isOwn"`
	Score          int       `json:"score"`
	Weight         float64   `json:"weight"`
	Created        time.Time `json:"created"`
	Archived       bool      `json:"archived"`
	DeliveryStatus string    `json:"deliveryStatus,omitempty"`
	ChannelID      string    `json:"channel,omitempty"`
}

// Memory — потокобезопасное хранилище сообщений (лимит 10 000)
type Memory struct {
	mu       sync.Mutex
	messages []Message
	seen     map[string]bool
}

func (m *Memory) Add(msg Message) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.seen == nil {
		m.seen = make(map[string]bool)
	}
	if m.seen[msg.ID] {
		return false
	}
	m.seen[msg.ID] = true

	if msg.Weight == 0 {
		msg.Weight = 0.5
	}
	if msg.Created.IsZero() {
		msg.Created = time.Now()
	}

	m.messages = append(m.messages, msg)

	if len(m.messages) > maxMessages {
		m.enforceLimit()
	}

	return true
}

func (m *Memory) enforceLimit() {
	var alive []Message
	archivedRemoved := 0
	for _, msg := range m.messages {
		if msg.Archived && len(m.messages)-archivedRemoved > maxMessages {
			delete(m.seen, msg.ID)
			archivedRemoved++
		} else {
			alive = append(alive, msg)
		}
	}
	m.messages = alive

	if len(m.messages) > maxMessages {
		excess := len(m.messages) - maxMessages
		var trimmed []Message
		for i, msg := range m.messages {
			if i < excess {
				delete(m.seen, msg.ID)
			} else {
				trimmed = append(trimmed, msg)
			}
		}
		m.messages = trimmed
		log.Printf("[MEMORY] Emergency trim: %d oldest messages removed", excess)
	}
}

func (m *Memory) GetAll() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetByChannel возвращает сообщения канала (новые сверху)
func (m *Memory) GetByChannel(channelID string) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []Message
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].ChannelID == channelID {
			result = append(result, m.messages[i])
		}
	}
	return result
}

func (m *Memory) GetActiveMessages(threshold float64) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	var active []Message
	for _, msg := range m.messages {
		if msg.Archived {
			continue
		}
		ageHours := time.Since(msg.Created).Hours()
		weight := msg.Weight - ageHours*0.01
		if weight < 0 {
			weight = 0
		}
		if weight >= threshold {
			msgCopy := msg
			msgCopy.Weight = weight
			active = append(active, msgCopy)
		}
	}
	return active
}

func (m *Memory) ArchiveOld(threshold float64) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for i := range m.messages {
		if m.messages[i].Archived {
			continue
		}
		ageHours := time.Since(m.messages[i].Created).Hours()
		weight := m.messages[i].Weight - ageHours*0.01
		if weight < threshold {
			m.messages[i].Archived = true
			count++
		}
	}
	return count
}

func (m *Memory) RestoreFromArchive(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.messages {
		if m.messages[i].ID == id && m.messages[i].Archived {
			m.messages[i].Archived = false
			m.messages[i].Weight = 0.5
			m.messages[i].Created = time.Now()
			return true
		}
	}
	return false
}

func (m *Memory) UpdateWeight(id string, delta float64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.messages {
		if m.messages[i].ID == id {
			m.messages[i].Weight += delta
			if m.messages[i].Weight > 1.0 {
				m.messages[i].Weight = 1.0
			}
			if m.messages[i].Weight < 0.0 {
				m.messages[i].Weight = 0.0
			}
			return true
		}
	}
	return false
}

func (m *Memory) UpdateDeliveryStatus(id string, status string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.messages {
		if m.messages[i].ID == id {
			m.messages[i].DeliveryStatus = status
			return true
		}
	}
	return false
}

func (m *Memory) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func (m *Memory) CountArchived() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, msg := range m.messages {
		if msg.Archived {
			count++
		}
	}
	return count
}

func (m *Memory) FindSimilar(text string, threshold float64) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	inputVec := textToVector(text)
	var similar []Message

	for _, msg := range m.messages {
		if msg.Archived {
			continue
		}
		ageHours := time.Since(msg.Created).Hours()
		weight := msg.Weight - ageHours*0.01
		if weight < 0.5 {
			continue
		}

		msgVec := textToVector(msg.Text)
		similarity := cosineSimilarity(inputVec, msgVec)
		if similarity > threshold {
			msgCopy := msg
			msgCopy.Weight = weight
			similar = append(similar, msgCopy)
		}
	}
	return similar
}

func (m *Memory) PurgeDead(threshold float64) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var alive []Message
	removed := 0
	for _, msg := range m.messages {
		ageHours := time.Since(msg.Created).Hours()
		weight := msg.Weight - ageHours*0.01
		if weight < threshold {
			delete(m.seen, msg.ID)
			removed++
		} else {
			alive = append(alive, msg)
		}
	}
	m.messages = alive
	return removed
}

func (m *Memory) SaveArchive(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var archived []Message
	for _, msg := range m.messages {
		if msg.Archived {
			archived = append(archived, msg)
		}
	}
	if len(archived) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(archived, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Memory) LoadArchive(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var archived []Message
	if err := json.Unmarshal(data, &archived); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.seen == nil {
		m.seen = make(map[string]bool)
	}
	loaded := 0
	for _, msg := range archived {
		if m.seen[msg.ID] {
			continue
		}
		m.seen[msg.ID] = true
		msg.Archived = true
		m.messages = append(m.messages, msg)
		loaded++
	}
	if loaded > 0 {
		log.Printf("[MEMORY] Loaded %d archived messages from %s", loaded, path)
	}
	return nil
}