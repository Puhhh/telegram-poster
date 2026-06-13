package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReadsYAMLAndTokenFromEnv(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:secret")
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(`
poll_interval: 2m
request_timeout: 15s
database_path: poster.sqlite
feeds:
  - name: news
    url: https://example.com/rss.xml
    channel: "@news_channel"
    interval: 30s
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.TelegramToken != "123:secret" {
		t.Fatalf("token = %q", cfg.TelegramToken)
	}
	if cfg.PollInterval != 2*time.Minute {
		t.Fatalf("poll interval = %s", cfg.PollInterval)
	}
	if cfg.RequestTimeout != 15*time.Second {
		t.Fatalf("request timeout = %s", cfg.RequestTimeout)
	}
	if cfg.DatabasePath != "poster.sqlite" {
		t.Fatalf("database path = %q", cfg.DatabasePath)
	}
	if got := cfg.Feeds[0]; got.Name != "news" || got.URL != "https://example.com/rss.xml" || got.Channel != "@news_channel" || got.Interval != 30*time.Second {
		t.Fatalf("feed = %+v", got)
	}
}

func TestLoadValidatesRequiredFields(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(`feeds: []`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadRejectsUnsafeFeedURLs(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:secret")
	tests := map[string]string{
		"plain_http": "http://example.com/rss.xml",
		"loopback":   "https://127.0.0.1/rss.xml",
		"private":    "https://10.0.0.1/rss.xml",
	}

	for name, feedURL := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			err := os.WriteFile(path, []byte("feeds:\n  - name: unsafe\n    url: "+feedURL+"\n    channel: \"@news\"\n"), 0o600)
			if err != nil {
				t.Fatal(err)
			}

			_, err = Load(path)
			if err == nil {
				t.Fatal("expected unsafe URL validation error")
			}
		})
	}
}
