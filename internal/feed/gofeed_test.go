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
