package dsp

import (
	"fmt"
	"path/filepath"
	"shazam/db"
	types "shazam/servertypes"
	"sort"
	"time"
)

const (
	// Minimum score threshold to consider a match valid
	// This helps filter out false positives from weak matches
	// A score of 15 means at least 15 fingerprints matched in the same time offset bucket
	minScoreThreshold = 15.0
	// Bucket size in milliseconds for offset binning
	offsetBucketSize = 100
)

func analyzeRelativeTiming(matches map[uint32][][2]uint32) map[uint32]float64 {
	scores := make(map[uint32]float64)

	for songID, times := range matches {
		if len(times) == 0 {
			continue
		}

		offsetCounts := make(map[int32]int)
		totalMatches := len(times)

		for _, timePair := range times {
			sampleTime := int32(timePair[0])
			dbTime := int32(timePair[1])
			offset := dbTime - sampleTime

			// Bin offsets in 100ms buckets to allow for small timing variations
			offsetBucket := offset / offsetBucketSize
			offsetCounts[offsetBucket]++
		}

		// Find the maximum count in any single bucket (most consistent offset)
		maxCount := 0
		for _, count := range offsetCounts {
			if count > maxCount {
				maxCount = count
			}
		}

		// Improved scoring: combine consistency (maxCount) with total matches
		// This gives higher scores to songs with both consistent timing AND more total matches
		// Formula: base score from maxCount, with bonus for total matches
		consistencyScore := float64(maxCount)
		totalMatchBonus := float64(totalMatches) * 0.1 // Small bonus for more total matches
		
		// The score is primarily based on consistency, but total matches help break ties
		scores[songID] = consistencyScore + totalMatchBonus
	}

	return scores
}

func FindMatchesFGP(sampleFingerprint map[uint32]uint32) ([]types.Match, time.Duration, error) {
	startTime := time.Now()
	addresses := make([]uint32, 0, len(sampleFingerprint))
	for address := range sampleFingerprint {
		addresses = append(addresses, address)
	}
	dbPath := filepath.Join("PROCESSED_DIR", "shazam.db")
	db, err := db.NewSQLiteClient(dbPath)
	if err != nil {
		return nil, time.Since(startTime), err
	}
	defer db.Close()
	m, err := db.GetCouples(addresses)
	if err != nil {
		return nil, time.Since(startTime), err
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
	scores := analyzeRelativeTiming(matches)

	var matchList []types.Match
	orphanedCount := 0

	// Process scores and filter out invalid song IDs and weak matches
	for songID, points := range scores {
		// Filter out matches below minimum threshold to reduce false positives
		if points < minScoreThreshold {
			continue
		}

		song, songExists, err := db.GetSongByID(songID)
		if err != nil {
			// Database error - log and skip
			fmt.Printf("Database error for song ID (%d): %v\n", songID, err)
			continue
		}
		if !songExists {
			// Song doesn't exist (orphaned fingerprint) - log and skip
			orphanedCount++
			fmt.Printf("song with ID (%d) doesn't exist\n", songID)
			continue
		}

		match := types.Match{
			SongID:     songID,
			SongTitle:  song.Title,
			SongArtist: song.Artist,
			YouTubeID:  song.YtID,
			Timestamp:  timestamps[songID],
			Score:      points,
		}
		matchList = append(matchList, match)
	}

	// Log orphaned fingerprints count if any were found
	if orphanedCount > 0 {
		fmt.Printf("Warning: Found %d orphaned fingerprints (songs that don't exist). Consider running cleanup.\n", orphanedCount)
	}

	sort.Slice(matchList, func(i, j int) bool {
		return matchList[i].Score > matchList[j].Score
	})

	return matchList, time.Since(startTime), nil
}
