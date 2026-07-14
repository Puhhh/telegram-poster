# Changelog

All notable changes to Telegram Poster are documented in this file.

## [Unreleased]

### Fixed

- Parse valid RSS or Atom bodies returned with redirect responses instead of following the redirect to a non-feed page, while preserving redirect URL validation.

## [0.1.5] - 2026-06-22

### Fixed

- Retry a feed fetch once when XML parsing fails with an EOF-like truncated response, and include fetch attempt metadata in the returned error.

## [0.1.4] - 2026-06-21

### Fixed

- Store newly delivered feed items as seen and posted in one SQLite write so a crash or database error cannot leave them permanently skipped with an empty `posted_at`.
- Collapse legacy raw tracking-URL rows into their canonical deduplication key during SQLite migration.

## [0.1.3] - 2026-06-20

### Fixed

- Preserve RSS item link fragments when they identify distinct feed entries, fixing feeds such as Lichess that publish multiple items under the same base URL with different fragments.
- Keep comment-style fragments such as `#comments` ignored during link deduplication so common article comment anchors do not create duplicate posts.
- Retry posting a new item after a Telegram `sendMessage` timeout or network failure instead of marking it as seen before delivery succeeds.

## [0.1.2] - 2026-06-17

### Changed

- Feed fetches now send an explicit app `User-Agent` and RSS `Accept` header to avoid common `403 Forbidden` blocks from feeds that reject Go's default HTTP client fingerprint.

## [0.1.1] - 2026-06-17

### Changed

- Updated the release workflow to use `actions/checkout@v6` and `actions/setup-go@v6`.
- Split the release workflow into read-only build/test and write-scoped publish jobs, with release artifacts passed between jobs.
- Pinned `govulncheck` in the release workflow to `v1.3.0`.
- Expanded README setup, release artifact, checksum verification, and local build guidance.

### Security

- Prevented Telegram network errors from including the bot token URL in returned errors and logs.

## [0.1.0] - 2026-06-16

### Added

- Initial Go daemon for polling RSS, Atom, and JSON feeds.
- Telegram posting through the Bot API `sendMessage` endpoint.
- YAML configuration for feed routes, polling interval, request timeout, and SQLite path.
- `TELEGRAM_BOT_TOKEN` environment variable support so bot tokens stay out of config files.
- One-feed-to-one-channel routing with optional per-feed polling interval.
- SQLite state store for deduplicating RSS items across restarts.
- First-run behavior that marks existing feed items as seen without posting old entries.
- Message formatting with title and link only.
- Linux `systemd` service template for VPS deployment.
- README guide for local runs, VPS deployment, updates, troubleshooting, and security behavior.
- Tests for config loading, message formatting, SQLite state, Telegram API requests, feed processing, daemon error handling, URL policy, redirect policy, and oversized feed rejection.

### Changed

- Go module now requires Go 1.25 or newer.
- Telegram post text now uses title and link only.
- RSS item deduplication is now global across all feeds, with common tracking parameters ignored when comparing links.
- Feed errors such as `403 Forbidden` are logged per feed without stopping the daemon or blocking other feeds.
- `golang.org/x/net` is pinned to `v0.55.0` to avoid reachable parser vulnerabilities reported by `govulncheck`.

### Security

- Feed URLs must use `https://`.
- Feed URLs reject `localhost`, private IPs, loopback IPs, link-local IPs, and unspecified IPs.
- HTTP redirects are validated with the same feed URL policy.
- Feed response bodies are capped at 5 MiB before parsing.
- `systemd` service template runs under a dedicated user with hardening options including `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem=strict`, and `ProtectHome=true`.
- Deployment docs set restrictive ownership and permissions for `/etc/telegram-poster/config.yaml` and `/etc/telegram-poster/env`.
- `govulncheck` currently reports no reachable vulnerabilities.
