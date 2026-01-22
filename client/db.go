package main

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DBClient struct {
	db *sql.DB
}

func NewDBClient(dbPath string) (*DBClient, error) {
	// Resolve path relative to client directory
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error resolving database path: %s", err)
	}

	db, err := sql.Open("sqlite3", absPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %s", err)
	}

	return &DBClient{db: db}, nil
}

func (c *DBClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// MatchResult contains the matched song and the match score information
type MatchResult struct {
	Song          *Song
	MatchCount    int     // Number of fingerprints that matched
	MatchPercent  float64 // Percentage of query fingerprints that matched
	TotalQueried  int     // Total number of fingerprints in the query
}

// QuerySong matches fingerprints against the database and returns the best matching song with score
func (c *DBClient) QuerySong(queryFingerprints map[uint32]bool) (*MatchResult, error) {
	if len(queryFingerprints) == 0 {
		return nil, fmt.Errorf("no fingerprints provided")
	}

	// Build query to find matching fingerprints
	// Count matches per songID
	addresses := make([]interface{}, 0, len(queryFingerprints))
	for address := range queryFingerprints {
		addresses = append(addresses, address)
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no addresses to query")
	}

	totalQueried := len(queryFingerprints)

	placeholders := buildPlaceholders(len(addresses))
	query := fmt.Sprintf(`
		SELECT f.songID, COUNT(*) as match_count
		FROM fingerprints f
		INNER JOIN songs s ON f.songID = s.id
		WHERE f.address IN (%s)
		GROUP BY f.songID
		ORDER BY match_count DESC
		LIMIT 1
	`, placeholders)

	var songID int64
	var matchCount int
	err := c.db.QueryRow(query, addresses...).Scan(&songID, &matchCount)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no matching song found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying database: %s", err)
	}

	// Get song details
	var song Song
	err = c.db.QueryRow("SELECT id, title, artist, ytID, key, songpath FROM songs WHERE id = ?", songID).
		Scan(&song.ID, &song.Title, &song.Artist, &song.YtID, &song.Key, &song.SongPath)
	if err != nil {
		return nil, fmt.Errorf("error getting song details: %s", err)
	}

	// Calculate match percentage
	matchPercent := (float64(matchCount) / float64(totalQueried)) * 100.0

	return &MatchResult{
		Song:         &song,
		MatchCount:   matchCount,
		MatchPercent: matchPercent,
		TotalQueried: totalQueried,
	}, nil
}

func buildPlaceholders(count int) string {
	if count == 0 {
		return ""
	}
	placeholders := make([]byte, count*2-1)
	for i := 0; i < count; i++ {
		if i > 0 {
			placeholders[i*2-1] = ','
		}
		placeholders[i*2] = '?'
	}
	return string(placeholders)
}
