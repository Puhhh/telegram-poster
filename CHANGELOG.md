# Changelog

All notable changes to Telegram Poster are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project uses semantic versioning once releases begin.

## [Unreleased]

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
