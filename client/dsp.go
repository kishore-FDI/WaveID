package main

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
)

const (
	targetSampleRate = 11025          // Target frequency for spectrogram
	windowSize       = 1024           // Size of each FFT window
	maxFreq          = 5000           // Maximum frequency to consider
	hopSize          = windowSize / 2 // Hop size between windows
	maxFreqBits      = 9
	maxDeltaBits     = 14
	targetZoneSize   = 5
)

// LowPassFilter is a first-order low-pass filter that attenuates high
// frequencies above the cutoffFrequency.
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
	spectrogram := make([][]float64, 0)
	for start := 0; start+windowSize <= len(downSampledOutput); start += hopSize {
		end := start + windowSize

		frame := make([]float64, windowSize)
		copy(frame, downSampledOutput[start:end])

		for i := range frame {
			frame[i] *= window[i]
		}

		fftResult := fft.FFTReal(frame)

		// Convert complex spectrum to magnitude spectrum
		magnitude := make([]float64, len(fftResult)/2)
		for j := range magnitude {
			magnitude[j] = cmplx.Abs(fftResult[j])
		}

		spectrogram = append(spectrogram, magnitude)
	}
	return spectrogram, nil
}

// ExtractPeaks analyzes a spectrogram and extracts significant peaks in the frequency domain over time.
func ExtractPeaks(spectrogram [][]float64, audioDuration float64, sampleRate int) []Peak {
	if len(spectrogram) < 1 {
		return []Peak{}
	}

	type maxies struct {
		maxMag  float64
		freqIdx int
	}

	bands := []struct{ min, max int }{
		{0, 10}, {10, 20}, {20, 40}, {40, 80}, {80, 160}, {160, 512},
	}

	var peaks []Peak
	frameDuration := audioDuration / float64(len(spectrogram))

	// Calculate frequency resolution (Hz per bin)
	effectiveSampleRate := float64(targetSampleRate)
	freqResolution := effectiveSampleRate / float64(windowSize)

	for frameIdx, frame := range spectrogram {
		var maxMags []float64
		var freqIndices []int

		binBandMaxies := []maxies{}
		for _, band := range bands {
			var maxx maxies
			var maxMag float64
			for idx, mag := range frame[band.min:band.max] {
				if mag > maxMag {
					maxMag = mag
					freqIdx := band.min + idx
					maxx = maxies{mag, freqIdx}
				}
			}
			binBandMaxies = append(binBandMaxies, maxx)
		}

		for _, value := range binBandMaxies {
			maxMags = append(maxMags, value.maxMag)
			freqIndices = append(freqIndices, value.freqIdx)
		}

		// Calculate the average magnitude
		var maxMagsSum float64
		for _, max := range maxMags {
			maxMagsSum += max
		}
		avg := maxMagsSum / float64(len(maxMags))

		// Add peaks that exceed the average magnitude
		for i, value := range maxMags {
			if value > avg {
				peakTime := float64(frameIdx) * frameDuration
				peakFreq := float64(freqIndices[i]) * freqResolution

				peaks = append(peaks, Peak{Time: peakTime, Freq: peakFreq})
			}
		}
	}

	return peaks
}

func createAddress(anchor, target Peak) uint32 {
	anchorFreqBin := uint32(anchor.Freq / 10) // Scale down to fit in 9 bits
	targetFreqBin := uint32(target.Freq / 10)

	deltaMsRaw := uint32((target.Time - anchor.Time) * 1000)

	// Mask to fit within bit constraints
	anchorFreqBits := anchorFreqBin & ((1 << maxFreqBits) - 1) // 9 bits
	targetFreqBits := targetFreqBin & ((1 << maxFreqBits) - 1) // 9 bits
	deltaBits := deltaMsRaw & ((1 << maxDeltaBits) - 1)        // 14 bits (max ~16 seconds)

	// Combine into 32-bit address
	address := (anchorFreqBits << 23) | (targetFreqBits << 14) | deltaBits

	return address
}

func CreateFingerprints(peaks []Peak) map[uint32]bool {
	fingerprints := make(map[uint32]bool)

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]
			address := createAddress(anchor, target)
			fingerprints[address] = true
		}
	}

	return fingerprints
}
