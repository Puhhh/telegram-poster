package poster

import (
	"context"
	"crypto/sha256"
	"fmt"

	"telegram-poster/internal/dedupe"
	"telegram-poster/internal/message"
	"telegram-poster/internal/state"
)

type Feed struct {
	Name    string
	URL     string
	Channel string
}

type FeedItem struct {
	GUID    string
	Title   string
	Summary string
	Link    string
}

type OutgoingMessage struct {
	ChatID string
	Text   string
}

type FeedClient interface {
	Fetch(ctx context.Context, url string) ([]FeedItem, error)
}

type StateStore interface {
	IsSeen(itemKey string) (bool, error)
	MarkSeen(feedName, itemKey string, meta state.ItemMeta) error
	MarkPosted(itemKey string) error
	IsFeedInitialized(feedName string) (bool, error)
	MarkFeedInitialized(feedName string) error
}

type TelegramSender interface {
	Send(ctx context.Context, msg OutgoingMessage) error
}

type Poster struct {
	feedClient FeedClient
	store      StateStore
	telegram   TelegramSender
}

func New(feedClient FeedClient, store StateStore, telegram TelegramSender) *Poster {
	return &Poster{feedClient: feedClient, store: store, telegram: telegram}
}

func (p *Poster) ProcessFeed(ctx context.Context, feed Feed) error {
	items, err := p.feedClient.Fetch(ctx, feed.URL)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", feed.Name, err)
	}

	initialized, err := p.store.IsFeedInitialized(feed.Name)
	if err != nil {
		return err
	}
	if !initialized {
		for _, item := range items {
			if err := p.store.MarkSeen(feed.Name, itemKey(item), state.ItemMeta{Title: item.Title, Link: item.Link}); err != nil {
				return err
			}
		}
		return p.store.MarkFeedInitialized(feed.Name)
	}

	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		key := itemKey(item)
		seen, err := p.store.IsSeen(key)
		if err != nil {
			return err
		}
		if seen {
			continue
		}
		if err := p.store.MarkSeen(feed.Name, key, state.ItemMeta{Title: item.Title, Link: item.Link}); err != nil {
			return err
		}
		text := message.Format(message.Item{Title: item.Title, Summary: item.Summary, Link: item.Link})
		if err := p.telegram.Send(ctx, OutgoingMessage{ChatID: feed.Channel, Text: text}); err != nil {
			return err
		}
		if err := p.store.MarkPosted(key); err != nil {
			return err
		}
	}
	return nil
}

func itemKey(item FeedItem) string {
	if item.Link != "" {
		return dedupe.CanonicalLink(item.Link)
	}
	if item.GUID != "" {
		return item.GUID
	}
	sum := sha256.Sum256([]byte(item.Title + "\x00" + item.Summary))
	return fmt.Sprintf("%x", sum[:])
}
