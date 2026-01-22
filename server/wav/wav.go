package wav

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	types "shazam/servertypes"
	"shazam/utils"
	"strings"
)

func ConvertToWAV(inputFilePath string) (wavFilePath string, err error) {
	_, err = os.Stat(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("input file does not exist: %v", err)
	}

	fileExt := filepath.Ext(inputFilePath)
	if fileExt != ".wav" {
		defer os.Remove(inputFilePath)
	}

	outputFile := strings.TrimSuffix(inputFilePath, fileExt) + ".wav"

	tmpFile := filepath.Join(filepath.Dir(outputFile), "tmp_"+filepath.Base(outputFile))
	defer os.Remove(tmpFile)

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", inputFilePath,
		"-c", "pcm_s16le",
		"-ar", "44100",
		"-ac", "1",
		tmpFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert to WAV: %v, output %v", err, string(output))
	}

	err = utils.MoveFile(tmpFile, outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to rename temporary file to output file: %v", err)
	}

	return outputFile, nil
}

func ReadWavInfo(filename string) (*types.WavInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if len(data) < 44 {
		return nil, errors.New("invalid WAV file size (too small)")
	}

	var header types.WavHeader
	if err := binary.Read(bytes.NewReader(data[:44]), binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if string(header.ChunkID[:]) != "RIFF" ||
		string(header.Format[:]) != "WAVE" ||
		header.AudioFormat != 1 {
		return nil, errors.New("invalid WAV header format")
	}
	if header.NumChannels != 1 {
		return nil, errors.New("unsupported channel count (expect mono)")
	}
	if header.BitsPerSample != 16 {
		return nil, errors.New("unsupported bits-per-sample (expect 16-bit PCM)")
	}

	info := &types.WavInfo{
		Channels:   1,
		SampleRate: int(header.SampleRate),
		Data:       data[44:],
	}

	sampleCount := len(info.Data) / 2
	int16Buf := make([]int16, sampleCount)
	if err := binary.Read(bytes.NewReader(info.Data), binary.LittleEndian, int16Buf); err != nil {
		return nil, err
	}

	const scale = 1.0 / 32768.0
	samples := make([]float64, sampleCount)
	for i, s := range int16Buf {
		samples[i] = float64(s) * scale
	}
	info.ChannelSamples = samples

	info.Duration = float64(sampleCount) / float64(header.SampleRate)
	return info, nil
}
