package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"shazam/dsp"
	types "shazam/servertypes"
	"shazam/utils"
	"shazam/wav"
	"time"
)

func setupHTTPServer(recordingsDir string) {

	if err := utils.MkDir(recordingsDir); err != nil {
		panic(fmt.Sprintf("Failed to create recordings directory: %v", err))
	}

	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	uploadHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		timestamp := time.Now().Format("20060102_150405")
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".webm"
		}
		filename := fmt.Sprintf("recording_%s%s", timestamp, ext)
		filepath := filepath.Join(recordingsDir, filename)

		dst, err := os.Create(filepath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating file: %v", err), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving file: %v", err), http.StatusInternalServerError)
			return
		}

		matches, searchDuration, findErr := findAndReturnResults(filepath)

		response := map[string]interface{}{
			"success":  true,
			"filename": filename,
			"message":  "Audio saved successfully",
		}

		if findErr != nil {
			response["findError"] = findErr.Error()
			response["matches"] = []interface{}{}
		} else {

			matchList := make([]map[string]interface{}, 0, len(matches))
			for _, match := range matches {
				matchList = append(matchList, map[string]interface{}{
					"songId":     match.SongID,
					"songTitle":  match.SongTitle,
					"songArtist": match.SongArtist,
					"youtubeId":  match.YouTubeID,
					"timestamp":  match.Timestamp,
					"score":      match.Score,
				})
			}
			response["matches"] = matchList
			response["searchDuration"] = searchDuration.String()
			response["matchCount"] = len(matches)

			if len(matches) > 0 {
				topMatch := matches[0]
				response["topMatch"] = map[string]interface{}{
					"songTitle":  topMatch.SongTitle,
					"songArtist": topMatch.SongArtist,
					"score":      topMatch.Score,
					"timestamp":  topMatch.Timestamp,
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	http.Handle("/api/upload-audio", corsMiddleware(uploadHandler))

	fmt.Println("HTTP Server started on :8080")
	fmt.Println("Upload endpoint: http://localhost:8080/api/upload-audio")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}
}

func findAndReturnResults(filePath string) ([]types.Match, time.Duration, error) {

	waveFilePath, err := wav.ConvertToWAV(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("error converting to WAV: %v", err)
	}

	fingerprint, err := dsp.FingerPrint(waveFilePath, int64(utils.GenerateUniqueID()))
	if err != nil {
		return nil, 0, fmt.Errorf("error generating fingerprint: %v", err)
	}

	sampleFingerprint := make(map[uint32]uint32)
	for address, couple := range fingerprint {
		sampleFingerprint[address] = couple.AnchorTimeMs
	}

	matches, searchDuration, err := dsp.FindMatchesFGP(sampleFingerprint)
	if err != nil {
		return nil, 0, fmt.Errorf("error finding matches: %v", err)
	}

	return matches, searchDuration, nil
}
