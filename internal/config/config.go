package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPollInterval   = 5 * time.Minute
	defaultRequestTimeout = 20 * time.Second
	defaultDatabasePath   = "telegram-poster.sqlite"
)

type Config struct {
	TelegramToken  string
	PollInterval   time.Duration
	RequestTimeout time.Duration
	DatabasePath   string
	Feeds          []Feed
}

type Feed struct {
	Name     string
	URL      string
	Channel  string
	Interval time.Duration
}

type rawConfig struct {
	PollInterval   duration  `yaml:"poll_interval"`
	RequestTimeout duration  `yaml:"request_timeout"`
	DatabasePath   string    `yaml:"database_path"`
	Feeds          []rawFeed `yaml:"feeds"`
}

type rawFeed struct {
	Name     string   `yaml:"name"`
	URL      string   `yaml:"url"`
	Channel  string   `yaml:"channel"`
	Interval duration `yaml:"interval"`
}

type duration time.Duration

func (d *duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == 0 || value.Value == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return err
	}
	*d = duration(parsed)
	return nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, err
	}

	cfg := Config{
		TelegramToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		PollInterval:   time.Duration(raw.PollInterval),
		RequestTimeout: time.Duration(raw.RequestTimeout),
		DatabasePath:   raw.DatabasePath,
		Feeds:          make([]Feed, 0, len(raw.Feeds)),
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = defaultDatabasePath
	}

	for _, feed := range raw.Feeds {
		cfg.Feeds = append(cfg.Feeds, Feed{
			Name:     feed.Name,
			URL:      feed.URL,
			Channel:  feed.Channel,
			Interval: time.Duration(feed.Interval),
		})
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.TelegramToken == "" {
		return errors.New("TELEGRAM_BOT_TOKEN is required")
	}
	if len(c.Feeds) == 0 {
		return errors.New("at least one feed is required")
	}
	for i, feed := range c.Feeds {
		if feed.Name == "" {
			return fmt.Errorf("feeds[%d].name is required", i)
		}
		if feed.URL == "" {
			return fmt.Errorf("feeds[%d].url is required", i)
		}
		if err := ValidateFeedURL(feed.URL); err != nil {
			return fmt.Errorf("feeds[%d].url is unsafe: %w", i, err)
		}
		if feed.Channel == "" {
			return fmt.Errorf("feeds[%d].channel is required", i)
		}
	}
	return nil
}

func ValidateFeedURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" {
		return errors.New("only https URLs are allowed")
	}
	host := parsed.Hostname()
	if host == "" {
		return errors.New("host is required")
	}
	if strings.EqualFold(host, "localhost") {
		return errors.New("localhost is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			return errors.New("invalid IP address")
		}
		if IsBlockedIP(addr) {
			return errors.New("private, loopback, link-local, and unspecified IPs are not allowed")
		}
	}
	return nil
}

func IsBlockedIP(addr netip.Addr) bool {
	return addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsUnspecified()
}
