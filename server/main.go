package main

import (
	"fmt"
	"os"
	"path/filepath"
	"shazam/db"
	"shazam/utils"
)

func main() {
	if err := utils.MkDir("SONGS_DIR"); err != nil {
		panic(err)
	}
	if err := utils.MkDir("PROCESSED_DIR"); err != nil {
		panic(err)
	}
	if err := utils.MkDir("RECORDINGS_DIR"); err != nil {
		panic(err)
	}

	dbPath := filepath.Join("PROCESSED_DIR", "shazam.db")
	client, err := db.NewSQLiteClient(dbPath)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if len(os.Args) < 2 {
		fmt.Println("usage: <program> <option> <arg>")
		fmt.Println("  or:  <program> server  (to start HTTP server)")
		return
	}

	fmt.Print("Starting the project")

	switch os.Args[1] {
	case "1":
		DownloadSongs(os.Args[2], client)
	case "cleanup":
		fmt.Println("Cleaning up orphaned fingerprints...")
		removed, err := client.CleanupOrphanedFingerprints()
		if err != nil {
			fmt.Printf("Error cleaning up orphaned fingerprints: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully removed %d orphaned fingerprints\n", removed)
	case "server":
		setupHTTPServer("RECORDINGS_DIR")
	default:
		fmt.Println("Choose a valid option")
		fmt.Println("  Options:")
		fmt.Println("    1 <json_path>  - Download and process songs from JSON file")
		fmt.Println("    cleanup         - Remove orphaned fingerprints from database")
		fmt.Println("    server         - Start HTTP server")
	}
}
