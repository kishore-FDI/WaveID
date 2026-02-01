package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	SONGS_DIR   = "songs"
	MAX_WORKERS = 5
	DB_PATH     = "shazam.db"
)

func main() {
	fmt.Println("Starting the Project Server...")

	if len(os.Args) < 2 {
		fmt.Println("Error: No configuration file provided.")
		os.Exit(1)
	}
	err := os.MkdirAll(SONGS_DIR, 0755)
	if err != nil {
		fmt.Printf("Error creating songs directory: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "find":
		if len(os.Args) < 3 {
			fmt.Println("Usage: main.go find <path_to_wav_file>")
			os.Exit(1)
		}
		filePath := os.Args[2]
		find(filePath)
	case "download":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go download example.json")
			os.Exit(1)
		}
		url := os.Args[2]
		download(url)
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		protocol := serveCmd.String("proto", "http", "Protocol to use (http or https)")
		port := serveCmd.String("p", "5000", "Port to use")
		serveCmd.Parse(os.Args[2:])
		serve(*protocol, *port)
	default:
		fmt.Println("Unknown command. Available commands: find")
	}

}
