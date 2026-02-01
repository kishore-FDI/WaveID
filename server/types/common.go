package types

type YTMeta struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Uploader string `json:"uploader"`
	Filename string `json:"filename"`
}

type Couple struct {
	AnchorTimeMs uint32
	SongID       uint32
}

type RecordData struct {
	Audio      string  `json:"audio"`
	Duration   float64 `json:"duration"`
	Channels   int     `json:"channels"`
	SampleRate int     `json:"sampleRate"`
	SampleSize int     `json:"sampleSize"`
}

type WavInfo struct {
	Channels           int
	SampleRate         int
	LeftChannelSamples []float64
	Data               []byte
	Duration           float64
}

type WavHeader struct {
	ChunkID       [4]byte
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte
	Subchunk2Size uint32
	BytesPerSec   uint32
}

type Match struct {
	SongID     uint32
	SongTitle  string
	SongArtist string
	YouTubeID  string
	Timestamp  uint32
	Score      float64
}

type Song struct {
	ID        uint32
	Title     string
	Artist    string
	YouTubeID string
}
