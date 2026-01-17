package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/gordonklaus/portaudio"
	_ "github.com/mattn/go-sqlite3"

	"shazam/shazamalgo"
	"shazam/utils"
)

const (
	targetRate = 11025
	seconds    = 10
	win        = 512
	hop        = 256
	fpbuf      = 256
)

func main() {
	fmt.Println("recording")

	pcm, srcRate := recordAudio()
	fmt.Println("raw samples:", len(pcm), "rate:", srcRate)

	x := pcmToFloat(pcm)
	x = resample(x, srcRate, targetRate)
	x = lowPassFIR(x, targetRate, 5000)
	normalize(x)
	saveWAV("recorded_input.wav", x, targetRate)
	fmt.Println("saved recorded_input.wav")

	spec := shazamalgo.STFT(x, win, hop)
	peaks := shazamalgo.PickPeaks(spec)
	hashes := shazamalgo.MakeHashes(peaks)

	fmt.Println("hashes:", len(hashes))
	fmt.Print(hashes)
	db := openDB()
	defer db.Close()

	songID := match(db, hashes)
	if songID == 0 {
		fmt.Println("no match")
		fmt.Printf("hashes: %d\n", len(hashes))
		return
	}

	title, artist := lookupSong(db, songID)
	fmt.Printf("match: %s — %s\n", artist, title)
}

func loadWAV(path string) ([]int16, float64) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := make([]byte, 44)
	if _, err := f.Read(h); err != nil {
		log.Fatal(err)
	}
	if string(h[0:4]) != "RIFF" || string(h[8:12]) != "WAVE" {
		log.Fatal("invalid wav")
	}

	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	n := len(data) / 2
	pcm := make([]int16, n)
	for i := 0; i < n; i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return pcm, 11025 // assuming 11025
}

func recordAudio() ([]int16, float64) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	dev, err := portaudio.DefaultInputDevice()
	if err != nil {
		log.Fatal(err)
	}

	rate := dev.DefaultSampleRate
	n := int(rate) * seconds

	buf := make([]int16, n)
	tmp := make([]int16, fpbuf)

	params := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   dev,
			Channels: 1,
			Latency:  dev.DefaultLowInputLatency,
		},
		SampleRate:      rate,
		FramesPerBuffer: fpbuf,
	}

	stream, err := portaudio.OpenStream(params, tmp)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	stream.Start()
	i := 0
	for i < len(buf) {
		stream.Read()
		copy(buf[i:], tmp)
		i += len(tmp)
	}
	stream.Stop()

	return buf, rate
}

func pcmToFloat(x []int16) []float64 {
	y := make([]float64, len(x))
	for i := range x {
		y[i] = float64(x[i]) / 32768.0
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

func normalize(x []float64) {
	var max float64
	for _, v := range x {
		av := math.Abs(v)
		if av > max {
			max = av
		}
	}
	if max == 0 {
		return
	}
	g := 0.9 / max
	for i := range x {
		x[i] *= g
	}
}

func resample(x []float64, src, dst float64) []float64 {
	ratio := dst / src
	n := int(float64(len(x)) * ratio)
	y := make([]float64, n)

	for i := 0; i < n; i++ {
		pos := float64(i) / ratio
		j := int(pos)
		if j+1 < len(x) {
			a := pos - float64(j)
			y[i] = x[j]*(1-a) + x[j+1]*a
		}
	}
	return y
}

func saveWAV(path string, x []float64, rate int) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	pcm := make([]int16, len(x))
	for i := range x {
		v := int(x[i] * 32767.0)
		if v > math.MaxInt16 {
			v = math.MaxInt16
		} else if v < math.MinInt16 {
			v = math.MinInt16
		}
		pcm[i] = int16(v)

	}

	dataLen := len(pcm) * 2
	riffLen := 36 + dataLen

	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(riffLen))
	f.Write([]byte("WAVE"))

	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint32(rate))
	binary.Write(f, binary.LittleEndian, uint32(rate*2))
	binary.Write(f, binary.LittleEndian, uint16(2))
	binary.Write(f, binary.LittleEndian, uint16(16))

	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataLen))
	binary.Write(f, binary.LittleEndian, pcm)
}

func openDB() *sql.DB {
	path := filepath.Join("..", "server", "processedAudios", "shazam.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func match(db *sql.DB, hashes []utils.Hash) int64 {
	counts := make(map[int64]map[int]int)

	for _, h := range hashes {
		rows, err := db.Query(
			`SELECT song_id, time_offset FROM hashes WHERE hash = ?`,
			h.Value,
		)
		if err != nil {
			continue
		}
		for rows.Next() {
			var sid int64
			var t int
			rows.Scan(&sid, &t)
			dt := (t - h.Time) / 2
			if counts[sid] == nil {
				counts[sid] = make(map[int]int)
			}
			counts[sid][dt]++
		}
		rows.Close()
	}

	var bestSong int64
	bestScore := 0
	for sid, m := range counts {
		for _, c := range m {
			if c > bestScore {
				bestScore = c
				bestSong = sid
			}
		}
	}
	fmt.Printf("best score: %d\n", bestScore)
	return bestSong
}

func lookupSong(db *sql.DB, id int64) (string, string) {
	var title, artist string
	db.QueryRow(
		`SELECT title, artist FROM songs WHERE id = ?`,
		id,
	).Scan(&title, &artist)
	return title, artist
}
