package main

import (
	"encoding/json"
	"os"
)

// ============================================================
// СОСТОЯНИЕ УЗЛА (СОХРАНЕНИЕ И ЗАГРУЗКА)
// ============================================================

// State — структура, которая сохраняется на диск.
type State struct {
	Layers   [][]float64     `json:"layers"`
	MsgCount int             `json:"msgCount"`
	Messages []Message       `json:"messages"`
	Seen     map[string]bool `json:"seen"`
	PreHash  string          `json:"preHash"`
	AntiHash string          `json:"antiHash"`
}

// saveState — сохраняет состояние узла в файл.
func (n *Node) saveState() error {
	if n.stateFile == "" {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	seen := n.memory.seen
	if seen == nil {
		seen = make(map[string]bool)
	}

	state := State{
		Layers:   n.layers,
		MsgCount: n.msgCount,
		Messages: n.memory.GetAll(),
		Seen:     seen,
		PreHash:  n.preHash,
		AntiHash: n.antiHash,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll("state", 0755); err != nil {
		return err
	}

	return os.WriteFile(n.stateFile, data, 0644)
}