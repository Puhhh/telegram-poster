package feed

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/mmcdole/gofeed"

	"telegram-poster/internal/poster"
)

const MaxFeedBytes = 5 << 20

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func (c *Client) Fetch(ctx context.Context, url string) ([]poster.FeedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feed returned %s", resp.Status)
	}

	parsed, err := gofeed.NewParser().Parse(&limitedReader{
		reader:    resp.Body,
		remaining: MaxFeedBytes,
	})
	if err != nil {
		return nil, err
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
	return items, nil
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
	if err == io.EOF && r.remaining == 0 {
		return n, fmt.Errorf("feed exceeds maximum size of %d bytes", MaxFeedBytes)
	}
	return n, err
}
