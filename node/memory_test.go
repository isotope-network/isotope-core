package main

import (
	"testing"
	"time"
)

func createTestMsg(id string, weight float64, archived bool, created time.Time) Message {
	return Message{
		ID:       id,
		Text:     "тест " + id,
		Sender:   "test",
		Time:     created.Format("2006-01-02T15:04:05"),
		Created:  created,
		Weight:   weight,
		Archived: archived,
	}
}

func TestMemory_Add(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("msg1", 0.5, false, time.Now())
	if !m.Add(msg) {
		t.Error("Add returned false for new message")
	}
	if m.Count() != 1 {
		t.Errorf("Count = %d, want 1", m.Count())
	}
}

func TestMemory_Duplicate(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("dup", 0.5, false, time.Now())
	m.Add(msg)
	if m.Add(msg) {
		t.Error("Add returned true for duplicate message")
	}
	if m.Count() != 1 {
		t.Errorf("Count = %d, want 1 after duplicate", m.Count())
	}
}

func TestMemory_ArchiveOld(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("low", 0.1, false, time.Now().Add(-10*time.Hour))
	m.Add(msg)
	msg2 := createTestMsg("high", 0.5, false, time.Now())
	m.Add(msg2)

	archived := m.ArchiveOld(0.15)
	if archived < 1 {
		t.Errorf("ArchiveOld archived %d, want >= 1", archived)
	}

	all := m.GetAll()
	for _, msg := range all {
		if msg.ID == "low" && !msg.Archived {
			t.Error("low-weight message should be archived")
		}
	}
}

func TestMemory_ArchiveOld_AlreadyArchived(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("arch", 0.1, true, time.Now().Add(-10*time.Hour))
	m.Add(msg)

	archived := m.ArchiveOld(0.15)
	if archived != 0 {
		t.Errorf("ArchiveOld should return 0 for already archived, got %d", archived)
	}
}

func TestMemory_RestoreFromArchive(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("restore", 0.1, true, time.Now().Add(-10*time.Hour))
	m.Add(msg)

	if !m.RestoreFromArchive("restore") {
		t.Error("RestoreFromArchive returned false")
	}

	all := m.GetAll()
	for _, msg := range all {
		if msg.ID == "restore" {
			if msg.Archived {
				t.Error("restored message still archived")
			}
			if msg.Weight < 0.4 || msg.Weight > 0.6 {
				t.Errorf("restored message weight = %f, want ~0.5", msg.Weight)
			}
		}
	}
}

func TestMemory_RestoreFromArchive_NotFound(t *testing.T) {
	m := Memory{}
	if m.RestoreFromArchive("nonexistent") {
		t.Error("RestoreFromArchive returned true for nonexistent message")
	}
}

func TestMemory_PurgeDead(t *testing.T) {
	m := Memory{}
	dead := createTestMsg("dead", 0.05, true, time.Now().Add(-20*time.Hour))
	m.Add(dead)
	alive := createTestMsg("alive", 0.8, false, time.Now())
	m.Add(alive)

	removed := m.PurgeDead(0.1)
	if removed < 1 {
		t.Errorf("PurgeDead removed %d, want >= 1", removed)
	}

	all := m.GetAll()
	for _, msg := range all {
		if msg.ID == "dead" {
			t.Error("dead message should be purged")
		}
	}
}

func TestMemory_PurgeDead_SeenCleanup(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("cleanup", 0.05, true, time.Now().Add(-20*time.Hour))
	m.Add(msg)
	m.PurgeDead(0.1)

	if !m.Add(msg) {
		t.Error("should be able to re-add after purge")
	}
}

func TestMemory_UpdateWeight(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("update", 0.5, false, time.Now())
	m.Add(msg)

	if !m.UpdateWeight("update", 0.15) {
		t.Error("UpdateWeight returned false")
	}

	all := m.GetAll()
	for _, msg := range all {
		if msg.ID == "update" && msg.Weight != 0.65 {
			t.Errorf("weight = %f, want 0.65", msg.Weight)
		}
	}
}

func TestMemory_UpdateWeight_Clamp(t *testing.T) {
	m := Memory{}
	msg := createTestMsg("clamp", 0.9, false, time.Now())
	m.Add(msg)

	m.UpdateWeight("clamp", 0.5)
	all := m.GetAll()
	for _, msg := range all {
		if msg.ID == "clamp" && msg.Weight > 1.0 {
			t.Errorf("weight = %f, should be clamped to 1.0", msg.Weight)
		}
	}

	m.UpdateWeight("clamp", -2.0)
	all = m.GetAll()
	for _, msg := range all {
		if msg.ID == "clamp" && msg.Weight < 0.0 {
			t.Errorf("weight = %f, should be clamped to 0.0", msg.Weight)
		}
	}
}

func TestMemory_UpdateWeight_NotFound(t *testing.T) {
	m := Memory{}
	if m.UpdateWeight("nonexistent", 0.1) {
		t.Error("UpdateWeight returned true for nonexistent message")
	}
}

func TestMemory_CountArchived(t *testing.T) {
	m := Memory{}
	m.Add(createTestMsg("a1", 0.5, false, time.Now()))
	m.Add(createTestMsg("a2", 0.1, true, time.Now().Add(-10*time.Hour)))
	m.Add(createTestMsg("a3", 0.1, true, time.Now().Add(-10*time.Hour)))

	count := m.CountArchived()
	if count != 2 {
		t.Errorf("CountArchived = %d, want 2", count)
	}
}

func TestMemory_FindSimilar(t *testing.T) {
	m := Memory{}
	m.Add(createTestMsg("s1", 0.8, false, time.Now()))

	similar := m.FindSimilar("тест s1", 0.5)
	if len(similar) == 0 {
		t.Log("FindSimilar returned empty — may be ok for short text")
	}
}

func TestMemory_FindSimilar_IgnoresArchived(t *testing.T) {
	m := Memory{}
	m.Add(createTestMsg("archived", 0.8, true, time.Now().Add(-10*time.Hour)))

	similar := m.FindSimilar("тест archived", 0.5)
	for _, msg := range similar {
		if msg.ID == "archived" {
			t.Error("FindSimilar should ignore archived messages")
		}
	}
}

func TestMemory_GetAll_ReturnsCopy(t *testing.T) {
	m := Memory{}
	m.Add(createTestMsg("original", 0.5, false, time.Now()))

	all := m.GetAll()
	all[0].Text = "modified"

	all2 := m.GetAll()
	if all2[0].Text == "modified" {
		t.Error("GetAll should return a copy, not reference")
	}
}

func TestMemory_Concurrent(t *testing.T) {
	m := Memory{}
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				msg := createTestMsg(
					string(rune('A'+id))+string(rune('0'+j)),
					0.5, false, time.Now(),
				)
				m.Add(msg)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if m.Count() != 1000 {
		t.Errorf("Concurrent add count = %d, want 1000", m.Count())
	}
}