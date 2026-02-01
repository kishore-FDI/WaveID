package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"shazam/db"
	waveid "shazam/process"
	"shazam/types"
	"shazam/utils"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

func downloadStatus(statusType, message string) string {
	data := map[string]interface{}{"type": statusType, "message": message}
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal download status", slog.String("error", err.Error()))
		return `{"type":"error","message":"Internal error"}`
	}
	return string(jsonData)
}

func handleTotalSongs(socket socketio.Conn) {
	ctx := context.Background()

	db, err := db.DBClient(DB_PATH)
	if err != nil {
		return
	}
	defer db.Close()

	totalSongs, err := db.TotalSongs()
	if err != nil {
		slog.ErrorContext(ctx, "Log error getting total songs", slog.Any("error", err))
		return
	}

	socket.Emit("totalSongs", totalSongs)
}

// handleNewRecording saves new recorded audio snippet to a WAV file.
func handleNewRecording(conn socketio.Conn, recordData string) {

	var recData types.RecordData
	if err := json.Unmarshal([]byte(recordData), &recData); err != nil {
		return
	}

	err := utils.CreateFolder("recordings")
	if err != nil {
	}

	now := time.Now()
	fileName := fmt.Sprintf("%04d_%02d_%02d_%02d_%02d_%02d.wav",
		now.Second(), now.Minute(), now.Hour(),
		now.Day(), now.Month(), now.Year(),
	)
	filePath := "recordings/" + fileName

	decodedAudioData, err := base64.StdEncoding.DecodeString(recData.Audio)
	if err != nil {
	}

	err = utils.WriteWavFile(filePath, decodedAudioData, recData.SampleRate, recData.Channels, recData.SampleSize)
	if err != nil {
	}
}

func handleNewFingerprint(socket socketio.Conn, fingerprintData string) {
	var data struct {
		Fingerprint map[uint32]uint32 `json:"fingerprint"`
	}
	if err := json.Unmarshal([]byte(fingerprintData), &data); err != nil {
		slog.Error("Failed to unmarshal fingerprint", "error", err)
		return
	}

	matches, _, err := waveid.FindMatchesFGP(data.Fingerprint)
	if err != nil {
		slog.Error("Error finding matches", "error", err)
	}

	if len(matches) == 0 {
		slog.Info("No matches found")
	} else {
		slog.Info("Matches found", "count", len(matches))
	}

	jsonData, err := json.Marshal(matches)
	if len(matches) > 10 {
		jsonData, _ = json.Marshal(matches[:10])
	}

	if err != nil {
		slog.Error("Failed to marshal matches", "error", err)
		return
	}

	socket.Emit("matches", string(jsonData))
}
