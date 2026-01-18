package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: <program> <option> <arg>")
		return
	}

	fmt.Print("Starting the project")

	switch os.Args[1] {
	case "1":
		DownloadSongs(os.Args[2])
	default:
		fmt.Println("Choose a valid option")
	}
}

func DownloadSongs(url string) {
	fmt.Println("Downloading songs from", url)
}
