package state

import (
	"path/filepath"
	"testing"
)

func TestSQLiteStoreMarksSeenAndPosted(t *testing.T) {
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seen, err := store.IsSeen("feed-a", "guid-1")
	if err != nil {
		t.Fatal(err)
	}
	if seen {
		t.Fatal("new item should not be seen")
	}

	if err := store.MarkSeen("feed-a", "guid-1", ItemMeta{Title: "Title", Link: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	seen, err = store.IsSeen("feed-a", "guid-1")
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("item should be seen")
	}

	if err := store.MarkPosted("feed-a", "guid-1"); err != nil {
		t.Fatal(err)
	}
}
