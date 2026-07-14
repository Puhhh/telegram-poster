package feed

import (
	"context"
	"errors"
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

func TestFetchParsesFeedBodyReturnedWithRedirect(t *testing.T) {
	requests := 0
	redirectChecked := false
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{
				StatusCode: http.StatusFound,
				Status:     "302 Found",
				Body: io.NopCloser(strings.NewReader(
					`<rss><channel><item><title>redirect feed</title><link>https://example.com/item</link></item></channel></rss>`,
				)),
				Header: http.Header{
					"Content-Type": []string{"application/rss+xml; charset=UTF-8"},
					"Location":     []string{"https://example.com/"},
				},
				Request: req,
			}, nil
		}),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectChecked = true
			if got := req.URL.String(); got != "https://example.com/" {
				t.Fatalf("redirect URL = %q", got)
			}
			return nil
		},
	})

	items, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if !redirectChecked {
		t.Fatal("redirect policy was not checked")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if len(items) != 1 || items[0].Title != "redirect feed" {
		t.Fatalf("items = %+v", items)
	}
}

func TestFetchFollowsOrdinaryRedirect(t *testing.T) {
	requests := 0
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.URL.Path == "/feed.xml" {
				return &http.Response{
					StatusCode: http.StatusFound,
					Status:     "302 Found",
					Body:       io.NopCloser(strings.NewReader(`<html>redirecting</html>`)),
					Header: http.Header{
						"Content-Type": []string{"text/html"},
						"Location":     []string{"https://example.com/final.xml"},
					},
					Request: req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body: io.NopCloser(strings.NewReader(
					`<rss><channel><item><title>final feed</title></item></channel></rss>`,
				)),
				Header:  make(http.Header),
				Request: req,
			}, nil
		}),
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	})

	items, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if len(items) != 1 || items[0].Title != "final feed" {
		t.Fatalf("items = %+v", items)
	}
}

func TestFetchRejectsUnsafeFeedRedirectBeforeParsing(t *testing.T) {
	requests := 0
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{
				StatusCode: http.StatusFound,
				Status:     "302 Found",
				Body:       io.NopCloser(strings.NewReader(`<rss><channel></channel></rss>`)),
				Header: http.Header{
					"Content-Type": []string{"application/rss+xml"},
					"Location":     []string{"http://127.0.0.1/feed.xml"},
				},
				Request: req,
			}, nil
		}),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("unsafe redirect")
		},
	})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err == nil || !strings.Contains(err.Error(), "unsafe redirect") {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestFetchRejectsErrorStatusWithFeedMediaType(t *testing.T) {
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Body:       io.NopCloser(strings.NewReader(`<rss><channel></channel></rss>`)),
			Header:     http.Header{"Content-Type": []string{"application/rss+xml"}},
			Request:    req,
		}, nil
	})})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err == nil || !strings.Contains(err.Error(), "feed returned 404 Not Found") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchStopsAfterTenOrdinaryRedirects(t *testing.T) {
	requests := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusFound,
			Status:     "302 Found",
			Body:       io.NopCloser(strings.NewReader(`<html>redirecting</html>`)),
			Header: http.Header{
				"Content-Type": []string{"text/html"},
				"Location":     []string{"https://example.com/feed.xml"},
			},
			Request: req,
		}, nil
	})})

	_, err := client.Fetch(context.Background(), "https://example.com/feed.xml")
	if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != maxRedirects {
		t.Fatalf("requests = %d, want %d", requests, maxRedirects)
	}
}
