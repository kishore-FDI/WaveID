package main

import (
	"fmt"
	"math"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate = 44100
	channels   = 1
	bufferSize = 4096
)

type AudioRecorder struct {
	stream   *portaudio.Stream
	buffer   []float32
	samples  []float64
	recording bool
}

func NewAudioRecorder() (*AudioRecorder, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize portaudio: %v", err)
	}

	recorder := &AudioRecorder{
		buffer: make([]float32, bufferSize),
		samples: make([]float64, 0),
	}

	stream, err := portaudio.OpenDefaultStream(
		channels,        // input channels
		0,               // output channels
		float64(sampleRate),
		bufferSize,      // frames per buffer
		recorder.buffer,
	)
	if err != nil {
		portaudio.Terminate()
		return nil, fmt.Errorf("failed to open audio stream: %v", err)
	}

	recorder.stream = stream
	return recorder, nil
}

func (r *AudioRecorder) Start() error {
	r.recording = true
	r.samples = make([]float64, 0)
	return r.stream.Start()
}

func (r *AudioRecorder) Stop() error {
	r.recording = false
	if r.stream != nil {
		return r.stream.Stop()
	}
	return nil
}

func (r *AudioRecorder) Close() error {
	if r.stream != nil {
		r.stream.Close()
	}
	portaudio.Terminate()
	return nil
}

// RecordForDuration records audio for a specified duration in seconds
func (r *AudioRecorder) RecordForDuration(duration float64) ([]float64, error) {
	if err := r.Start(); err != nil {
		return nil, err
	}
	defer r.Stop()

	totalSamples := int(sampleRate * duration)
	samplesCollected := 0

	for samplesCollected < totalSamples {
		err := r.stream.Read()
		if err != nil {
			return nil, err
		}

		// Convert float32 to float64
		remaining := totalSamples - samplesCollected
		toAdd := len(r.buffer)
		if toAdd > remaining {
			toAdd = remaining
		}

		for i := 0; i < toAdd; i++ {
			r.samples = append(r.samples, float64(r.buffer[i]))
		}

		samplesCollected += toAdd
	}

	return r.samples, nil
}

// NormalizeAudio normalizes audio samples to [-1, 1] range
func NormalizeAudio(samples []float64) []float64 {
	if len(samples) == 0 {
		return samples
	}

	// Find max absolute value
	maxVal := 0.0
	for _, s := range samples {
		absVal := math.Abs(s)
		if absVal > maxVal {
			maxVal = absVal
		}
	}

	// Normalize if max value is greater than 1.0
	if maxVal > 1.0 && maxVal > 0 {
		normalized := make([]float64, len(samples))
		for i, s := range samples {
			normalized[i] = s / maxVal
		}
		return normalized
	}

	return samples
}
