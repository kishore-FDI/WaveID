package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"shazam/db"
	"shazam/dsp"
	"shazam/utils"
	"strings"
)

func DownloadSongs(jsonPath string, client *db.SQLiteClient) {
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		panic(err)
	}

	var songs []utils.Song
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
		fmt.Print(fingerprints)
	}

}
