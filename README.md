# Telegram Poster

> RSS/Atom/JSON feed daemon that posts new entries to Telegram channels.

Telegram Poster polls configured feeds, remembers already-seen items in SQLite, and sends only new entries to the channel mapped to each feed. Duplicate entries are suppressed globally across all feeds so the same article is posted only once. It is designed to run as a small `systemd` service on a Linux VPS.

Use it when you want a small self-hosted feed-to-Telegram bridge without a database server, a web UI, or long-lived credentials in config files.

## Quick Start

Prerequisites:

- Go 1.25 or newer
- Telegram bot token from `@BotFather`
- Bot added as an admin to every target channel

Build locally:

```sh
go build -o telegram-poster ./cmd/telegram-poster
```

Or download a Linux VPS binary from the latest GitHub Release:

```sh
curl -L -o telegram-poster https://github.com/Puhhh/telegram-poster/releases/latest/download/telegram-poster-linux-amd64
chmod +x telegram-poster
```

For ARM VPS hosts, download `telegram-poster-linux-arm64` instead. Each release also publishes `SHA256SUMS` and GitHub artifact attestations for the Linux binaries.

Create `config.yaml`:

```sh
cp config.example.yaml config.yaml
```

Then edit the feeds:

```yaml
poll_interval: 5m
request_timeout: 20s
database_path: telegram-poster.sqlite

feeds:
  - name: example-news
    url: https://example.com/rss.xml
    channel: "@example_channel"
    interval: 2m
```

Run:

```sh
export TELEGRAM_BOT_TOKEN='123456:your_token_here'
./telegram-poster -config config.yaml
```

On first run, existing RSS items are marked as seen and are not posted. Only new items after that are sent. If the same article appears in more than one configured feed, the first feed that sees it wins and later copies are skipped.

Each Telegram message contains the feed item title and source link. Long messages are truncated to fit Telegram's 4096-character message limit while keeping the link when possible.

## Configuration

`config.yaml` controls polling, storage, and feed routing.

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `poll_interval` | No | `5m` | Default interval for feeds without their own `interval`. |
| `request_timeout` | No | `20s` | HTTP timeout for feed and Telegram API requests. |
| `database_path` | No | `telegram-poster.sqlite` | SQLite state file path. |
| `feeds[].name` | Yes | | Stable feed identifier used for first-run state and logs. |
| `feeds[].url` | Yes | | Feed URL. Must be `https://`. |
| `feeds[].channel` | Yes | | Telegram channel username, for example `@example_channel`. |
| `feeds[].interval` | No | `poll_interval` | Per-feed polling interval. |

Security rules for feed URLs:

- only `https://` URLs are allowed
- `localhost`, private IPs, loopback IPs, link-local IPs, and unspecified IPs are rejected
- redirects are checked with the same policy
- feed responses are capped at 5 MiB

Deduplication uses the item link when available, with common tracking parameters such as `utm_*`, `fbclid`, `gclid`, and `yclid` ignored. If no link is available, the daemon falls back to the item GUID, then a title/summary hash.

Keep the bot token outside `config.yaml`:

```sh
export TELEGRAM_BOT_TOKEN='123456:your_token_here'
```

## Linux VPS Deployment

These commands assume:

- local machine builds the binary
- VPS user is `user`
- VPS host is `vps`
- target Linux architecture is `amd64`

### 1. Prepare Telegram

Create a bot with `@BotFather`, then add it as an admin to each target channel. It needs permission to post messages.

### 2. Build and Copy Files

Download the release binary:

```sh
curl -L -o telegram-poster https://github.com/Puhhh/telegram-poster/releases/latest/download/telegram-poster-linux-amd64
chmod +x telegram-poster
```

For ARM VPS, use `telegram-poster-linux-arm64`.

You can verify the downloaded binary with the release `SHA256SUMS` file:

