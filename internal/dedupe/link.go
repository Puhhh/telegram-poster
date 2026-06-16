package dedupe

import (
	"net/url"
	"strings"
)

func CanonicalLink(link string) string {
	trimmed := strings.TrimSpace(link)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	query := parsed.Query()
	for name := range query {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "yclid" {
			query.Del(name)
		}
	}
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String()
}
