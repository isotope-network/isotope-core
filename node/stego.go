package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math/rand"
)

// ============================================================
// ГОЛОСОВАЯ СТЕГАНОГРАФИЯ (LSB + случайное распределение)
// ============================================================

// wavHeaderSize — размер заголовка WAV (44 байта для PCM)
const wavHeaderSize = 44

// getStegoSeed — возвращает seed для случайного распределения из ключа
func getStegoSeed(key []byte) int64 {
	hash := sha256.Sum256(key)
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

// embedLSB — встраивает данные в младшие биты аудио-сэмплов WAV
func embedLSB(wavBytes []byte, data []byte, key []byte) ([]byte, error) {
	if len(wavBytes) < wavHeaderSize {
		return nil, errors.New("WAV file too small")
	}

	audioData := wavBytes[wavHeaderSize:]

	// Данные + размер (4 байта заголовка)
	dataWithSize := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(dataWithSize[:4], uint32(len(data)))
	copy(dataWithSize[4:], data)

	bitCount := len(dataWithSize) * 8
	if bitCount > len(audioData) {
		return nil, errors.New("WAV file too small for data")
	}

	result := make([]byte, len(wavBytes))
	copy(result, wavBytes)
	resultAudio := result[wavHeaderSize:]

	seed := getStegoSeed(key)
	rng := rand.New(rand.NewSource(seed))

	indices := make([]int, len(audioData))
	for i := range indices {
		indices[i] = i
	}
	rng.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	for bitIdx := 0; bitIdx < bitCount; bitIdx++ {
		byteIdx := bitIdx / 8
		bitPos := bitIdx % 8
		bitVal := (dataWithSize[byteIdx] >> bitPos) & 1

		sampleIdx := indices[bitIdx%len(indices)]
		if bitVal == 1 {
			resultAudio[sampleIdx] |= 1
		} else {
			resultAudio[sampleIdx] &^= 1
		}
	}

	return result, nil
}

// extractLSB — извлекает данные из младших битов аудио-сэмплов WAV
func extractLSB(wavBytes []byte, key []byte) ([]byte, error) {
	if len(wavBytes) < wavHeaderSize {
		return nil, errors.New("WAV file too small")
	}

	audioData := wavBytes[wavHeaderSize:]

	seed := getStegoSeed(key)
	rng := rand.New(rand.NewSource(seed))

	indices := make([]int, len(audioData))
	for i := range indices {
		indices[i] = i
	}
	rng.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	// Извлекаем 4 байта размера
	sizeBytes := make([]byte, 4)
	for i := 0; i < 32; i++ {
		byteIdx := i / 8
		bitPos := i % 8
		sampleIdx := indices[i]
		bitVal := audioData[sampleIdx] & 1
		if bitVal == 1 {
			sizeBytes[byteIdx] |= 1 << bitPos
		}
	}

	dataLen := binary.BigEndian.Uint32(sizeBytes)
	if dataLen > uint32(len(audioData)/8) {
		return nil, errors.New("invalid data length in stego")
	}

	result := make([]byte, dataLen)
	totalBits := int(dataLen) * 8
	for i := 0; i < totalBits; i++ {
		byteIdx := i / 8
		bitPos := i % 8
		sampleIdx := indices[32+i]
		bitVal := audioData[sampleIdx] & 1
		if bitVal == 1 {
			result[byteIdx] |= 1 << bitPos
		}
	}

	return result, nil
}

// wavToBase64 — WAV в base64
func wavToBase64(wavBytes []byte) string {
	return base64.StdEncoding.EncodeToString(wavBytes)
}

// base64ToWav — base64 в WAV
func base64ToWav(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}

// isWAV — проверяет, что байты — валидный WAV
func isWAV(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	return bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WAVE"))
}