package main

import (
	"bytes"
	"testing"
)

// createTestWAV — создаёт синтетический WAV-файл
func createTestWAV(samples int) []byte {
	// WAV header: RIFF, WAVE, fmt, data
	header := []byte{
		'R', 'I', 'F', 'F',
		0, 0, 0, 0, // размер файла (заполним позже)
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0, // размер fmt
		1, 0, // PCM
		1, 0, // моно
		0x40, 0x1F, 0, 0, // 8000 Гц
		0x80, 0x3E, 0, 0, // байт в секунду
		2, 0, // блок
		16, 0, // бит
		'd', 'a', 't', 'a',
		0, 0, 0, 0, // размер данных (заполним)
	}

	dataSize := samples * 2 // 16 бит
	fileSize := 36 + dataSize

	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)

	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	// Генерируем синусоиду
	audioData := make([]byte, dataSize)
	for i := 0; i < samples; i++ {
		sample := int16(10000 * float64(i%100) / 100.0)
		audioData[i*2] = byte(sample)
		audioData[i*2+1] = byte(sample >> 8)
	}

	return append(header, audioData...)
}

func TestEmbedExtractLSB(t *testing.T) {
	wav := createTestWAV(10000) // 10000 сэмплов
	key := []byte("test-stego-key-123")
	message := []byte("секретное сообщение для теста")

	// Встраиваем
	wavWithData, err := embedLSB(wav, message, key)
	if err != nil {
		t.Fatalf("embedLSB failed: %v", err)
	}

	// Извлекаем
	extracted, err := extractLSB(wavWithData, key)
	if err != nil {
		t.Fatalf("extractLSB failed: %v", err)
	}

	if !bytes.Equal(extracted, message) {
		t.Errorf("extracted = %s, want %s", string(extracted), string(message))
	}

	// Проверяем, что WAV остался валидным
	if !isWAV(wavWithData) {
		t.Error("WAV corrupted after embedding")
	}

	t.Log("PASS")
}

func TestEmbedLSB_WrongKey(t *testing.T) {
	wav := createTestWAV(10000)
	key1 := []byte("key-one")
	key2 := []byte("key-two")
	message := []byte("test message")

	wavWithData, err := embedLSB(wav, message, key1)
	if err != nil {
		t.Fatalf("embedLSB failed: %v", err)
	}

	// Попытка извлечь с другим ключом
	_, err = extractLSB(wavWithData, key2)
	if err == nil {
		// Может не быть ошибки, но данные должны быть мусором
		// Проверяем, что данные не равны оригиналу
		extracted, _ := extractLSB(wavWithData, key2)
		if bytes.Equal(extracted, message) {
			t.Error("extracted with wrong key should not match")
		}
	}

	t.Log("PASS")
}

func TestEmbedLSB_TooSmall(t *testing.T) {
	wav := createTestWAV(100) // слишком мало
	key := []byte("test-key")
	message := []byte("это сообщение слишком длинное для такого маленького WAV")

	_, err := embedLSB(wav, message, key)
	if err == nil {
		t.Error("embedLSB should fail for too small WAV")
	}

	t.Log("PASS")
}

func TestWavToBase64(t *testing.T) {
	wav := createTestWAV(100)
	b64 := wavToBase64(wav)

	decoded, err := base64ToWav(b64)
	if err != nil {
		t.Fatalf("base64ToWav failed: %v", err)
	}

	if !bytes.Equal(decoded, wav) {
		t.Error("WAV corrupted after base64 roundtrip")
	}

	t.Log("PASS")
}

func TestIsWAV(t *testing.T) {
	wav := createTestWAV(100)
	if !isWAV(wav) {
		t.Error("valid WAV not recognized")
	}

	if isWAV([]byte("not a wav file")) {
		t.Error("invalid data recognized as WAV")
	}

	t.Log("PASS")
}