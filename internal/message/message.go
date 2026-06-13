package message

import (
	"html"
	"regexp"
	"strings"
)

const MaxTelegramMessageRunes = 4096

var (
	tagPattern        = regexp.MustCompile(`<[^>]*>`)
	whitespacePattern = regexp.MustCompile(`\s+`)
)

type Item struct {
	Title   string
	Summary string
	Link    string
}

func Format(item Item) string {
	title := cleanPlainText(item.Title)
	summary := cleanText(item.Summary)
	link := strings.TrimSpace(item.Link)

	parts := make([]string, 0, 3)
	if title != "" {
		parts = append(parts, title)
	}
	if summary != "" {
		parts = append(parts, summary)
	}
	if link != "" {
		parts = append(parts, link)
	}

	msg := strings.Join(parts, "\n\n")
	return truncateKeepingLink(msg, link)
}

func cleanText(value string) string {
	value = tagPattern.ReplaceAllString(value, " ")
	return cleanPlainText(value)
}

func cleanPlainText(value string) string {
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = whitespacePattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func truncateKeepingLink(msg, link string) string {
	runes := []rune(msg)
	if len(runes) <= MaxTelegramMessageRunes {
		return msg
	}
	if link == "" {
		return string(runes[:MaxTelegramMessageRunes-1]) + "…"
	}

	suffix := "\n\n" + link
	suffixRunes := []rune(suffix)
	limit := MaxTelegramMessageRunes - len(suffixRunes) - 1
	if limit < 0 {
		return string([]rune(link)[:min(len([]rune(link)), MaxTelegramMessageRunes)])
	}
	prefix := strings.TrimSpace(string(runes[:limit]))
	return prefix + "…" + suffix
}
