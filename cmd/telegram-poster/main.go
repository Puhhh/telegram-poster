package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-poster/internal/config"
	"telegram-poster/internal/feed"
	"telegram-poster/internal/poster"
	"telegram-poster/internal/state"
	"telegram-poster/internal/telegram"
)

type telegramAdapter struct {
	client *telegram.Client
}

func (a telegramAdapter) Send(ctx context.Context, msg poster.OutgoingMessage) error {
	return a.client.SendMessage(ctx, telegram.Message{ChatID: msg.ChatID, Text: msg.Text})
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := state.OpenSQLite(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open state: %v", err)
	}
	defer store.Close()

	httpClient := newHTTPClient(cfg.RequestTimeout)
	p := poster.New(
		feed.NewClient(httpClient),
		store,
		telegramAdapter{client: telegram.NewClient(cfg.TelegramToken, telegram.WithHTTPClient(httpClient))},
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run(ctx, cfg, p)
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: secureDialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return config.ValidateFeedURL(req.URL.String())
		},
	}
}

func secureDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if config.IsBlockedIP(ip) {
			continue
		}
		return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}
	return nil, net.InvalidAddrError("host resolves only to blocked IP ranges")
}

type processor interface {
	ProcessFeed(ctx context.Context, feed poster.Feed) error
}

func run(ctx context.Context, cfg config.Config, p processor) {
	nextRun := make(map[string]time.Time, len(cfg.Feeds))
	for _, feedCfg := range cfg.Feeds {
		nextRun[feedCfg.Name] = time.Now()
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		now := time.Now()
		processDueFeeds(ctx, cfg, p, nextRun, now)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func processDueFeeds(ctx context.Context, cfg config.Config, p processor, nextRun map[string]time.Time, now time.Time) {
	for _, feedCfg := range cfg.Feeds {
		if now.Before(nextRun[feedCfg.Name]) {
			continue
		}
		interval := feedCfg.Interval
		if interval == 0 {
			interval = cfg.PollInterval
		}
		nextRun[feedCfg.Name] = now.Add(interval)

		feed := poster.Feed{Name: feedCfg.Name, URL: feedCfg.URL, Channel: feedCfg.Channel}
		if err := p.ProcessFeed(ctx, feed); err != nil {
			log.Printf("process feed %q: %v", feed.Name, err)
			continue
		}
	}
}
