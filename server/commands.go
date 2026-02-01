package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"shazam/db"
	waveid "shazam/process"
	"shazam/types"
	"shazam/utils"
	"strings"
	"sync"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

func find(filePath string) {
	wavFilePath, err := utils.ConvertToWAV(filePath)
	if err != nil {
		fmt.Println("Error converting to WAV:", err)
		return
	}

	fingerprint, err := waveid.Fingerprint(wavFilePath, utils.GenerateUniqueID())
	if err != nil {
		fmt.Println("Error generating fingerprint for sample: ", err)
		return
	}

	sampleFingerprint := make(map[uint32]uint32)
	for address, couple := range fingerprint {
		sampleFingerprint[address] = couple.AnchorTimeMs
	}

	matches, searchDuration, err := waveid.FindMatchesFGP(sampleFingerprint)
	if err != nil {
		fmt.Println("Error finding matches:", err)
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

func download(path string) {
	if err := os.MkdirAll(SONGS_DIR, 0755); err != nil {
		panic(err)
	}

	if _, err := os.Stat(path); err == nil {
		downloadFromJSON(path)
		return
	}

	downloadFromYTDLP(path)
}

func downloadFromJSON(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var queries []string
	if err := json.Unmarshal(b, &queries); err != nil {
		panic(err)
	}

	sem := make(chan struct{}, MAX_WORKERS)
	var wg sync.WaitGroup

	for _, q := range queries {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			meta := runYTDLP(query)

			artist := meta.Artist
			if artist == "" {
				artist = meta.Uploader
			}

			process(
				meta.Filename,
				meta.Title,
				artist,
				meta.ID,
			)

			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			p := filepath.Join(wd, strings.TrimSuffix(meta.Filename, filepath.Ext(meta.Filename))+".wav")

			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				panic(err)
			}

		}(q)
	}

	wg.Wait()
}

func runYTDLP(query string) types.YTMeta {
	out := filepath.Join(SONGS_DIR, "%(title)s.%(ext)s")

	cmd := exec.Command(
		"yt-dlp",
		"ytsearch1:"+query,
		"-x",
		"--audio-format", "mp3",
		"--print-json",
		"-o", out,
	)

	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(string(b))
	}

	var m types.YTMeta
	if err := json.Unmarshal(b, &m); err != nil {
		panic(err)
	}

	m.Filename = strings.TrimSpace(m.Filename)
	ext := strings.ToLower(filepath.Ext(m.Filename))
	if ext != ".mp3" {
		m.Filename = strings.TrimSuffix(m.Filename, ext) + ".mp3"
	}

	return m
}

func downloadFromYTDLP(input string) {
	cmd := exec.Command(
		"yt-dlp",
		input,
		"-x",
		"--audio-format", "mp3",
		"-o", filepath.Join(SONGS_DIR, "%(playlist_title)s/%(title)s.%(ext)s"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func process(filePath, songTitle, songArtist, ytID string) {
	dbClient, err := db.DBClient(DB_PATH)
	if err != nil {
		panic(err)
	}
	defer dbClient.Close()

	songID, err := dbClient.RegisterSong(songTitle, songArtist, ytID)
	if err != nil {
		panic(err)
	}
	fingerprint, err := waveid.Fingerprint(filePath, songID)
	if err != nil {
		panic(err)
	}
	err = dbClient.StoreFingerprints(fingerprint)
}

func serve(protocol, port string) {
	protocol = strings.ToLower(protocol)
	var allowOriginFunc = func(r *http.Request) bool {
		return true
	}

	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	})

	server.OnConnect("/", func(socket socketio.Conn) error {
		socket.SetContext("")
		log.Println("CONNECTED: ", socket.ID())

		return nil
	})

	server.OnEvent("/", "totalSongs", handleTotalSongs)
	server.OnEvent("/", "newDownload", handleSongDownload)
	server.OnEvent("/", "newRecording", handleNewRecording)
	server.OnEvent("/", "newFingerprint", handleNewFingerprint)

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed", reason)
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	serveHTTPS := protocol == "https"

	serveHTTP(server, serveHTTPS, port)
}
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func serveHTTP(socketServer *socketio.Server, serveHTTPS bool, port string) {
	mux := http.NewServeMux()
	mux.Handle("/socket.io/", socketServer)
	mux.Handle("/", http.FileServer(http.Dir("static")))

	handler := withCORS(mux)

	if serveHTTPS {
		httpsServer := &http.Server{
			Addr:      ":" + port,
			Handler:   handler,
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		}
		log.Fatal(httpsServer.ListenAndServeTLS(
			"/etc/letsencrypt/live/localport.online/fullchain.pem",
			"/etc/letsencrypt/live/localport.online/privkey.pem",
		))
		return
	}

	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func handleSongDownload(s socketio.Conn, msg string) {
}
