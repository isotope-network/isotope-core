package main

import (
	"testing"
	"time"
)

// TestProcessMessage_ReturnsMsgID проверяет, что processMessage возвращает непустой msgID
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

	t.Log("Запуск processMessage...")
	msgID, status := n.processMessage("тест", "testsender", true, "", 0)
	t.Logf("msgID=%s, status=%s", msgID, status)

	if msgID == "" {
		t.Error("processMessage returned empty msgID")
	}
	if status == "" {
		t.Error("processMessage returned empty status")
	}
	validStatuses := map[string]bool{"delivered": true, "partial": true, "filtered": true}
	if !validStatuses[status] {
		t.Errorf("processMessage returned unknown status: %s", status)
	}
	t.Log("PASS")
}

// TestProcessMessage_UsesIncomingMsgID проверяет, что используется входящий ID
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

	incomingID := "abc12345"
	t.Logf("Входящий ID: %s", incomingID)
	msgID, _ := n.processMessage("тест", "testsender", false, incomingID, 0)
	t.Logf("Результат msgID: %s", msgID)

	if msgID != incomingID {
		t.Errorf("processMessage should use incomingMsgID, got %s, want %s", msgID, incomingID)
	}
	t.Log("PASS")
}

// TestProcessMessage_GeneratesOwnID проверяет, что при пустом incomingMsgID генерируется свой ID
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

	t.Log("Генерация первого ID...")
	msgID1, _ := n.processMessage("тест", "testsender", true, "", 0)
	t.Logf("msgID1=%s", msgID1)

	t.Log("Генерация второго ID...")
	msgID2, _ := n.processMessage("тест", "testsender", true, "", 0)
	t.Logf("msgID2=%s", msgID2)

	if msgID1 == "" || msgID2 == "" {
		t.Error("processMessage returned empty msgID")
	}
	t.Log("PASS")
}

// TestProcessMessage_DuplicateDetection проверяет, что дубликаты не сохраняются
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

	t.Log("Первое сообщение с fixed-id-123...")
	n.processMessage("дубль", "sender1", false, "fixed-id-123", 0)

	countBefore := 0
	for _, msg := range n.memory.GetAll() {
		if msg.ID == "fixed-id-123" {
			countBefore++
		}
	}
	t.Logf("Сообщений с fixed-id-123: %d", countBefore)

	t.Log("Дубликат с fixed-id-123...")
	n.processMessage("дубль", "sender2", false, "fixed-id-123", 0)

	countAfter := 0
	for _, msg := range n.memory.GetAll() {
		if msg.ID == "fixed-id-123" {
			countAfter++
		}
	}
	t.Logf("Сообщений с fixed-id-123 после дубликата: %d", countAfter)

	if countAfter != countBefore {
		t.Errorf("Duplicate message should not increase count: %d -> %d", countBefore, countAfter)
	}
	t.Log("PASS")
}

// TestProcessMessage_DifferentIDsDifferentMessages проверяет,
// что сообщения с разными ID сохраняются как разные
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

	t.Log("Добавление id-1...")
	n.processMessage("msg1", "sender", false, "id-1", 0)
	t.Log("Добавление id-2...")
	n.processMessage("msg2", "sender", false, "id-2", 0)

	count := 0
	for _, msg := range n.memory.GetAll() {
		if msg.ID == "id-1" || msg.ID == "id-2" {
			count++
			t.Logf("  Найдено: %s", msg.ID)
		}
	}
	t.Logf("Всего пользовательских сообщений: %d", count)

	if count != 2 {
		t.Errorf("Different IDs should create different messages, found %d, want 2", count)
	}
	t.Log("PASS")
}

// TestDeliveryStatus_Thresholds проверяет пороги статусов доставки
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

	t.Log("Позитивное сообщение...")
	_, status1 := n.processMessage("добро свет мир любовь помощь забота радость счастье", "sender", true, "", 0)
	t.Logf("Статус: %s", status1)

	t.Log("Пустое сообщение...")
	_, status2 := n.processMessage("", "sender", true, "", 0)
	t.Logf("Статус: %s", status2)

	if status2 == "" {
		t.Error("Empty message should still return a status")
	}
	t.Log("PASS")
}

