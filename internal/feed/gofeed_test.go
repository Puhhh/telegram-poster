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
	requests := 0
	body := `<rss><channel><item><title>large</title><description>` +
		strings.Repeat("x", MaxFeedBytes+1) +
		`</description></item></channel></rss>`
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
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
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
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

func TestFetchRetriesTruncatedXML(t *testing.T) {
	requests := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if got := req.Header.Get("User-Agent"); got != "TelegramPoster/1.0 (+https://github.com/puhhh/telegram-poster)" {
			t.Fatalf("user-agent = %q", got)
		}
		if got := req.Header.Get("Accept"); got != "application/rss+xml, application/atom+xml, application/xml, text/xml, */*;q=0.8" {
			t.Fatalf("accept = %q", got)
		}
		body := `<rss><channel><item><title>cut</title>`
		if requests == 2 {
			body = `<rss><channel><item><title>ok</title><link>https://example.com/ok</link></item></channel></rss>`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})})

	items, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Title != "ok" {
		t.Fatalf("title = %q, want ok", items[0].Title)
	}
}

func TestFetchDoesNotRetryHTTPStatusErrors(t *testing.T) {
	requests := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Status:     "403 Forbidden",
			Body:       io.NopCloser(strings.NewReader("forbidden")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err == nil {
		t.Fatal("expected HTTP status error")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestFetchIncludesAttemptMetadataAfterRetryFailure(t *testing.T) {
	requests := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`<rss><channel><item><title>cut</title>`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err == nil {
		t.Fatal("expected truncated XML error")
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	for _, want := range []string{"attempt 2/2", "status 200 OK", "read 38 bytes", "unexpected EOF"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err.Error(), want)
		}
	}
}
