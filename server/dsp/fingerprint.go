package dsp

import (
	"fmt"
	"shazam/utils"
	"shazam/wav"
)

func FingerPrint(songPath string, songID int64) (map[uint32]utils.Couple, error) {
	wavFilePath, err := wav.ConvertToWAV(songPath)
	if err != nil {
		panic(err)
	}
	wavInfo, err := wav.ReadWavInfo(wavFilePath)
	fmt.Print(wavInfo.SampleRate)
	fingerprints := make(map[uint32]utils.Couple)
	spectro, err := Spectrogram(wavInfo.ChannelSamples, wavInfo.SampleRate)
	fmt.Print(spectro)
	return fingerprints, nil
}
