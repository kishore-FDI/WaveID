package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"shazam/dsp"
	"shazam/utils"
	"shazam/wav"
	types "shazam/servertypes"
	"time"
)

func setupHTTPServer(recordingsDir string) {
	// Ensure recordings directory exists
	if err := utils.MkDir(recordingsDir); err != nil {
		panic(fmt.Sprintf("Failed to create recordings directory: %v", err))
	}

	// CORS middleware
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

	// Upload audio endpoint handler
	uploadHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse multipart form (10MB max)
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
			return
		}

		// Get the audio file from the form
		file, header, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Generate filename with timestamp
		timestamp := time.Now().Format("20060102_150405")
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".webm" // Default extension for web audio
		}
		filename := fmt.Sprintf("recording_%s%s", timestamp, ext)
		filepath := filepath.Join(recordingsDir, filename)

		// Create the file
		dst, err := os.Create(filepath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating file: %v", err), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the destination
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving file: %v", err), http.StatusInternalServerError)
			return
		}

		// Call find() on the recorded audio
		matches, searchDuration, findErr := findAndReturnResults(filepath)
		
		// Prepare response
		response := map[string]interface{}{
			"success": true,
			"filename": filename,
			"message": "Audio saved successfully",
		}

		if findErr != nil {
			response["findError"] = findErr.Error()
			response["matches"] = []interface{}{}
		} else {
			// Convert matches to JSON-serializable format
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
				}
			}
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Apply CORS middleware and register handler
	http.Handle("/api/upload-audio", corsMiddleware(uploadHandler))

	fmt.Println("HTTP Server started on :8080")
	fmt.Println("Upload endpoint: http://localhost:8080/api/upload-audio")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}
}

// findAndReturnResults performs the find operation and returns results
func findAndReturnResults(filePath string) ([]types.Match, time.Duration, error) {
	// Convert to WAV
	waveFilePath, err := wav.ConvertToWAV(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("error converting to WAV: %v", err)
	}

	// Generate fingerprint
	fingerprint, err := dsp.FingerPrint(waveFilePath, int64(utils.GenerateUniqueID()))
	if err != nil {
		return nil, 0, fmt.Errorf("error generating fingerprint: %v", err)
	}

	// Convert fingerprint to sample format
	sampleFingerprint := make(map[uint32]uint32)
	for address, couple := range fingerprint {
		sampleFingerprint[address] = couple.AnchorTimeMs
	}

	// Find matches
	matches, searchDuration, err := dsp.FindMatchesFGP(sampleFingerprint)
	if err != nil {
		return nil, 0, fmt.Errorf("error finding matches: %v", err)
	}

	return matches, searchDuration, nil
}
