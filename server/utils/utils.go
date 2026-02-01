package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"shazam/types"
	"strings"
)

func WriteWavFile(filename string, data []byte, sampleRate int, channels int, bitsPerSample int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if sampleRate <= 0 || channels <= 0 || bitsPerSample <= 0 {
		return fmt.Errorf(
			"values must be greater than zero (sampleRate: %d, channels: %d, bitsPerSample: %d)",
			sampleRate, channels, bitsPerSample,
		)
	}

	err = writeWavHeader(f, data, sampleRate, channels, bitsPerSample)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}

func writeWavHeader(f *os.File, data []byte, sampleRate int, channels int, bitsPerSample int) error {
	// Validate input
	if len(data)%channels != 0 {
		return errors.New("data size not divisible by channels")
	}

	// Calculate derived values
	subchunk1Size := uint32(16) // Assuming PCM format
	bytesPerSample := bitsPerSample / 8
	blockAlign := uint16(channels * bytesPerSample)
	subchunk2Size := uint32(len(data))

	// Build WAV header
	header := types.WavHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     uint32(36 + len(data)),
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: subchunk1Size,
		AudioFormat:   uint16(1), // PCM format
		NumChannels:   uint16(channels),
		SampleRate:    uint32(sampleRate),
		BytesPerSec:   uint32(sampleRate * channels * bytesPerSample),
		BlockAlign:    blockAlign,
		BitsPerSample: uint16(bitsPerSample),
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: subchunk2Size,
	}

	// Write header to file
	err := binary.Write(f, binary.LittleEndian, header)
	return err
}

func CreateFolder(folderName string) error {
	err := os.MkdirAll(folderName, 0755)
	return err
}

func GenerateUniqueID() uint32 {
	return rand.Uint32()

}

func ReadWavInfo(filename string) (*types.WavInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 {
		return nil, errors.New("invalid wav")
	}

	r := bytes.NewReader(data)

	var riff [4]byte
	var size uint32
	var wave [4]byte

	binary.Read(r, binary.LittleEndian, &riff)
	binary.Read(r, binary.LittleEndian, &size)
	binary.Read(r, binary.LittleEndian, &wave)

	if string(riff[:]) != "RIFF" || string(wave[:]) != "WAVE" {
		return nil, errors.New("invalid wav header")
	}

	var fmtFound bool
	var dataFound bool
	var hdr types.WavHeader
	var raw []byte

	for r.Len() > 8 {
		var id [4]byte
		var sz uint32

		binary.Read(r, binary.LittleEndian, &id)
		binary.Read(r, binary.LittleEndian, &sz)

		switch string(id[:]) {
		case "fmt ":
			if sz < 16 {
				return nil, errors.New("invalid fmt chunk")
			}

			var audioFormat uint16
			var numChannels uint16
			var sampleRate uint32
			var byteRate uint32
			var blockAlign uint16
			var bitsPerSample uint16

			binary.Read(r, binary.LittleEndian, &audioFormat)
			binary.Read(r, binary.LittleEndian, &numChannels)
			binary.Read(r, binary.LittleEndian, &sampleRate)
			binary.Read(r, binary.LittleEndian, &byteRate)
			binary.Read(r, binary.LittleEndian, &blockAlign)
			binary.Read(r, binary.LittleEndian, &bitsPerSample)

			if sz > 16 {
				r.Seek(int64(sz-16), io.SeekCurrent)
			}

			hdr.AudioFormat = audioFormat
			hdr.NumChannels = numChannels
			hdr.SampleRate = sampleRate
			hdr.BitsPerSample = bitsPerSample

			fmtFound = true

		case "data":
			raw = make([]byte, sz)
			if _, err := io.ReadFull(r, raw); err != nil {
				return nil, err
			}
			dataFound = true

		default:
			r.Seek(int64(sz), io.SeekCurrent)
		}

		if sz%2 == 1 {
			r.Seek(1, io.SeekCurrent)
		}
	}

	if !fmtFound || !dataFound {
		return nil, errors.New("missing fmt or data chunk")
	}

	if hdr.AudioFormat != 1 || hdr.NumChannels != 1 || hdr.BitsPerSample != 16 {
		return nil, errors.New("unsupported wav format")
	}

	n := len(raw) / 2
	buf := make([]int16, n)
	binary.Read(bytes.NewReader(raw), binary.LittleEndian, &buf)

	samples := make([]float64, n)
	for i, v := range buf {
		samples[i] = float64(v) / 32768.0
	}

	return &types.WavInfo{
		Channels:           1,
		SampleRate:         int(hdr.SampleRate),
		LeftChannelSamples: samples,
		Data:               raw,
		Duration:           float64(n) / float64(hdr.SampleRate),
	}, nil
}

// ConvertToWAV converts an input audio file to WAV format with specified channels.
func ConvertToWAV(inputFilePath string) (wavFilePath string, err error) {
	fileExt := filepath.Ext(inputFilePath)
	if fileExt != ".wav" {
		defer os.Remove(inputFilePath)
	}

	outputFile := strings.TrimSuffix(inputFilePath, fileExt) + ".wav"

	// Output file may already exists. If it does FFmpeg will fail as
	// it cannot edit existing files in-place. Use a temporary file.
	tmpFile := filepath.Join(filepath.Dir(outputFile), "tmp_"+filepath.Base(outputFile))
	defer os.Remove(tmpFile)

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", inputFilePath,
		"-c:a", "pcm_s16le",
		"-ar", "44100",
		"-ac", "1",
		tmpFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert to WAV: %v, output %v", err, string(output))
	}

	// Rename the temporary file to the output file
	err = MoveFile(tmpFile, outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to rename temporary file to output file: %v", err)
	}

	return outputFile, nil
}

func MoveFile(src, dst string) error {
	err := os.Rename(src, dst)
	return err
}

func GenerateSongKey(songTitle, songArtist string) string {
	return songTitle + "---" + songArtist
}
