package main

import (
	"sync"
	"time"
)

// Channel — канал с весовыми уровнями доступа
type Channel struct {
	mu               sync.Mutex
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Owner            string    `json:"owner"`
	Messages         []Message `json:"messages"`
	CreatedAt        time.Time `json:"createdAt"`
	MinFullWeight    float64   `json:"minFullWeight"`
	MinCommentWeight float64   `json:"minCommentWeight"`
	MinVoteWeight    float64   `json:"minVoteWeight"`
}

// NewChannel — создаёт канал с дефолтными порогами
func NewChannel(id, name, owner string) *Channel {
	return &Channel{
		ID:               id,
		Name:             name,
		Owner:            owner,
		Messages:         make([]Message, 0),
		CreatedAt:        time.Now(),
		MinFullWeight:    0.3,
		MinCommentWeight: 0.5,
		MinVoteWeight:    0.7,
	}
}

// AddMessage — добавляет сообщение в канал
func (c *Channel) AddMessage(msg Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, msg)
}

// GetVisibleMessages — возвращает сообщения в зависимости от веса
func (c *Channel) GetVisibleMessages(weight float64) []Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if weight >= c.MinFullWeight {
		return c.Messages
	}

	// Тизер: 30% последних сообщений
	count := len(c.Messages) * 3 / 10
	if count < 1 {
		count = 1
	}
	if count > len(c.Messages) {
		count = len(c.Messages)
	}
	return c.Messages[len(c.Messages)-count:]
}

// CanComment — проверяет право комментирования
func (c *Channel) CanComment(weight float64) bool {
	return weight >= c.MinCommentWeight
}

// CanVote — проверяет право голосования
func (c *Channel) CanVote(weight float64) bool {
	return weight >= c.MinVoteWeight
}

// ChannelStore — хранилище каналов
type ChannelStore struct {
	mu       sync.Mutex
	channels map[string]*Channel
}

// NewChannelStore — создаёт хранилище
func NewChannelStore() *ChannelStore {
	return &ChannelStore{
		channels: make(map[string]*Channel),
	}
}

// Create — создаёт канал
func (cs *ChannelStore) Create(id, name, owner string) *Channel {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	ch := NewChannel(id, name, owner)
	cs.channels[id] = ch
	return ch
}

// Get — возвращает канал по ID
func (cs *ChannelStore) Get(id string) *Channel {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.channels[id]
}

// GetAll — возвращает все каналы
func (cs *ChannelStore) GetAll() []*Channel {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	result := make([]*Channel, 0, len(cs.channels))
	for _, ch := range cs.channels {
		result = append(result, ch)
	}
	return result
}