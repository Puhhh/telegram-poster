package state

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type ItemMeta struct {
	Title string
	Link  string
}

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &SQLiteStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS feed_state (
	feed_name TEXT PRIMARY KEY,
	initialized_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS seen_items (
	feed_name TEXT NOT NULL,
	item_key TEXT NOT NULL,
	title TEXT NOT NULL,
	link TEXT NOT NULL,
	first_seen_at TIMESTAMP NOT NULL,
	posted_at TIMESTAMP,
	PRIMARY KEY (feed_name, item_key)
);
`)
	return err
}

func (s *SQLiteStore) IsSeen(feedName, itemKey string) (bool, error) {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM seen_items WHERE feed_name = ? AND item_key = ?`, feedName, itemKey).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLiteStore) MarkSeen(feedName, itemKey string, meta ItemMeta) error {
	_, err := s.db.Exec(`
INSERT INTO seen_items (feed_name, item_key, title, link, first_seen_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(feed_name, item_key) DO NOTHING
`, feedName, itemKey, meta.Title, meta.Link, time.Now().UTC())
	return err
}

func (s *SQLiteStore) MarkPosted(feedName, itemKey string) error {
	_, err := s.db.Exec(`
UPDATE seen_items
SET posted_at = ?
WHERE feed_name = ? AND item_key = ?
`, time.Now().UTC(), feedName, itemKey)
	return err
}

func (s *SQLiteStore) IsFeedInitialized(feedName string) (bool, error) {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM feed_state WHERE feed_name = ?`, feedName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLiteStore) MarkFeedInitialized(feedName string) error {
	_, err := s.db.Exec(`
INSERT INTO feed_state (feed_name, initialized_at)
VALUES (?, ?)
ON CONFLICT(feed_name) DO NOTHING
`, feedName, time.Now().UTC())
	return err
}
