package state

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"

	"telegram-poster/internal/dedupe"
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
CREATE TABLE IF NOT EXISTS global_seen_items (
	item_key TEXT PRIMARY KEY,
	first_feed_name TEXT NOT NULL,
	title TEXT NOT NULL,
	link TEXT NOT NULL,
	first_seen_at TIMESTAMP NOT NULL,
	posted_at TIMESTAMP
);
`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO global_seen_items (item_key, first_feed_name, title, link, first_seen_at, posted_at)
SELECT item_key, feed_name, title, link, first_seen_at, posted_at
FROM seen_items;
INSERT OR IGNORE INTO global_seen_items (item_key, first_feed_name, title, link, first_seen_at, posted_at)
SELECT link, feed_name, title, link, first_seen_at, posted_at
FROM seen_items
WHERE link != '';
`)
	if err != nil {
		return err
	}
	return s.backfillCanonicalLinks(ctx)
}

func (s *SQLiteStore) backfillCanonicalLinks(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
SELECT feed_name, title, link, first_seen_at, posted_at
FROM seen_items
WHERE link != ''
`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type seenItem struct {
		feedName    string
		title       string
		link        string
		firstSeenAt time.Time
		postedAt    sql.NullTime
	}
	var items []seenItem
	for rows.Next() {
		var item seenItem
		if err := rows.Scan(&item.feedName, &item.title, &item.link, &item.firstSeenAt, &item.postedAt); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, item := range items {
		canonicalLink := dedupe.CanonicalLink(item.link)
		if canonicalLink == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO global_seen_items (item_key, first_feed_name, title, link, first_seen_at, posted_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(item_key) DO NOTHING
`, canonicalLink, item.feedName, item.title, item.link, item.firstSeenAt, item.postedAt); err != nil {
			return err
		}
	}
	return err
}

func (s *SQLiteStore) IsSeen(itemKey string) (bool, error) {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM global_seen_items WHERE item_key = ?`, itemKey).Scan(&exists)
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
INSERT INTO global_seen_items (item_key, first_feed_name, title, link, first_seen_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(item_key) DO NOTHING
`, itemKey, feedName, meta.Title, meta.Link, time.Now().UTC())
	return err
}

func (s *SQLiteStore) MarkPosted(itemKey string) error {
	_, err := s.db.Exec(`
UPDATE global_seen_items
SET posted_at = ?
WHERE item_key = ?
`, time.Now().UTC(), itemKey)
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
