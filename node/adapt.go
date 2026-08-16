package main

import (
	"log"
	"sync"
	"time"
)

// ============================================================
// САМОАДАПТАЦИЯ СЕТИ
// ============================================================

// AdaptiveParams — параметры, которые сеть регулирует сама
type AdaptiveParams struct {
	mu               sync.Mutex
	ArchiveThreshold float64   `json:"archiveThreshold"`
	LearningRate     float64   `json:"learningRate"`
	SyncInterval     time.Duration `json:"-"`
	LastAdapted      time.Time `json:"lastAdapted"`
}

// NewAdaptiveParams — начальные значения
func NewAdaptiveParams() *AdaptiveParams {
	return &AdaptiveParams{
		ArchiveThreshold: 0.15,
		LearningRate:     0.01,
		SyncInterval:     30 * time.Second,
		LastAdapted:      time.Now(),
	}
}

// CollectMetrics — собирает метрики для адаптации
func (n *Node) CollectMetrics() map[string]float64 {
	msgs := n.memory.GetAll()
	if len(msgs) == 0 {
		return map[string]float64{
			"avgWeight":      0.5,
			"lowWeightRatio": 0,
			"highWeightRatio": 0,
		}
	}

	totalWeight := 0.0
	lowCount := 0
	highCount := 0
	for _, m := range msgs {
		totalWeight += m.Weight
		if m.Weight < 0.3 {
			lowCount++
		}
		if m.Weight > 0.7 {
			highCount++
		}
	}

	return map[string]float64{
		"avgWeight":       totalWeight / float64(len(msgs)),
		"lowWeightRatio":  float64(lowCount) / float64(len(msgs)),
		"highWeightRatio": float64(highCount) / float64(len(msgs)),
	}
}

// Adapt — применяет правила автокоррекции
func (n *Node) Adapt(metrics map[string]float64) {
	if n.adaptive == nil {
		return
	}

	n.adaptive.mu.Lock()
	defer n.adaptive.mu.Unlock()

	changed := false

	// Правило 1: средний вес < 0.3 → снизить порог архивации
	if metrics["avgWeight"] < 0.3 && n.adaptive.ArchiveThreshold > 0.05 {
		old := n.adaptive.ArchiveThreshold
		n.adaptive.ArchiveThreshold -= 0.01
		log.Printf("[ADAPT] archiveThreshold: %.2f → %.2f (avgWeight=%.2f)", old, n.adaptive.ArchiveThreshold, metrics["avgWeight"])
		changed = true
	}

	// Правило 2: средний вес > 0.7 → повысить порог архивации
	if metrics["avgWeight"] > 0.7 && n.adaptive.ArchiveThreshold < 0.30 {
		old := n.adaptive.ArchiveThreshold
		n.adaptive.ArchiveThreshold += 0.01
		log.Printf("[ADAPT] archiveThreshold: %.2f → %.2f (avgWeight=%.2f)", old, n.adaptive.ArchiveThreshold, metrics["avgWeight"])
		changed = true
	}

	// Правило 3: много низковесных → ускорить очистку
	if metrics["lowWeightRatio"] > 0.3 && n.adaptive.SyncInterval > 5*time.Second {
		old := n.adaptive.SyncInterval
		n.adaptive.SyncInterval -= 5 * time.Second
		log.Printf("[ADAPT] syncInterval: %v → %v (lowRatio=%.2f)", old, n.adaptive.SyncInterval, metrics["lowWeightRatio"])
		changed = true
	}

	// Правило 4: мало сообщений → замедлить синхронизацию
	if metrics["lowWeightRatio"] < 0.1 && n.adaptive.SyncInterval < 300*time.Second {
		old := n.adaptive.SyncInterval
		n.adaptive.SyncInterval += 5 * time.Second
		log.Printf("[ADAPT] syncInterval: %v → %v (lowRatio=%.2f)", old, n.adaptive.SyncInterval, metrics["lowWeightRatio"])
		changed = true
	}

	if changed {
		n.adaptive.LastAdapted = time.Now()
		n.saveState()
	}
}

// StartAdaptation — фоновая адаптация каждые 5 минут
func (n *Node) StartAdaptation() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			metrics := n.CollectMetrics()
			n.Adapt(metrics)
		}
	}()
}