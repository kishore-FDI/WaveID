package utils

import "os"

func MkDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func GenerateSongKey(songTitle, songArtist string) string {
	return songTitle + "---" + songArtist
}
