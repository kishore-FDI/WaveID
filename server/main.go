package main

import (
	"fmt"
	"os"
	"shazam/helper"
)

func main() {
	fmt.Println(
		"Hello and thank you for trying out this project.\n" +
			"This is a minimal replica of how Shazam works.\n\n" +
			"Usage:\n" +
			"  1 <spotify_url>  Download playlist and store it in the DB\n" +
			"  2               List all songs in the DB\n" +
			"  3 <song_id>     Delete a song by ID\n" +
			"  4               Start HTTP server",
	)

	if len(os.Args) < 2 {
		return
	}

	switch os.Args[1] {
	case "1":
		if len(os.Args) < 3 {
			fmt.Println("missing spotify url")
			return
		}
		fmt.Println("downloading spotify metadata for:", os.Args[2])
		helper.DownloadSpotifyMetadata(os.Args[2])

	case "2":
		fmt.Println("listing all songs in database")

	case "3":
		if len(os.Args) < 3 {
			fmt.Println("missing song id")
			return
		}
		fmt.Println("deleting song with id:", os.Args[2])

	case "4":
		fmt.Println("starting http server")

	default:
		fmt.Println("unknown command")
	}
}
