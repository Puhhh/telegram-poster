package poster

import (
	"context"
	"path/filepath"
	"testing"

	"telegram-poster/internal/state"
)

type fakeFeedClient struct {
	items []FeedItem
}

func (f fakeFeedClient) Fetch(ctx context.Context, url string) ([]FeedItem, error) {
	return f.items, nil
}

type fakeFeedClientByURL map[string][]FeedItem

func (f fakeFeedClientByURL) Fetch(ctx context.Context, url string) ([]FeedItem, error) {
	return f[url], nil
}

type fakeTelegram struct {
	messages []OutgoingMessage
}

func (f *fakeTelegram) Send(ctx context.Context, msg OutgoingMessage) error {
	f.messages = append(f.messages, msg)
	return nil
}

func TestProcessFeedFirstRunMarksExistingWithoutPosting(t *testing.T) {
	store, err := state.OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tg := &fakeTelegram{}
	p := New(fakeFeedClient{items: []FeedItem{{GUID: "1", Title: "Old", Link: "https://example.com/old"}}}, store, tg)

	if err := p.ProcessFeed(context.Background(), Feed{Name: "news", URL: "https://example.com/rss", Channel: "@news"}); err != nil {
		t.Fatal(err)
	}
	if len(tg.messages) != 0 {
		t.Fatalf("first run posted %d messages", len(tg.messages))
	}
}

func TestProcessFeedPostsOnlyNewItemsAfterFirstRun(t *testing.T) {
	store, err := state.OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tg := &fakeTelegram{}
	p := New(fakeFeedClient{items: []FeedItem{{GUID: "1", Title: "Old", Link: "https://example.com/old"}}}, store, tg)
	feed := Feed{Name: "news", URL: "https://example.com/rss", Channel: "@news"}
	if err := p.ProcessFeed(context.Background(), feed); err != nil {
		t.Fatal(err)
	}

	p.feedClient = fakeFeedClient{items: []FeedItem{
		{GUID: "1", Title: "Old", Link: "https://example.com/old"},
		{GUID: "2", Title: "New", Summary: "<b>Fresh</b>", Link: "https://example.com/new"},
	}}
	if err := p.ProcessFeed(context.Background(), feed); err != nil {
		t.Fatal(err)
	}

	if len(tg.messages) != 1 {
		t.Fatalf("posted messages = %d", len(tg.messages))
	}
	if tg.messages[0].ChatID != "@news" {
		t.Fatalf("chat = %q", tg.messages[0].ChatID)
	}
	if got := tg.messages[0].Text; got != "New\n\nhttps://example.com/new" {
		t.Fatalf("message = %q", got)
	}
}

func TestProcessFeedDeduplicatesItemsAcrossFeeds(t *testing.T) {
	store, err := state.OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tg := &fakeTelegram{}
	p := New(fakeFeedClientByURL{
		"https://example.com/feed-a.xml": {
			{GUID: "old-a", Title: "Old A", Link: "https://example.com/old-a"},
		},
		"https://example.com/feed-b.xml": {
			{GUID: "old-b", Title: "Old B", Link: "https://example.com/old-b"},
		},
	}, store, tg)
	feedA := Feed{Name: "feed-a", URL: "https://example.com/feed-a.xml", Channel: "@news"}
	feedB := Feed{Name: "feed-b", URL: "https://example.com/feed-b.xml", Channel: "@news"}
	if err := p.ProcessFeed(context.Background(), feedA); err != nil {
		t.Fatal(err)
	}
	if err := p.ProcessFeed(context.Background(), feedB); err != nil {
		t.Fatal(err)
	}

	p.feedClient = fakeFeedClientByURL{
		"https://example.com/feed-a.xml": {
			{GUID: "a-guid", Title: "Shared", Link: "https://example.com/shared"},
		},
		"https://example.com/feed-b.xml": {
			{GUID: "b-guid", Title: "Shared", Link: "https://example.com/shared"},
		},
	}
	if err := p.ProcessFeed(context.Background(), feedA); err != nil {
		t.Fatal(err)
	}
	if err := p.ProcessFeed(context.Background(), feedB); err != nil {
		t.Fatal(err)
	}

	if len(tg.messages) != 1 {
		t.Fatalf("posted messages = %d", len(tg.messages))
	}
	if tg.messages[0].ChatID != "@news" {
		t.Fatalf("chat = %q", tg.messages[0].ChatID)
	}
}

func TestProcessFeedDeduplicatesTrackingURLVariants(t *testing.T) {
	store, err := state.OpenSQLite(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tg := &fakeTelegram{}
	p := New(fakeFeedClientByURL{
		"https://example.com/feed-a.xml": {
			{Title: "Old A", Link: "https://example.com/old-a"},
		},
		"https://example.com/feed-b.xml": {
			{Title: "Old B", Link: "https://example.com/old-b"},
		},
	}, store, tg)
	feedA := Feed{Name: "feed-a", URL: "https://example.com/feed-a.xml", Channel: "@news"}
	feedB := Feed{Name: "feed-b", URL: "https://example.com/feed-b.xml", Channel: "@news"}
	if err := p.ProcessFeed(context.Background(), feedA); err != nil {
		t.Fatal(err)
	}
	if err := p.ProcessFeed(context.Background(), feedB); err != nil {
		t.Fatal(err)
	}

	p.feedClient = fakeFeedClientByURL{
		"https://example.com/feed-a.xml": {
			{Title: "Shared", Link: "https://habr.com/ru/articles/1042858/?utm_campaign=1042858&utm_source=habrahabr&utm_medium=rss#comments"},
		},
		"https://example.com/feed-b.xml": {
			{Title: "Shared", Link: "https://habr.com/ru/articles/1042858/"},
		},
	}
	if err := p.ProcessFeed(context.Background(), feedA); err != nil {
		t.Fatal(err)
	}
	if err := p.ProcessFeed(context.Background(), feedB); err != nil {
		t.Fatal(err)
	}

	if len(tg.messages) != 1 {
		t.Fatalf("posted messages = %d", len(tg.messages))
	}
}
