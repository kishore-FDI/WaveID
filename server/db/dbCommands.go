package db

import (
	"fmt"
	"shazam/types"
	"shazam/utils"

	"github.com/mattn/go-sqlite3"
)

func (db *SQLiteClient) RegisterSong(songTitle, songArtist, ytID string) (uint32, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %s", err)
	}

	stmt, err := tx.Prepare("INSERT INTO songs (id, title, artist, ytID, key) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("error preparing statement: %s", err)
	}
	defer stmt.Close()

	songID := utils.GenerateUniqueID()
	songKey := utils.GenerateSongKey(songTitle, songArtist)
	if _, err := stmt.Exec(songID, songTitle, songArtist, ytID, songKey); err != nil {
		tx.Rollback()
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
			return 0, fmt.Errorf("song with ytID or key already exists: %v", err)
		}
		return 0, fmt.Errorf("failed to register song: %v", err)
	}

	return songID, tx.Commit()
}

func (db *SQLiteClient) StoreFingerprints(fingerprints map[uint32]types.Couple) error {
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
