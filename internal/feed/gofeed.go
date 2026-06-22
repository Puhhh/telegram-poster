package feed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"telegram-poster/internal/poster"
)

const MaxFeedBytes = 5 << 20
const maxFetchAttempts = 2
const (
	defaultUserAgent = "TelegramPoster/1.0 (+https://github.com/puhhh/telegram-poster)"
	defaultAccept    = "application/rss+xml, application/atom+xml, application/xml, text/xml, */*;q=0.8"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func (c *Client) Fetch(ctx context.Context, url string) ([]poster.FeedItem, error) {
	for attempt := 1; attempt <= maxFetchAttempts; attempt++ {
		items, retryable, err := c.fetchOnce(ctx, url, attempt)
		if err == nil {
			return items, nil
		}
		if !retryable || attempt == maxFetchAttempts {
			return nil, err
		}
	}
	return nil, errors.New("unreachable fetch retry state")
}

func (c *Client) fetchOnce(ctx context.Context, url string, attempt int) ([]poster.FeedItem, bool, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", defaultAccept)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, isRetryableFeedError(err), fetchError(attempt, "", 0, start, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fetchError(attempt, resp.Status, 0, start, fmt.Errorf("feed returned %s", resp.Status))
	}

	reader := &limitedReader{
		reader:    resp.Body,
		remaining: MaxFeedBytes,
	}
	parsed, err := gofeed.NewParser().Parse(reader)
	if err != nil {
		return nil, isRetryableFeedError(err), fetchError(attempt, resp.Status, reader.read, start, err)
	}
	items := make([]poster.FeedItem, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		items = append(items, poster.FeedItem{
			GUID:    item.GUID,
			Title:   item.Title,
			Summary: firstNonEmpty(item.Description, item.Content),
			Link:    item.Link,
		})
	}
	return items, false, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type limitedReader struct {
	reader    io.Reader
	remaining int
	read      int
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, fmt.Errorf("feed exceeds maximum size of %d bytes", MaxFeedBytes)
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.reader.Read(p)
	r.remaining -= n
	r.read += n
	if err == io.EOF && r.remaining == 0 {
		return n, fmt.Errorf("feed exceeds maximum size of %d bytes", MaxFeedBytes)
	}
	return n, err
}

func isRetryableFeedError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "unexpected eof")
}

func fetchError(attempt int, status string, bytesRead int, start time.Time, err error) error {
	elapsed := time.Since(start).Round(time.Millisecond)
	if status == "" {
		status = "no response"
	}
	return fmt.Errorf("attempt %d/%d, status %s, read %d bytes in %s: %w", attempt, maxFetchAttempts, status, bytesRead, elapsed, err)
}
