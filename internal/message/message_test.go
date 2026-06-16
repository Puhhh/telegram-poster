package message

import (
	"strings"
	"testing"
)

func TestFormatBuildsTitleSummaryAndLink(t *testing.T) {
	got := Format(Item{
		Title:   "Hello <World>",
		Summary: "<p>One&nbsp;<b>two</b> &amp; three</p>",
		Link:    "https://example.com/post",
	})

	want := "Hello <World>\n\nOne two & three\n\nhttps://example.com/post"
	if got != want {
		t.Fatalf("message mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestFormatTruncatesLongMessageKeepingLink(t *testing.T) {
	got := Format(Item{
		Title:   strings.Repeat("x", MaxTelegramMessageRunes),
		Summary: "Summary",
		Link:    "https://example.com/post",
	})

	if len([]rune(got)) > MaxTelegramMessageRunes {
		t.Fatalf("message length = %d", len([]rune(got)))
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("expected ellipsis in %q", got)
	}
	if !strings.HasSuffix(got, "https://example.com/post") {
		t.Fatalf("expected link suffix in %q", got)
	}
}

func TestFormatShortensSummary(t *testing.T) {
	got := Format(Item{
		Title:   "Title",
		Summary: strings.Repeat("я", MaxSummaryRunes+50),
		Link:    "https://example.com/post",
	})

	want := "Title\n\n" + strings.Repeat("я", MaxSummaryRunes) + "…\n\nhttps://example.com/post"
	if got != want {
		t.Fatalf("message mismatch\nwant: %q\n got: %q", want, got)
	}
}