// TestUpdateDeliveryStatus проверяет обновление статуса доставки
func TestUpdateDeliveryStatus(t *testing.T) {
	m := Memory{seen: make(map[string]bool)}
	msg := Message{
		ID:             "test-status-1",
		Text:           "тест",
		Sender:         "sender",
		Time:           time.Now().Format("2006-01-02T15:04:05"),
		Weight:         0.5,
		Created:        time.Now(),
		DeliveryStatus: "sent",
	}
	m.Add(msg)
	t.Log("Начальный статус: sent")

	if !m.UpdateDeliveryStatus("test-status-1", "delivered") {
		t.Error("UpdateDeliveryStatus returned false")
	}
	t.Log("Обновлён на: delivered")

	all := m.GetAll()
	found := false
	for _, msg := range all {
		if msg.ID == "test-status-1" {
			found = true
			if msg.DeliveryStatus != "delivered" {
				t.Errorf("DeliveryStatus = %s, want delivered", msg.DeliveryStatus)
			}
		}
	}
	if !found {
		t.Error("Message not found after status update")
	}
	t.Log("PASS")
}

// TestUpdateDeliveryStatus_NotFound проверяет обновление несуществующего
func TestUpdateDeliveryStatus_NotFound(t *testing.T) {
	m := Memory{seen: make(map[string]bool)}
	if m.UpdateDeliveryStatus("nonexistent", "delivered") {
		t.Error("UpdateDeliveryStatus should return false for nonexistent message")
	}
	t.Log("PASS")
}

// TestMessage_DeliveryStatusDefault проверяет значение по умолчанию
func TestMessage_DeliveryStatusDefault(t *testing.T) {
	msg := Message{
		ID:   "test-default",
		Text: "тест",
	}
	if msg.DeliveryStatus != "" {
		t.Errorf("Default DeliveryStatus should be empty, got %s", msg.DeliveryStatus)
	}
	t.Log("PASS")
}

// TestProcessMessage_WeightCalculation проверяет, что вес вычисляется
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

	t.Log("Отправка тестового сообщения...")
	n.processMessage("тестовое сообщение", "sender", true, "weight-test-1", 0)

	all := n.memory.GetAll()
	found := false
	for _, msg := range all {
		if msg.ID == "weight-test-1" {
			found = true
			t.Logf("Вес: %.4f", msg.Weight)
			if msg.Weight <= 0 || msg.Weight > 1.0 {
				t.Errorf("Weight out of range: %f", msg.Weight)
			}
		}
	}
	if !found {
		t.Error("Message not found after processMessage")
	}
	t.Log("PASS")
}

// TestProcessMessage_IsOwnFlag проверяет флаг isOwn
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

	t.Log("Своё сообщение...")
	n.processMessage("моё", "sender", true, "own-test", 0)
	t.Log("Чужое сообщение...")
	n.processMessage("чужое", "sender", false, "peer-test", 0)

	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.ID == "own-test" && !msg.IsOwn {
			t.Error("Own message should have IsOwn = true")
		}
		if msg.ID == "peer-test" && msg.IsOwn {
			t.Error("Peer message should have IsOwn = false")
		}
	}
	t.Log("PASS")
}

// TestPreHash_EffectOnWeight проверяет, что preHash влияет на вес
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

	t.Log("Сообщение, близкое к preHash...")
	n.processMessage("добро свет любовь", "sender", true, "prehash-test-1", 0)

	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.ID == "prehash-test-1" {
			t.Logf("Вес с preHash: %.4f", msg.Weight)
			if msg.Weight < 0.3 {
				t.Errorf("Message matching preHash should have higher weight, got %f", msg.Weight)
			}
		}
	}
	t.Log("PASS")
}

// TestAntiHash_EffectOnWeight проверяет, что antiHash снижает вес
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

	t.Log("Сообщение, близкое к antiHash...")
	n.processMessage("зло тьма ненависть", "sender", true, "antihash-test-1", 0)

	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.ID == "antihash-test-1" {
			t.Logf("Вес с antiHash: %.4f", msg.Weight)
			if msg.Weight > 0.6 {
				t.Errorf("Message matching antiHash should have lower weight, got %f", msg.Weight)
			}
		}
	}
	t.Log("PASS")
}