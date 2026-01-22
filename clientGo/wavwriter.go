package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteWAVFile writes audio samples to a WAV file
func WriteWAVFile(filename string, samples []float64, sampleRate int) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Convert float64 samples to int16
	numSamples := len(samples)
	data := make([]int16, numSamples)
	for i, s := range samples {
		// Clamp to [-1, 1] and convert to int16
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		data[i] = int16(s * 32767.0)
	}

	// WAV file header
	chunkID := [4]byte{'R', 'I', 'F', 'F'}
	format := [4]byte{'W', 'A', 'V', 'E'}
	subchunk1ID := [4]byte{'f', 'm', 't', ' '}
	subchunk2ID := [4]byte{'d', 'a', 't', 'a'}

	subchunk1Size := uint32(16) // PCM format
	audioFormat := uint16(1)     // PCM
	numChannels := uint16(1)     // Mono
	bitsPerSample := uint16(16)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := uint16(numChannels * bitsPerSample / 8)
	subchunk2Size := uint32(numSamples * int(numChannels) * int(bitsPerSample) / 8)
	chunkSize := uint32(36 + subchunk2Size)

	// Write header
	binary.Write(file, binary.LittleEndian, chunkID)
	binary.Write(file, binary.LittleEndian, chunkSize)
	binary.Write(file, binary.LittleEndian, format)
	binary.Write(file, binary.LittleEndian, subchunk1ID)
	binary.Write(file, binary.LittleEndian, subchunk1Size)
	binary.Write(file, binary.LittleEndian, audioFormat)
	binary.Write(file, binary.LittleEndian, numChannels)
	binary.Write(file, binary.LittleEndian, uint32(sampleRate))
	binary.Write(file, binary.LittleEndian, byteRate)
	binary.Write(file, binary.LittleEndian, blockAlign)
	binary.Write(file, binary.LittleEndian, bitsPerSample)
	binary.Write(file, binary.LittleEndian, subchunk2ID)
	binary.Write(file, binary.LittleEndian, subchunk2Size)

	// Write audio data
	binary.Write(file, binary.LittleEndian, data)

	return nil
}

// GenerateRecordingFilename generates a unique filename for the recording
func GenerateRecordingFilename(outputDir string) string {
	timestamp := time.Now().Format("20060102_150405")
	return filepath.Join(outputDir, fmt.Sprintf("recording_%s.wav", timestamp))
}
