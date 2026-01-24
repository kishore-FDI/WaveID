package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"shazam/db"
	"shazam/dsp"
	types "shazam/servertypes"
	"shazam/utils"
	"shazam/wav"
	"strings"
	"sync"

	"github.com/fatih/color"
)

var yellow = color.New(color.FgYellow)

type ProcessingResult struct {
	Song         types.Song
	Fingerprints map[uint32]types.Couple
	Error        error
}

func DownloadSongs(jsonPath string, client *db.SQLiteClient) {
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		panic(err)
	}

	var songs []types.Song
	if err := json.Unmarshal(b, &songs); err != nil {
		panic(err)
	}

	outDir := "SONGS_DIR"

	results := make(chan ProcessingResult, len(songs))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		dbWriter(client, results)
	}()

	numWorkers := 4
	semaphore := make(chan struct{}, numWorkers)

	for i := range songs {
		wg.Add(1)
		go func(song types.Song) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := processSong(song, outDir)
			results <- result
		}(songs[i])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	wg.Wait()
}

func processSong(song types.Song, outDir string) ProcessingResult {
	result := ProcessingResult{Song: song}

	if song.YtID == "" {
		q := song.Title + " " + song.Artist
		cmd := exec.Command("yt-dlp", "ytsearch1:"+q, "--get-id")
		idb, err := cmd.Output()
		if err != nil {
			result.Error = fmt.Errorf("error getting YouTube ID for %s: %v", q, err)
			return result
		}
		song.YtID = strings.TrimSpace(string(idb))
		result.Song.YtID = song.YtID
	}

	url := "https://www.youtube.com/watch?v=" + song.YtID
	out := filepath.Join(outDir, "%(title)s.%(ext)s")

	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"-o", out,
		"--print", "after_move:filepath",
		url,
	)
	output, err := cmd.Output()
	if err != nil {
		result.Error = fmt.Errorf("error downloading song %s: %v", song.Title, err)
		return result
	}

	songPath := strings.TrimSpace(string(output))
	result.Song.SongPath = songPath

	fingerprints, err := dsp.FingerPrint(songPath, 0)
	if err != nil {
		result.Error = fmt.Errorf("error generating fingerprints for %s: %v", song.Title, err)
		return result
	}

	result.Fingerprints = fingerprints
	return result
}

func dbWriter(client *db.SQLiteClient, results <-chan ProcessingResult) {
	for result := range results {
		if result.Error != nil {
			yellow.Printf("Error processing song %s: %v\n", result.Song.Title, result.Error)
			continue
		}

		songID, err := client.AddSong(result.Song)
		if err != nil {
			yellow.Printf("Error adding song %s to database: %v\n", result.Song.Title, err)
			continue
		}

		updatedFingerprints := make(map[uint32]types.Couple)
		for address, couple := range result.Fingerprints {
			couple.SongID = uint32(songID)
			updatedFingerprints[address] = couple
		}

		err = client.AddFingerPrints(updatedFingerprints)
		if err != nil {
			yellow.Printf("Error adding fingerprints for song %s: %v\n", result.Song.Title, err)
			continue
		}

		fmt.Printf("Successfully processed song: %s by %s\n", result.Song.Title, result.Song.Artist)
	}
}

func find(filePath string) {
	waveFilePath, err := wav.ConvertToWAV(filePath)
	if err != nil {
		yellow.Println("Error converting to WAV:", err)
		return
	}
	fingerprint, err := dsp.FingerPrint(waveFilePath, int64(utils.GenerateUniqueID()))
	if err != nil {
		yellow.Println("Error generating fingerprint for sample: ", err)
		return
	}
	sampleFingerprint := make(map[uint32]uint32)
	for address, couple := range fingerprint {
		sampleFingerprint[address] = couple.AnchorTimeMs
	}
	matches, searchDuration, err := dsp.FindMatchesFGP(sampleFingerprint)

	if err != nil {
		yellow.Println("Error finding matches:", err)
		return
	}

	if len(matches) == 0 {
		fmt.Println("\nNo match found.")
		fmt.Printf("\nSearch took: %s\n", searchDuration)
		return
	}

	msg := "Matches:"
	topMatches := matches
	if len(matches) >= 20 {
		msg = "Top 20 matches:"
		topMatches = matches[:20]
	}

	fmt.Println(msg)
	for _, match := range topMatches {
		fmt.Printf("\t- %s by %s, score: %.2f\n",
			match.SongTitle, match.SongArtist, match.Score)
	}

	fmt.Printf("\nSearch took: %s\n", searchDuration)
	topMatch := topMatches[0]
	fmt.Printf("\nFinal prediction: %s by %s , score: %.2f\n",
		topMatch.SongTitle, topMatch.SongArtist, topMatch.Score)
}
