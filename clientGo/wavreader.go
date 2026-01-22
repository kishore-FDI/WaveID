package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WavInfo contains information about a WAV file
type WavInfo struct {
	Channels       int
	SampleRate     int
	Duration       float64
	Data           []byte
	ChannelSamples []float64
}

// WavHeader represents the WAV file header structure
type WavHeader struct {
	ChunkID       [4]byte
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte
	Subchunk2Size uint32
}

// ConvertToWAV converts an audio file to WAV format using ffmpeg
func ConvertToWAV(inputFilePath string) (wavFilePath string, err error) {
	_, err = os.Stat(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("input file does not exist: %v", err)
	}

	fileExt := filepath.Ext(inputFilePath)
	if fileExt == ".wav" {
		// Already a WAV file, return as-is
		return inputFilePath, nil
	}

	// Create temporary output file
	outputFile := strings.TrimSuffix(inputFilePath, fileExt) + "_converted.wav"
	tmpFile := filepath.Join(filepath.Dir(outputFile), "tmp_"+filepath.Base(outputFile))
	defer os.Remove(tmpFile)

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", inputFilePath,
		"-c", "pcm_s16le",
		"-ar", "44100",
		"-ac", "1",
		tmpFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert to WAV: %v, output %v", err, string(output))
	}

	err = os.Rename(tmpFile, outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to rename temporary file to output file: %v", err)
	}

	return outputFile, nil
}

// ReadWavInfo reads a WAV file and returns its information
func ReadWavInfo(filename string) (*WavInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if len(data) < 44 {
		return nil, errors.New("invalid WAV file size (too small)")
	}

	var header WavHeader
	if err := binary.Read(bytes.NewReader(data[:44]), binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if string(header.ChunkID[:]) != "RIFF" ||
		string(header.Format[:]) != "WAVE" ||
		header.AudioFormat != 1 {
		return nil, errors.New("invalid WAV header format")
	}
	if header.NumChannels != 1 {
		return nil, errors.New("unsupported channel count (expect mono)")
	}
	if header.BitsPerSample != 16 {
		return nil, errors.New("unsupported bits-per-sample (expect 16-bit PCM)")
	}

	info := &WavInfo{
		Channels:   1,
		SampleRate: int(header.SampleRate),
		Data:       data[44:],
	}

	sampleCount := len(info.Data) / 2
	int16Buf := make([]int16, sampleCount)
	if err := binary.Read(bytes.NewReader(info.Data), binary.LittleEndian, int16Buf); err != nil {
		return nil, err
	}

	const scale = 1.0 / 32768.0
	samples := make([]float64, sampleCount)
	for i, s := range int16Buf {
		samples[i] = float64(s) * scale
	}
	info.ChannelSamples = samples

	info.Duration = float64(sampleCount) / float64(header.SampleRate)
	return info, nil
}

// ExtractRandomSegment extracts a random segment of specified duration from audio samples
func ExtractRandomSegment(samples []float64, sampleRate int, segmentDuration float64) ([]float64, float64, error) {
	totalDuration := float64(len(samples)) / float64(sampleRate)
	
	if totalDuration < segmentDuration {
		// If the file is shorter than requested segment, return all samples
		return samples, totalDuration, nil
	}

	// Calculate the maximum start position to ensure we can extract the full segment
	maxStartTime := totalDuration - segmentDuration
	
	// Generate random start time
	rand.Seed(time.Now().UnixNano())
	startTime := rand.Float64() * maxStartTime
	
	// Calculate sample indices
	startSample := int(startTime * float64(sampleRate))
	endSample := startSample + int(segmentDuration*float64(sampleRate))
	
	if endSample > len(samples) {
		endSample = len(samples)
	}
	
	segment := samples[startSample:endSample]
	actualDuration := float64(len(segment)) / float64(sampleRate)
	
	return segment, actualDuration, nil
}
