package main

import (
	"sync"
	"time"
)

// ============================================================
// ПАМЯТЬ СООБЩЕНИЙ (ВЗВЕШЕННАЯ, С АРХИВОМ)
// ============================================================

// Message — структура одного сообщения
type Message struct {
	ID              string    `json:"id"`
	Text            string    `json:"text"`
	Sender          string    `json:"sender"`
	Time            string    `json:"time"`
	IsOwn           bool      `json:"isOwn"`
	Score           int       `json:"score"`
	Weight          float64   `json:"weight"`
	Created         time.Time `json:"created"`
	Archived        bool      `json:"archived"`
	Priority        int       `json:"priority"`
	Mode            int       `json:"mode"`
	Relayed         bool      `json:"relayed"`
	ExpiresAt       time.Time `json:"expiresAt"`
	ReplicatedFrom  string    `json:"replicatedFrom"` // от какого узла реплика
	ReplicatedAt    time.Time `json:"replicatedAt"`   // когда реплицировано
}

// Memory — потокобезопасное хранилище сообщений (без лимита)
type Memory struct {
	mu       sync.Mutex
	messages []Message
	seen     map[string]bool
}

// Add — добавляет сообщение, если оно не дубликат
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
	return true
}

// GetAll — возвращает все сообщения
func (m *Memory) GetAll() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetActiveMessages — возвращает неархивированные сообщения с весом >= threshold
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

// ArchiveOld — отправляет в архив сообщения с весом ниже порога
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

// RestoreFromArchive — восстанавливает сообщение из архива по ID
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

// UpdateWeight — обновляет вес сообщения по ID
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

// Count — возвращает количество сообщений
func (m *Memory) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// CountArchived — возвращает количество архивированных сообщений
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

// FindSimilar — находит активные сообщения, похожие на заданный текст
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

// PurgeDead — удаляет сообщения с весом ниже порога (с учётом старения)
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

// DeleteExpired — удаляет сообщения с истёкшим сроком жизни
func (m *Memory) DeleteExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var alive []Message
	removed := 0
	for _, msg := range m.messages {
		if !msg.ExpiresAt.IsZero() && now.After(msg.ExpiresAt) {
			delete(m.seen, msg.ID)
			removed++
		} else {
			alive = append(alive, msg)
		}
	}
	m.messages = alive
	return removed
}

// GetMessagesFrom — возвращает сообщения от определённого узла
func (m *Memory) GetMessagesFrom(senderID string) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []Message
	for _, msg := range m.messages {
		if msg.Sender == senderID || msg.ReplicatedFrom == senderID {
			result = append(result, msg)
		}
	}
	return result
}

// GetReplicasFor — возвращает реплики, хранящиеся для определённого узла
func (m *Memory) GetReplicasFor(nodeID string) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []Message
	for _, msg := range m.messages {
		if msg.ReplicatedFrom == nodeID {
			result = append(result, msg)
		}
	}
	return result
}

// ============================================================
// АССОЦИАТИВНАЯ ПАМЯТЬ (кто у кого что спрашивал)
// ============================================================

// Association — запись о том, какой узел к какому обращался
type Association struct {
	FromPeerID string    `json:"fromPeerID"`
	ToPeerID   string    `json:"toPeerID"`
	Topic      string    `json:"topic"`
	Count      int       `json:"count"`
	LastAsked  time.Time `json:"lastAsked"`
}

// AssocMemory — хранилище ассоциаций
type AssocMemory struct {
	mu           sync.Mutex
	associations []Association
}

// AddAssociation — добавляет или обновляет ассоциацию
func (am *AssocMemory) AddAssociation(fromPeerID, toPeerID, topic string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i := range am.associations {
		if am.associations[i].FromPeerID == fromPeerID &&
			am.associations[i].ToPeerID == toPeerID &&
			am.associations[i].Topic == topic {
			am.associations[i].Count++
			am.associations[i].LastAsked = time.Now()
			return
		}
	}

	am.associations = append(am.associations, Association{
		FromPeerID: fromPeerID,
		ToPeerID:   toPeerID,
		Topic:      topic,
		Count:      1,
		LastAsked:  time.Now(),
	})
}

// GetRecommendations — возвращает пиров, к которым этот отправитель часто обращается
func (am *AssocMemory) GetRecommendations(fromPeerID string, minCount int) []string {
	am.mu.Lock()
	defer am.mu.Unlock()

	counts := make(map[string]int)
	for _, a := range am.associations {
		if a.FromPeerID == fromPeerID && a.Count >= minCount {
			counts[a.ToPeerID] += a.Count
		}
	}

	var peers []string
	for peerID, count := range counts {
		if count > 0 {
			peers = append(peers, peerID)
		}
	}
	return peers
}

// GetAllAssociations — возвращает все ассоциации
func (am *AssocMemory) GetAllAssociations() []Association {
	am.mu.Lock()
	defer am.mu.Unlock()
	result := make([]Association, len(am.associations))
	copy(result, am.associations)
	return result
}

// CleanupOldAssociations — удаляет ассоциации старше N дней
func (am *AssocMemory) CleanupOldAssociations(days int) {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	var alive []Association
	for _, a := range am.associations {
		if a.LastAsked.After(cutoff) {
			alive = append(alive, a)
		}
	}
	am.associations = alive
}