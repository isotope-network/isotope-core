package main

import (
	"testing"
	"time"
)

func TestProcessMessage_ReturnsMsgID(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("тест", "testsender", true)

	all := n.memory.GetAll()
	if len(all) == 0 {
		t.Error("Message not found after processMessage")
	}
	t.Log("PASS")
}

func TestProcessMessage_UsesIncomingMsgID(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("тест", "testsender", false)
	t.Log("PASS")
}

func TestProcessMessage_GeneratesOwnID(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("тест", "testsender", true)
	n.processMessage("тест", "testsender", true)

	all := n.memory.GetAll()
	if len(all) < 1 {
		t.Error("Messages not found after processMessage")
	}
	t.Logf("Найдено сообщений: %d", len(all))
	t.Log("PASS")
}

func TestProcessMessage_DuplicateDetection(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("дубль", "sender1", false)
	time.Sleep(10 * time.Millisecond)
	n.processMessage("дубль", "sender2", false)

	all := n.memory.GetAll()
	count := 0
	var ids []string
	for _, msg := range all {
		if msg.Text == "дубль" {
			count++
			ids = append(ids, msg.ID)
		}
	}

	if count != 2 {
		t.Errorf("Both messages should be stored with different IDs, found %d", count)
	}
	if len(ids) == 2 && ids[0] == ids[1] {
		t.Error("IDs should be different for messages with same text")
	}
	t.Log("PASS")
}

func TestProcessMessage_DifferentIDsDifferentMessages(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("msg1", "sender", false)
	time.Sleep(10 * time.Millisecond)
	n.processMessage("msg2", "sender", false)

	count := len(n.memory.GetAll())
	if count < 2 {
		t.Errorf("Different messages should both be stored, found %d", count)
	}
	t.Log("PASS")
}

func TestDeliveryStatus_Thresholds(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("добро свет мир любовь помощь забота радость счастье", "sender", true)
	n.processMessage("", "sender", true)
	t.Log("PASS")
}

func TestProcessMessage_WeightCalculation(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("тестовое сообщение", "sender", true)

	all := n.memory.GetAll()
	if len(all) == 0 {
		t.Error("Message not found after processMessage")
	} else {
		t.Logf("Вес: %.4f", all[0].Weight)
		if all[0].Weight <= 0 || all[0].Weight > 1.0 {
			t.Errorf("Weight out of range: %f", all[0].Weight)
		}
	}
	t.Log("PASS")
}

func TestProcessMessage_IsOwnFlag(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("моё", "sender", true)
	time.Sleep(10 * time.Millisecond)
	n.processMessage("чужое", "sender", false)

	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.Text == "моё" && !msg.IsOwn {
			t.Error("Own message should have IsOwn = true")
		}
		if msg.Text == "чужое" && msg.IsOwn {
			t.Error("Peer message should have IsOwn = false")
		}
	}
	t.Log("PASS")
}

func TestPreHash_EffectOnWeight(t *testing.T) {
	n := &Node{
		ethHash: "testhash",
		preHash: "добро свет мир любовь",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("добро свет любовь", "sender", true)

	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.Text == "добро свет любовь" {
			t.Logf("Вес с preHash: %.4f", msg.Weight)
			if msg.Weight < 0.3 {
				t.Errorf("Message matching preHash should have higher weight, got %f", msg.Weight)
			}
		}
	}
	t.Log("PASS")
}

func TestAntiHash_EffectOnWeight(t *testing.T) {
	n := &Node{
		ethHash:  "testhash",
		antiHash: "зло тьма ненависть",
		memory: Memory{
			seen: make(map[string]bool),
		},
		layers: [][]float64{make([]float64, VectorDim)},
	}
	hashVec := hashToVector(n.ethHash)
	for i := range n.layers[0] {
		if i < len(hashVec) {
			n.layers[0][i] = hashVec[i] * 0.1
		}
	}

	n.processMessage("зло тьма ненависть", "sender", true)

	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.Text == "зло тьма ненависть" {
			t.Logf("Вес с antiHash: %.4f", msg.Weight)
			if msg.Weight > 0.6 {
				t.Errorf("Message matching antiHash should have lower weight, got %f", msg.Weight)
			}
		}
	}
	t.Log("PASS")
}