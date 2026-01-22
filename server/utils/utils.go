package utils

import (
	"io"
	"math/rand"
	"os"
	"time"
)

func MkDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func MoveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			out.Close()
			os.Remove(dst)
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	err = out.Sync()
	if err != nil {
		return err
	}

	err = out.Close()
	if err != nil {
		return err
	}

	return os.Remove(src)
}

func GenerateSongKey(songTitle, songArtist string) string {
	return songTitle + "---" + songArtist
}

func GenerateUniqueID() uint32 {
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Uint32()

	return randomNumber
}
