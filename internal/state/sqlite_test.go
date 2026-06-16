package state

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteStoreMarksSeenAndPosted(t *testing.T) {
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seen, err := store.IsSeen("guid-1")
	if err != nil {
		t.Fatal(err)
	}
	if seen {
		t.Fatal("new item should not be seen")
	}

	if err := store.MarkSeen("feed-a", "guid-1", ItemMeta{Title: "Title", Link: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	seen, err = store.IsSeen("guid-1")
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("item should be seen")
	}

	if err := store.MarkPosted("guid-1"); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteStoreSeenItemsAreGlobalAcrossFeeds(t *testing.T) {
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.MarkSeen("feed-a", "https://example.com/post", ItemMeta{Title: "Title", Link: "https://example.com/post"}); err != nil {
		t.Fatal(err)
	}
	seen, err := store.IsSeen("https://example.com/post")
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("item should be seen globally")
	}
}

func TestSQLiteStoreBackfillsGlobalSeenItemsFromOldSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE seen_items (
	feed_name TEXT NOT NULL,
	item_key TEXT NOT NULL,
	title TEXT NOT NULL,
	link TEXT NOT NULL,
	first_seen_at TIMESTAMP NOT NULL,
	posted_at TIMESTAMP,
	PRIMARY KEY (feed_name, item_key)
);
INSERT INTO seen_items (feed_name, item_key, title, link, first_seen_at)
VALUES ('feed-a', 'https://example.com/post', 'Title', 'https://example.com/post', CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seen, err := store.IsSeen("https://example.com/post")
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("old seen_items row should be backfilled into global seen state")
	}
}

func TestSQLiteStoreBackfillsCanonicalLinksFromOldSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE seen_items (
	feed_name TEXT NOT NULL,
	item_key TEXT NOT NULL,
	title TEXT NOT NULL,
	link TEXT NOT NULL,
	first_seen_at TIMESTAMP NOT NULL,
	posted_at TIMESTAMP,
	PRIMARY KEY (feed_name, item_key)
);
INSERT INTO seen_items (feed_name, item_key, title, link, first_seen_at)
VALUES (
	'feed-a',
	'guid-1',
	'Title',
	'https://habr.com/ru/articles/1042858/?utm_source=rss#comments',
	CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seen, err := store.IsSeen("https://habr.com/ru/articles/1042858/")
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("old tracking link should be backfilled using canonical item key")
	}
}
