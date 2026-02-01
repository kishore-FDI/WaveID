package waveid

import (
	"fmt"
	"shazam/db"
	"shazam/types"
	"shazam/utils"
	"sort"
	"time"
)

const (
	maxFreqBits    = 9
	maxDeltaBits   = 14
	targetZoneSize = 5
)

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

func Extract(peaks []Peak, songID uint32) map[uint32]types.Couple {
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

func Fingerprint(filePath string, songID uint32) (map[uint32]types.Couple, error) {
	wavFilePath, err := utils.ConvertToWAV(filePath)
	if err != nil {
		return map[uint32]types.Couple{}, fmt.Errorf("WAV conversion failed: %v", err)
	}
	wavInfo, err := utils.ReadWavInfo(wavFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading WAV info: %v", err)
	}

	spectro, err := Spectrogram(wavInfo.LeftChannelSamples, wavInfo.SampleRate)
	if err != nil {
		return nil, fmt.Errorf("error generating spectrogram: %v", err)
	}
	peaks := ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
	return Extract(peaks, songID), nil
}

// FindMatchesFGP uses the sample fingerprint to find matching songs in the database.
func FindMatchesFGP(sampleFingerprint map[uint32]uint32) ([]types.Match, time.Duration, error) {
	startTime := time.Now()
	addresses := make([]uint32, 0, len(sampleFingerprint))
	for address := range sampleFingerprint {
		addresses = append(addresses, address)
	}
	db, err := db.DBClient("shazam.db")
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()
	m, err := db.GetCouples(addresses)
	if err != nil {
		return nil, 0, err
	}
	matches := map[uint32][][2]uint32{}        // songID -> [(sampleTime, dbTime)]
	timestamps := map[uint32]uint32{}          // songID -> earliest timestamp
	targetZones := map[uint32]map[uint32]int{} // songID -> timestamp -> count

	for address, couples := range m {
		for _, couple := range couples {
			matches[couple.SongID] = append(
				matches[couple.SongID],
				[2]uint32{sampleFingerprint[address], couple.AnchorTimeMs},
			)

			if existingTime, ok := timestamps[couple.SongID]; !ok || couple.AnchorTimeMs < existingTime {
				timestamps[couple.SongID] = couple.AnchorTimeMs
			}

			if _, ok := targetZones[couple.SongID]; !ok {
				targetZones[couple.SongID] = make(map[uint32]int)
			}
			targetZones[couple.SongID][couple.AnchorTimeMs]++
		}
	}
	// matches = filterMatches(10, matches, targetZones)

	scores := analyzeRelativeTiming(matches)

	var matchList []types.Match

	for songID, points := range scores {
		song, songExists, err := db.GetSongByID(songID)
		if !songExists {
			continue
		}
		if err != nil {
			continue
		}

		match := types.Match{SongID: songID, SongTitle: song.Title, SongArtist: song.Artist, YouTubeID: song.YouTubeID, Timestamp: timestamps[songID], Score: points}
		matchList = append(matchList, match)
	}

	sort.Slice(matchList, func(i, j int) bool {
		return matchList[i].Score > matchList[j].Score
	})

	return matchList, time.Since(startTime), nil
}

func analyzeRelativeTiming(matches map[uint32][][2]uint32) map[uint32]float64 {
	scores := make(map[uint32]float64)

	for songID, times := range matches {
		offsetCounts := make(map[int32]int)

		for _, timePair := range times {
			sampleTime := int32(timePair[0])
			dbTime := int32(timePair[1])
			offset := dbTime - sampleTime

			// Bin offsets in 100ms buckets to allow for small timing variations
			offsetBucket := offset / 100
			offsetCounts[offsetBucket]++
		}

		maxCount := 0
		for _, count := range offsetCounts {
			if count > maxCount {
				maxCount = count
			}
		}

		scores[songID] = float64(maxCount)
	}

	return scores
}
