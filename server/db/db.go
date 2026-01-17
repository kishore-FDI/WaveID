// db/db.go
package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"shazam/utils"

	_ "github.com/mattn/go-sqlite3"
)

const baseDir = "processedAudios"
const dbFile = "shazam.db"

type DB struct {
	*sql.DB
}

func init() {
	must(os.MkdirAll(baseDir, 0755))
}

func Open() *DB {
	p := filepath.Join(baseDir, dbFile)
	db, err := sql.Open("sqlite3", p)
	must(err)

	_, err = db.Exec(`
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS songs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		artist TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		duration REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS hashes (
		hash INTEGER NOT NULL,
		song_id INTEGER NOT NULL,
		time_offset INTEGER NOT NULL,
		FOREIGN KEY(song_id) REFERENCES songs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_hash ON hashes(hash);
	CREATE INDEX IF NOT EXISTS idx_song ON hashes(song_id);
	`)
	must(err)

	return &DB{DB: db}
}

func (d *DB) InsertSong(title, artist, path string, duration float64) int64 {
	res, err := d.Exec(
		`INSERT INTO songs(title, artist, path, duration) VALUES(?,?,?,?)`,
		title, artist, path, duration,
	)
	must(err)
	id, _ := res.LastInsertId()
	return id
}

func (d *DB) SongExists(title, artist string) bool {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM songs WHERE title = ? AND artist = ?", title, artist).Scan(&count)
	must(err)
	return count > 0
}

func (d *DB) InsertHashes(hashes []utils.Hash, songID int64) {
	tx, err := d.Begin()
	must(err)

	stmt, err := tx.Prepare(
		`INSERT INTO hashes(hash, song_id, time_offset) VALUES(?,?,?)`,
	)
	must(err)

	for _, h := range hashes {
		_, err := stmt.Exec(h.Value, songID, h.Time)
		must(err)
	}

	must(stmt.Close())
	must(tx.Commit())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
