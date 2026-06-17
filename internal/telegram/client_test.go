package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSendMessagePostsToBotAPI(t *testing.T) {
	var req sendMessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123:secret/sendMessage" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer server.Close()

	client := NewClient("123:secret", WithBaseURL(server.URL))
	err := client.SendMessage(context.Background(), Message{
		ChatID: "@channel",
		Text:   "hello",
	})
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}

	if req.ChatID != "@channel" || req.Text != "hello" {
		t.Fatalf("request = %+v", req)
	}
}

func TestSendMessageReturnsTelegramError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	}))
	defer server.Close()

	client := NewClient("123:secret", WithBaseURL(server.URL))
	err := client.SendMessage(context.Background(), Message{ChatID: "@missing", Text: "hello"})
	if err == nil {
		t.Fatal("expected Telegram error")
	}
}

func TestSendMessageRedactsTokenFromNetworkErrors(t *testing.T) {
	client := NewClient("123:secret", WithHTTPClient(&http.Client{Transport: failingTransport{}}))
	err := client.SendMessage(context.Background(), Message{ChatID: "@channel", Text: "hello"})
	if err == nil {
		t.Fatal("expected network error")
	}

	got := err.Error()
	if strings.Contains(got, "123:secret") || strings.Contains(got, "/bot") {
		t.Fatalf("error leaks bot token: %q", got)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, &url.Error{
		Op:  "Post",
		URL: req.URL.String(),
		Err: errors.New("network down"),
	}
}
