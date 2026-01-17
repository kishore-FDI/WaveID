package main

import (
	"os"
	"shazam/db"
	"shazam/helper"
	"shazam/utils"
)

func main() {
	utils.InitLogger(utils.INFO)

	// utils.Log.Info(
	// 	"Hello and thank you for trying out this project.\n" +
	// 		"This is a minimal replica of how Shazam works.\n\n" +
	// 		"Usage:\n" +
	// 		"  1 <spotify_url>  Download playlist and store it in the DB\n" +
	// 		"  2               List all songs in the DB\n" +
	// 		"  3 <song_id>     Delete a song by ID\n" +
	// 		"  4               Start HTTP server",
	// )
	if len(os.Args) < 2 {
		utils.Log.Fatal("missing command")
	}

	database := db.Open()
	defer database.Close()

	switch os.Args[1] {
	case "1":
		if len(os.Args) < 3 {
			// utils.Log.Fatal("missing spotify url or json file")
		}
		// utils.Log.Info("downloading playlist: %s", os.Args[2])
		helper.DownloadSpotifyMetadata(os.Args[2])

	case "2":
		utils.Log.Info("listing all songs in database")

	case "3":
		if len(os.Args) < 3 {
			utils.Log.Fatal("missing song id")
		}
		utils.Log.Info("deleting song id: %s", os.Args[2])

	case "4":
		utils.Log.Info("starting http server")

	default:
		utils.Log.Fatal("unknown command")
	}
}
