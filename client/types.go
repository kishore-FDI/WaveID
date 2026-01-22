package main

type Peak struct {
	Freq float64 // Frequency in Hz
	Time float64 // Time in seconds
}

type Song struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	YtID     string `json:"yt_id"`
	Key      string `json:"key"`
	SongPath string `json:"song_path"`
}
