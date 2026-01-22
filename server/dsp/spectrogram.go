package dsp

import (
	"errors"
	"fmt"
	"math"
)

const (
	targetSampleRate = 11025          // Target frequency for spectrogram
	windowSize       = 1024           // Size of each FFT window
	maxFreq          = 5000           // Maximum frequency to consider
	hopSize          = windowSize / 2 // Hop size between windows
)

// LowPassFilter is a first-order low-pass filter that attenuates high
// frequencies above the cutoffFrequency.
// It uses the transfer function H(s) = 1 / (1 + sRC), where RC is the time constant.
func lowPassFilter(input []float64, sampleRate float64) []float64 {
	rc := 1.0 / (2 * math.Pi * maxFreq)
	dt := 1.0 / sampleRate
	alpha := dt / (rc + dt)

	filteredSignal := make([]float64, len(input))
	var prevOutput float64 = 0

	for i, x := range input {
		if i == 0 {
			filteredSignal[i] = x * alpha
		} else {

			filteredSignal[i] = alpha*x + (1-alpha)*prevOutput
		}
		prevOutput = filteredSignal[i]
	}
	return filteredSignal
}

func downSample(input []float64, originalSampleRate int) ([]float64, error) {
	if targetSampleRate <= 0 || originalSampleRate <= 0 {
		return nil, errors.New("sample rates must be positive")
	}
	if targetSampleRate > originalSampleRate {
		return nil, errors.New("target sample rate must be less than or equal to original sample rate")
	}

	ratio := originalSampleRate / targetSampleRate
	if ratio <= 0 {
		return nil, errors.New("invalid ratio calculated from sample rates")
	}
	var resampled []float64
	for i := 0; i < len(input); i += ratio {
		end := i + ratio
		if end > len(input) {
			end = len(input)
		}
		sum := 0.0
		for j := i; j < end; j++ {
			sum += input[j]
		}
		avg := sum / float64(end-i)
		resampled = append(resampled, avg)
	}
	return resampled, nil
}

func Spectrogram(sample []float64, sampleRate int) ([][]float64, error) {
	filteredSamples := lowPassFilter(sample, float64(sampleRate))
	downSampledOutput, err := downSample(filteredSamples, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("couldn't downsample audio sample: %v", err)
	}
	window := make([]float64, windowSize)
	for i := range window {
		window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(windowSize-1))
	}
	fmt.Print(downSampledOutput)
	return nil, nil
}
