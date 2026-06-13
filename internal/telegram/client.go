package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	ChatID string
	Text   string
}

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func NewClient(token string, opts ...Option) *Client {
	c := &Client{
		token:   token,
		baseURL: "https://api.telegram.org",
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type sendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func (c *Client) SendMessage(ctx context.Context, msg Message) error {
	body, err := json.Marshal(sendMessageRequest{
		ChatID: msg.ChatID,
		Text:   msg.Text,
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !apiResp.OK {
		if apiResp.Description == "" {
			apiResp.Description = resp.Status
		}
		return fmt.Errorf("telegram sendMessage failed: %s", apiResp.Description)
	}
	return nil
}
