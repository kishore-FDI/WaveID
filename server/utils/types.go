package utils

type Song struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	YtID     string `json:"yt_id"`
	Key      string `json:"key"`
	SongPath string `json:"song_path"`
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
}

type WavInfo struct {
	Channels       int
	SampleRate     int
	Duration       float64
	Data           []byte
	ChannelSamples []float64
}
