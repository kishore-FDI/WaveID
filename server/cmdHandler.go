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

	"github.com/fatih/color"
)

var yellow = color.New(color.FgYellow)

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

	for i := range songs {
		if songs[i].YtID == "" {
			q := songs[i].Title + " " + songs[i].Artist
			cmd := exec.Command("yt-dlp", "ytsearch1:"+q, "--get-id")
			idb, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			songs[i].YtID = strings.TrimSpace(string(idb))
		}

		url := "https://www.youtube.com/watch?v=" + songs[i].YtID
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
			panic(err)
		}

		// Get the actual file path from yt-dlp output
		songs[i].SongPath = strings.TrimSpace(string(output))

		songID, err := client.AddSong(songs[i])
		if err != nil {
			panic(err)
		}
		fingerprints, err := dsp.FingerPrint(songs[i].SongPath, songID)
		dbErr := client.AddFingerPrints(fingerprints)
		if dbErr != nil {
			panic(dbErr)
		}
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
