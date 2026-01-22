package db

import (
	"database/sql"
	"fmt"
	types "shazam/servertypes"
	"shazam/utils"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var sqlitefilterKeys = "id | ytID | key"

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
        key TEXT NOT NULL UNIQUE,
        songpath TEXT NOT NULL
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

func (db *SQLiteClient) AddSong(song types.Song) (int64, error) {
	key := song.Key
	if key == "" {
		key = utils.GenerateSongKey(song.Title, song.Artist)
	}
	result, err := db.db.Exec("INSERT OR IGNORE INTO songs (title, artist, ytID, key, songpath) VALUES (?, ?, ?, ?, ?)", song.Title, song.Artist, song.YtID, key, song.SongPath)
	if err != nil {
		return 0, fmt.Errorf("error adding song: %s", err)
	}
	songID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting song ID: %s", err)
	}
	return songID, nil
}

func (db *SQLiteClient) AddFingerPrints(fingerprints map[uint32]types.Couple) error {
	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %s", err)
	}

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO fingerprints (address, anchorTimeMs, songID) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error preparing statement: %s", err)
	}
	defer stmt.Close()

	for address, couple := range fingerprints {
		if _, err := stmt.Exec(address, couple.AnchorTimeMs, couple.SongID); err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing statement: %s", err)
		}
	}

	return tx.Commit()
}

func (db *SQLiteClient) GetCouples(addresses []uint32) (map[uint32][]types.Couple, error) {
	couples := make(map[uint32][]types.Couple)

	for _, address := range addresses {
		// Use INNER JOIN to only get fingerprints for songs that exist
		rows, err := db.db.Query(`
			SELECT f.anchorTimeMs, f.songID 
			FROM fingerprints f
			INNER JOIN songs s ON f.songID = s.id
			WHERE f.address = ?
		`, address)
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
	return db.GetSong("id", songID)
}

// GetSong retrieves a song by filter key
func (s *SQLiteClient) GetSong(filterKey string, value interface{}) (types.Song, bool, error) {

	if !strings.Contains(sqlitefilterKeys, filterKey) {
		return types.Song{}, false, fmt.Errorf("invalid filter key")
	}

	query := fmt.Sprintf("SELECT title, artist, ytID FROM songs WHERE %s = ?", filterKey)

	row := s.db.QueryRow(query, value)

	var song types.Song
	err := row.Scan(&song.Title, &song.Artist, &song.YtID)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Song{}, false, nil
		}
		return types.Song{}, false, fmt.Errorf("failed to retrieve song: %s", err)
	}

	return song, true, nil
}

// CleanupOrphanedFingerprints removes fingerprints that reference non-existent songs
func (db *SQLiteClient) CleanupOrphanedFingerprints() (int64, error) {
	query := `
		DELETE FROM fingerprints 
		WHERE songID NOT IN (SELECT id FROM songs)
	`
	result, err := db.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("error cleaning up orphaned fingerprints: %s", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error getting rows affected: %s", err)
	}
	
	return rowsAffected, nil
}
