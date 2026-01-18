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

	dbPath := filepath.Join("PROCESSED_DIR", "shazam.db")
	client, err := db.NewSQLiteClient(dbPath)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if len(os.Args) < 3 {
		fmt.Println("usage: <program> <option> <arg>")
		return
	}

	fmt.Print("Starting the project")

	switch os.Args[1] {
	case "1":
		DownloadSongs(os.Args[2], client)
	default:
		fmt.Println("Choose a valid option")
	}
}