```sh
curl -L -o SHA256SUMS https://github.com/Puhhh/telegram-poster/releases/latest/download/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
```

Or build locally:

```sh
GOOS=linux GOARCH=amd64 go build -o telegram-poster ./cmd/telegram-poster
```

Copy files:

```sh
scp telegram-poster user@vps:/tmp/
scp config.yaml user@vps:/tmp/
scp deploy/telegram-poster.service user@vps:/tmp/
```

For ARM VPS local builds, use `GOARCH=arm64`.

### 3. Install on the VPS

```sh
sudo useradd --system --home /opt/telegram-poster --shell /usr/sbin/nologin telegram-poster
sudo mkdir -p /opt/telegram-poster /etc/telegram-poster
sudo mv /tmp/telegram-poster /usr/local/bin/telegram-poster
sudo mv /tmp/config.yaml /etc/telegram-poster/config.yaml
sudo chmod +x /usr/local/bin/telegram-poster
sudo chown -R telegram-poster:telegram-poster /opt/telegram-poster
sudo chown root:telegram-poster /etc/telegram-poster/config.yaml
sudo chmod 640 /etc/telegram-poster/config.yaml
```

For the provided hardened `systemd` unit, set the VPS database path in `/etc/telegram-poster/config.yaml`:

```yaml
database_path: /opt/telegram-poster/telegram-poster.sqlite
```

### 4. Store the Token

```sh
sudo vi /etc/telegram-poster/env
sudo chown root:telegram-poster /etc/telegram-poster/env
sudo chmod 640 /etc/telegram-poster/env
```

`/etc/telegram-poster/env`:

```sh
TELEGRAM_BOT_TOKEN=123456:your_token_here
```

### 5. Start the Service

```sh
sudo mv /tmp/telegram-poster.service /etc/systemd/system/telegram-poster.service
sudo systemctl daemon-reload
sudo systemctl enable telegram-poster
sudo systemctl start telegram-poster
```

Check status and logs:

```sh
sudo systemctl status telegram-poster
sudo journalctl -u telegram-poster -f
```

## Updating the VPS

Build a new binary locally:

```sh
GOOS=linux GOARCH=amd64 go build -o telegram-poster ./cmd/telegram-poster
scp telegram-poster user@vps:/tmp/telegram-poster
```

Or download the latest release binary:

```sh
curl -L -o telegram-poster https://github.com/Puhhh/telegram-poster/releases/latest/download/telegram-poster-linux-amd64
chmod +x telegram-poster
scp telegram-poster user@vps:/tmp/telegram-poster
```

Replace it on the VPS:

```sh
sudo systemctl stop telegram-poster
sudo mv /tmp/telegram-poster /usr/local/bin/telegram-poster
sudo chmod +x /usr/local/bin/telegram-poster
sudo systemctl start telegram-poster
```

If `deploy/telegram-poster.service` changed, copy and reload it too:

```sh
scp deploy/telegram-poster.service user@vps:/tmp/
sudo mv /tmp/telegram-poster.service /etc/systemd/system/telegram-poster.service
sudo systemctl daemon-reload
sudo systemctl restart telegram-poster
```

## Troubleshooting

View live logs:

```sh
sudo journalctl -u telegram-poster -f
```

Common cases:

- `TELEGRAM_BOT_TOKEN is required`: `/etc/telegram-poster/env` is missing, unreadable, or does not define the token.
- `telegram sendMessage failed: Bad Request: chat not found`: channel username is wrong or the bot is not a channel admin.
- `feed returned 403 Forbidden`: feed blocks the server. The daemon logs the error and continues with other feeds.
- `url is unsafe`: feed URL is not `https://` or points to a blocked local/private address.
- no posts after first start: expected. First run marks existing feed items as seen.

## Development

Run tests:

```sh
go test ./...
```

Run dependency vulnerability check:

```sh
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Build a local binary:

```sh
go build -o telegram-poster ./cmd/telegram-poster
```
