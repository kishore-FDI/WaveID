package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	recordDuration  = 5.0  // seconds
	segmentDuration = 10.0 // seconds for file processing
	dbPath          = "../server/PROCESSED_DIR/shazam.db"
	minMatchPercent = 5.0          // Minimum match percentage to consider it a valid match
	minMatchCount   = 10           // Minimum number of matching fingerprints
	recordingsDir   = "recordings" // Directory to store recorded audio files
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	mode := os.Args[1]

	// Initialize database client
	dbPathAbs, err := filepath.Abs(dbPath)
	if err != nil {
		fmt.Printf("Error resolving database path: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Shazam Client - Real-time Song Recognition")
	fmt.Println("==========================================")
	fmt.Printf("Connecting to database at: %s\n", dbPathAbs)

	dbClient, err := NewDBClient(dbPath)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		fmt.Println("Make sure the server has processed songs and the database exists.")
		os.Exit(1)
	}
	defer dbClient.Close()

	var samples []float64
	var duration float64
	var sourceInfo string

	switch mode {
	case "1":
		// Real-time recording mode
		samples, duration, sourceInfo = recordFromMicrophone()
	case "2":
		// File processing mode
		if len(os.Args) < 3 {
			fmt.Println("Error: File path required for mode 2")
			fmt.Println("Usage: go run . 2 <audio_file_path>")
			os.Exit(1)
		}
		audioPath := os.Args[2]
		samples, duration, sourceInfo = processAudioFile(audioPath)
	default:
		fmt.Printf("Error: Unknown mode '%s'\n", mode)
		printUsage()
		os.Exit(1)
	}

	// Process and query
	processAndQuery(dbClient, samples, duration, sourceInfo)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run . 1                    - Record from microphone (real-time)")
	fmt.Println("  go run . 2 <audio_file_path>  - Process audio file (random 10-second segment)")
}

func recordFromMicrophone() ([]float64, float64, string) {
	fmt.Println("\n=== Mode 1: Real-time Recording ===")

	// Initialize audio recorder
	fmt.Println("Initializing audio recorder...")
	recorder, err := NewAudioRecorder()
	if err != nil {
		fmt.Printf("Error initializing audio recorder: %v\n", err)
		fmt.Println("Make sure you have audio input devices available.")
		os.Exit(1)
	}
	defer recorder.Close()

	fmt.Printf("\nRecording audio for %.1f seconds...\n", recordDuration)
	fmt.Println("Please play the song you want to identify...")
	fmt.Println("Recording started...")

	// Record audio
	samples, err := recorder.RecordForDuration(recordDuration)
	if err != nil {
		fmt.Printf("Error recording audio: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Recorded %d samples\n", len(samples))

	// Normalize audio
	samples = NormalizeAudio(samples)

	// Save recorded audio to file
	recordingFile := GenerateRecordingFilename(recordingsDir)
	fmt.Printf("Saving recording to: %s\n", recordingFile)
	if err := WriteWAVFile(recordingFile, samples, sampleRate); err != nil {
		fmt.Printf("Warning: Failed to save recording: %v\n", err)
	} else {
		fmt.Printf("Recording saved successfully\n")
	}

	duration := float64(len(samples)) / float64(sampleRate)
	return samples, duration, "microphone recording"
}

func processAudioFile(audioPath string) ([]float64, float64, string) {
	fmt.Println("\n=== Mode 2: Audio File Processing ===")
	fmt.Printf("Processing file: %s\n", audioPath)

	// Convert to WAV if needed
	fmt.Println("Converting to WAV format (if needed)...")
	wavPath, err := ConvertToWAV(audioPath)
	if err != nil {
		fmt.Printf("Error converting audio file: %v\n", err)
		fmt.Println("Make sure ffmpeg is installed and the file is a valid audio file.")
		os.Exit(1)
	}

	// Clean up converted file if it was created
	if wavPath != audioPath {
		defer os.Remove(wavPath)
	}

	// Read WAV file
	fmt.Println("Reading audio file...")
	wavInfo, err := ReadWavInfo(wavPath)
	if err != nil {
		fmt.Printf("Error reading WAV file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File duration: %.2f seconds\n", wavInfo.Duration)
	fmt.Printf("Sample rate: %d Hz\n", wavInfo.SampleRate)

	// Extract random 10-second segment
	fmt.Printf("Extracting random %.1f-second segment...\n", segmentDuration)
	segment, actualDuration, err := ExtractRandomSegment(wavInfo.ChannelSamples, wavInfo.SampleRate, segmentDuration)
	if err != nil {
		fmt.Printf("Error extracting segment: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Extracted segment: %.2f seconds (%d samples)\n", actualDuration, len(segment))

	// Resample to 44100 Hz if needed (for consistency with recording mode)
	if wavInfo.SampleRate != sampleRate {
		fmt.Printf("Resampling from %d Hz to %d Hz...\n", wavInfo.SampleRate, sampleRate)
		segment = resampleAudio(segment, wavInfo.SampleRate, sampleRate)
		actualDuration = float64(len(segment)) / float64(sampleRate)
	}

	// Normalize audio
	segment = NormalizeAudio(segment)

	return segment, actualDuration, fmt.Sprintf("file: %s (%.1fs segment)", audioPath, actualDuration)
}

func processAndQuery(dbClient *DBClient, samples []float64, duration float64, sourceInfo string) {
	// Process audio to create fingerprints
	fmt.Println("\nProcessing audio and creating fingerprints...")
	// Use sampleRate constant (44100) for processing, as all audio is resampled to this rate
	spectro, err := Spectrogram(samples, sampleRate)
	if err != nil {
		fmt.Printf("Error creating spectrogram: %v\n", err)
		os.Exit(1)
	}

	peaks := ExtractPeaks(spectro, duration, sampleRate)
	fmt.Printf("Extracted %d peaks\n", len(peaks))

	fingerprints := CreateFingerprints(peaks)
	fmt.Printf("Created %d fingerprints\n", len(fingerprints))

	// Query database
	fmt.Println("\nQuerying database for matching song...")
	result, err := dbClient.QuerySong(fingerprints)
	if err != nil {
		fmt.Printf("Error querying database: %v\n", err)
		fmt.Println("No matching song found. Try again or make sure the song is in the database.")
		os.Exit(1)
	}

	// Check if match score is high enough
	if result.MatchPercent < minMatchPercent || result.MatchCount < minMatchCount {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("NO MATCH FOUND")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Source: %s\n", sourceInfo)
		fmt.Printf("Match Score: %.2f%% (%d/%d fingerprints matched)\n",
			result.MatchPercent, result.MatchCount, result.TotalQueried)
		fmt.Printf("Threshold: %.2f%% minimum match percentage and %d minimum matches required\n",
			minMatchPercent, minMatchCount)
		fmt.Println("\nThe audio did not match any song in the database with sufficient confidence.")
		fmt.Println(strings.Repeat("=", 50))
		os.Exit(0)
	}

	// Display result
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("MATCH FOUND!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Source: %s\n", sourceInfo)
	fmt.Printf("Title:  %s\n", result.Song.Title)
	fmt.Printf("Artist: %s\n", result.Song.Artist)
	if result.Song.YtID != "" {
		fmt.Printf("YouTube ID: %s\n", result.Song.YtID)
	}
	fmt.Printf("\nMatch Score: %.2f%% (%d/%d fingerprints matched)\n",
		result.MatchPercent, result.MatchCount, result.TotalQueried)
	fmt.Println(strings.Repeat("=", 50))
}
