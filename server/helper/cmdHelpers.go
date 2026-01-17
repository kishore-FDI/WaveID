package helper

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"shazam/db"
	"shazam/shazamalgo"
	"shazam/utils"
)

type SpotifyPlaylist struct {
	Tracks struct {
		Items []struct {
			Track struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"track"`
		} `json:"items"`
	} `json:"tracks"`
}

func DownloadSpotifyMetadata(input string) {
	must(os.MkdirAll("audios", 0755))

	var pl SpotifyPlaylist

	if isJSON(input) {
		b, err := os.ReadFile(input)
		must(err)
		must(json.Unmarshal(b, &pl))
	} else {
		tok := os.Getenv("SPOTIFY_ACCESS_TOKEN")
		if tok == "" {
			utils.Log.Fatal("SPOTIFY_ACCESS_TOKEN missing")
		}

		id := playlistID(input)
		req, _ := http.NewRequest(
			"GET",
			"https://api.spotify.com/v1/playlists/"+id,
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := http.DefaultClient.Do(req)
		must(err)
		defer resp.Body.Close()
		must(json.NewDecoder(resp.Body).Decode(&pl))
	}

	if len(pl.Tracks.Items) == 0 {
		utils.Log.Fatal("playlist empty")
	}

	d := db.Open()
	defer d.Close()

	for _, it := range pl.Tracks.Items {
		if it.Track.Name == "" || len(it.Track.Artists) == 0 {
			continue
		}

		title := it.Track.Name
		artist := it.Track.Artists[0].Name

		if d.SongExists(title, artist) {
			continue
		}

		downloadYT(title, artist, d)
	}
}

func downloadYT(title, artist string, d *db.DB) {
	bin := "yt-dlp"
	if filepath.Separator == '\\' {
		bin = "yt-dlp.exe"
	}

	q := artist + " " + title
	outTpl := "audios/%(title)s.%(ext)s"

	cmd := exec.Command(
		bin,
		"-x",
		"--audio-format",
		"mp3",
		"--print-json",
		"-o",
		outTpl,
		"ytsearch1:"+q,
	)
	// cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	must(err)

	var meta struct {
		Title    string  `json:"title"`
		Duration float64 `json:"duration"`
		Ext      string  `json:"ext"`
	}
	must(json.Unmarshal(out, &meta))

	path := filepath.Join("audios", meta.Title+".mp3")
	processedPath := filepath.Join("processedAudios", "temp.wav")
	convertAudio(path, processedPath)
	songID := d.InsertSong(
		title,
		artist,
		path,
		meta.Duration,
	)
	shazamalgo.FingerPrint(processedPath, songID, d)
	finalizeProcessedAudio(songID)
}

func convertAudio(inputPath, outputPath string) {
	bin := "ffmpeg"
	if filepath.Separator == '\\' {
		bin = "ffmpeg.exe"
	}

	cmd := exec.Command(
		bin,
		"-i", inputPath,
		"-ac", "1",
		"-ar", "11050",
		outputPath,
	)
	// cmd.Stderr = os.Stderr
	must(cmd.Run())
}

func finalizeProcessedAudio(songID int64) {
	tempPath := filepath.Join("processedAudios", "temp.wav")
	newPath := filepath.Join("processedAudios", strconv.FormatInt(songID, 10)+".wav")
	must(os.Rename(tempPath, newPath))
}

func isJSON(p string) bool {
	return strings.HasSuffix(strings.ToLower(p), ".json")
}

func playlistID(u string) string {
	i := strings.LastIndex(u, "/")
	if i == -1 {
		utils.Log.Fatal("invalid spotify url")
	}
	id := u[i+1:]
	if j := strings.IndexByte(id, '?'); j != -1 {
		id = id[:j]
	}
	return id
}

func must(err error) {
	if err != nil {
		utils.Log.Fatal("%v", err)
	}
}
