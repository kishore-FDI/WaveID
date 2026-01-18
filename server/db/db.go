package db

import (
	"database/sql"
	"fmt"
	"shazam/utils"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteClient struct {
	db *sql.DB
}

func NewSQLiteClient(dbPath string) (*SQLiteClient, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %s", err)
	}
	err = createTables(db)
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %s", err)
	}
	return &SQLiteClient{db: db}, nil
}

func (db *SQLiteClient) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
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

func (db *SQLiteClient) AddSong(song utils.Song) error {
	key := song.Key
	if key == "" {
		key = utils.GenerateSongKey(song.Title, song.Artist)
	}
	_, err := db.db.Exec("INSERT OR IGNORE INTO songs (title, artist, ytID, key) VALUES (?, ?, ?, ?)", song.Title, song.Artist, song.YtID, key)
	if err != nil {
		return fmt.Errorf("error adding song: %s", err)
	}
	return nil
}
