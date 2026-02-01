package db

import (
	"database/sql"
	"fmt"
	"shazam/types"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteClient struct {
	db *sql.DB
}

func (db *SQLiteClient) TotalSongs() (int, error) {
	var count int
	err := db.db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting songs: %s", err)
	}
	return count, nil
}

func DBClient(path string) (*SQLiteClient, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("error connecting to SQLite: %s", err)
	}

	err = createTables(db)
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %s", err)
	}

	return &SQLiteClient{db: db}, nil
}

func (client *SQLiteClient) Close() error {
	return client.db.Close()
}

// createTables creates the required tables if they don't exist
func createTables(db *sql.DB) error {
	createSongsTable := `
    CREATE TABLE IF NOT EXISTS songs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        artist TEXT NOT NULL,
        ytID TEXT,
        key TEXT NOT NULL UNIQUE
    );
    `

	createFingerprintsTable := `
    CREATE TABLE IF NOT EXISTS fingerprints (
        address INTEGER NOT NULL,
        anchorTimeMs INTEGER NOT NULL,
        songID INTEGER NOT NULL,
        PRIMARY KEY (address, anchorTimeMs, songID)
    );
    `

	_, err := db.Exec(createSongsTable)
	if err != nil {
		return fmt.Errorf("error creating songs table: %s", err)
	}

	_, err = db.Exec(createFingerprintsTable)
	if err != nil {
		return fmt.Errorf("error creating fingerprints table: %s", err)
	}

	return nil
}

func (db *SQLiteClient) GetCouples(addresses []uint32) (map[uint32][]types.Couple, error) {
	couples := make(map[uint32][]types.Couple)

	for _, address := range addresses {
		rows, err := db.db.Query("SELECT anchorTimeMs, songID FROM fingerprints WHERE address = ?", address)
		if err != nil {
			return nil, fmt.Errorf("error querying database: %s", err)
		}

		var docCouples []types.Couple
		for rows.Next() {
			var couple types.Couple
			if err := rows.Scan(&couple.AnchorTimeMs, &couple.SongID); err != nil {
				rows.Close() // close before returning error
				return nil, fmt.Errorf("error scanning row: %s", err)
			}
			docCouples = append(docCouples, couple)
		}

		rows.Close() // close explicitly after reading

		couples[address] = docCouples
	}

	return couples, nil
}

func (db *SQLiteClient) GetSongByID(songID uint32) (types.Song, bool, error) {
	var song types.Song
	err := db.db.QueryRow("SELECT id, title, artist, ytID FROM songs WHERE id = ?", songID).Scan(&song.ID, &song.Title, &song.Artist, &song.YouTubeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return song, false, nil
		}
		return song, false, fmt.Errorf("error querying song by ID: %s", err)
	}
	return song, true, nil
}
