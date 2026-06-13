package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"telegram-poster/internal/config"
	"telegram-poster/internal/poster"
)

type recordingProcessor struct {
	calls []string
	fail  map[string]error
}

func TestHTTPClientRejectsUnsafeRedirect(t *testing.T) {
	client := newHTTPClient(time.Second)
	req := &http.Request{URL: mustParseURL(t, "http://127.0.0.1/feed.xml")}

	err := client.CheckRedirect(req, nil)
	if err == nil {
		t.Fatal("expected unsafe redirect error")
	}
}

func TestSecureDialContextRejectsPrivateIPLiteral(t *testing.T) {
	_, err := secureDialContext(context.Background(), "tcp", "127.0.0.1:443")
	if err == nil {
		t.Fatal("expected blocked IP error")
	}
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func (p *recordingProcessor) ProcessFeed(ctx context.Context, feed poster.Feed) error {
	p.calls = append(p.calls, feed.Name)
	if err := p.fail[feed.Name]; err != nil {
		return err
	}
	return nil
}

func TestProcessDueFeedsContinuesAfterFeedError(t *testing.T) {
	processor := &recordingProcessor{
		fail: map[string]error{
			"forbidden": errors.New("feed returned 403 Forbidden"),
		},
	}
	now := time.Date(2026, 6, 13, 18, 40, 0, 0, time.UTC)
	nextRun := map[string]time.Time{
		"forbidden": now,
		"next":      now,
	}

	processDueFeeds(context.Background(), config.Config{
		PollInterval: time.Minute,
		Feeds: []config.Feed{
			{Name: "forbidden", URL: "https://example.com/forbidden.xml", Channel: "@aiitsec_rss"},
			{Name: "next", URL: "https://example.com/next.xml", Channel: "@aiitsec_rss"},
		},
	}, processor, nextRun, now)

	if len(processor.calls) != 2 {
		t.Fatalf("calls = %v", processor.calls)
	}
	if processor.calls[0] != "forbidden" || processor.calls[1] != "next" {
		t.Fatalf("calls = %v", processor.calls)
	}
}
