package feed

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchRejectsOversizedResponse(t *testing.T) {
	body := `<rss><channel><item><title>large</title><description>` +
		strings.Repeat("x", MaxFeedBytes+1) +
		`</description></item></channel></rss>`
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err == nil {
		t.Fatal("expected oversized feed error")
	}
	if !strings.Contains(err.Error(), "feed exceeds maximum size") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchSetsBrowserFriendlyHeaders(t *testing.T) {
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("User-Agent"); got != "TelegramPoster/1.0 (+https://github.com/puhhh/telegram-poster)" {
			t.Fatalf("user-agent = %q", got)
		}
		if got := req.Header.Get("Accept"); got != "application/rss+xml, application/atom+xml, application/xml, text/xml, */*;q=0.8" {
			t.Fatalf("accept = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`<rss><channel></channel></rss>`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
}
