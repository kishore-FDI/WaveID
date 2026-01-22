package main

// resampleAudio resamples audio from one sample rate to another using linear interpolation
func resampleAudio(samples []float64, fromRate, toRate int) []float64 {
	if fromRate == toRate {
		return samples
	}

	ratio := float64(fromRate) / float64(toRate)
	newLength := int(float64(len(samples)) / ratio)
	resampled := make([]float64, newLength)

	for i := 0; i < newLength; i++ {
		// Calculate the position in the original sample array
		pos := float64(i) * ratio
		index := int(pos)
		frac := pos - float64(index)

		if index+1 < len(samples) {
			// Linear interpolation
			resampled[i] = samples[index]*(1-frac) + samples[index+1]*frac
		} else if index < len(samples) {
			// Last sample, no interpolation
			resampled[i] = samples[index]
		} else {
			// Beyond array, use last sample
			resampled[i] = samples[len(samples)-1]
		}
	}

	return resampled
}
