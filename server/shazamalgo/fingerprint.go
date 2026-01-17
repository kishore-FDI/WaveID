package shazamalgo

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"sort"

	"shazam/db"
	"shazam/utils"
)

func wavToPCM(path string) ([]int16, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := make([]byte, 44)
	if _, err := f.Read(h); err != nil {
		return nil, err
	}
	if string(h[0:4]) != "RIFF" || string(h[8:12]) != "WAVE" {
		return nil, errors.New("invalid wav")
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	n := len(data) / 2
	pcm := make([]int16, n)
	for i := 0; i < n; i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return pcm, nil
}

type Peak struct {
	Time int
	Freq int
	Amp  float64
}

func pcmToFloat(x []int16) []float64 {
	y := make([]float64, len(x))
	for i := range x {
		y[i] = float64(x[i])
	}
	return y
}

func lowPassFIR(x []float64, fs, fc float64) []float64 {
	n := 101
	h := make([]float64, n)
	m := (n - 1) / 2

	for i := 0; i < n; i++ {
		if i == m {
			h[i] = 2 * fc / fs
		} else {
			t := float64(i - m)
			h[i] = math.Sin(2*math.Pi*fc*t/fs) / (math.Pi * t)
		}
		h[i] *= 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}

	y := make([]float64, len(x))
	for i := range x {
		for j := 0; j < n; j++ {
			if i-j >= 0 {
				y[i] += x[i-j] * h[j]
			}
		}
	}
	return y
}

func preprocess(path string) ([]float64, error) {
	pcm16, err := wavToPCM(path)
	if err != nil {
		return nil, err
	}

	maxSamples := 11025 * 30
	if len(pcm16) > maxSamples {
		pcm16 = pcm16[:maxSamples]
	}

	x := pcmToFloat(pcm16)
	x = lowPassFIR(x, 11025, 5000)
	return x, nil
}

func PickPeaks(spec [][]float64) []Peak {
	const maxPerFrame = 5
	const maxTotal = 5000

	var peaks []Peak

	for t := range spec {
		type local struct {
			f int
			a float64
		}
		var locals []local

		for f := 1; f < len(spec[t])-1; f++ {
			if spec[t][f] > spec[t][f-1] && spec[t][f] > spec[t][f+1] {
				locals = append(locals, local{f, spec[t][f]})
			}
		}

		sort.Slice(locals, func(i, j int) bool {
			return locals[i].a > locals[j].a
		})

		for i := 0; i < len(locals) && i < maxPerFrame; i++ {
			peaks = append(peaks, Peak{Time: t, Freq: locals[i].f, Amp: locals[i].a})
			if len(peaks) >= maxTotal {
				return peaks
			}
		}
	}
	return peaks
}

func MakeHashes(peaks []Peak) []utils.Hash {
	const maxPairs = 20
	const minDt = 5
	const maxDt = 100

	var hashes []utils.Hash

	for i, p := range peaks {
		pairs := 0
		for j := i + 1; j < len(peaks) && pairs < maxPairs; j++ {
			dt := peaks[j].Time - p.Time
			if dt < minDt || dt > maxDt {
				continue
			}
			h := uint32(p.Freq)<<20 | uint32(peaks[j].Freq)<<10 | uint32(dt)
			hashes = append(hashes, utils.Hash{Value: h, Time: p.Time})
			pairs++
		}
	}
	return hashes
}

func FingerPrint(path string, songID int64, d *db.DB) error {
	x, err := preprocess(path)
	if err != nil {
		return err
	}
	spec := STFT(x, 512, 256)

	peaks := PickPeaks(spec)
	hashes := MakeHashes(peaks)
	d.InsertHashes(hashes, songID)

	return nil
}
