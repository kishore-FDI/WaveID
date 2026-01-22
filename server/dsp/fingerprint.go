package dsp

import (
	"fmt"
	types "shazam/servertypes"
	"shazam/wav"
)

const (
	maxFreqBits    = 9
	maxDeltaBits   = 14
	targetZoneSize = 5
)

func createFingerprint(peaks []types.Peak, songID uint32) map[uint32]types.Couple {
	fingerprints := map[uint32]types.Couple{}

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			address := createAddress(anchor, target)
			anchorTimeMs := uint32(anchor.Time * 1000)

			fingerprints[address] = types.Couple{
				AnchorTimeMs: anchorTimeMs,
				SongID:       songID,
			}
		}
	}

	return fingerprints
}

func createAddress(anchor, target types.Peak) uint32 {
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

func FingerPrint(songPath string, songID int64) (map[uint32]types.Couple, error) {
	wavFilePath, err := wav.ConvertToWAV(songPath)
	if err != nil {
		panic(err)
	}
	wavInfo, err := wav.ReadWavInfo(wavFilePath)
	fmt.Print(wavInfo.SampleRate)
	spectro, err := Spectrogram(wavInfo.ChannelSamples, wavInfo.SampleRate)
	if err != nil {
		return nil, err
	}
	peaks := ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
	fingerprints := createFingerprint(peaks, uint32(songID))
	return fingerprints, nil
}
