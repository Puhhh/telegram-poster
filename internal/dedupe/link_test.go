package dedupe

import "testing"

func TestCanonicalLinkPreservesFragmentIdentity(t *testing.T) {
	first := CanonicalLink("https://lichess.org/feed#cJdb21")
	second := CanonicalLink("https://lichess.org/feed#A6LIjg")

	if first == second {
		t.Fatalf("expected fragment to distinguish feed item links, got %q", first)
	}
}

func TestCanonicalLinkStripsTrackingQueryParameters(t *testing.T) {
	got := CanonicalLink("HTTPS://Example.COM/post?utm_source=rss&fbclid=abc&id=1#section")
	want := "https://example.com/post?id=1#section"

	if got != want {
		t.Fatalf("CanonicalLink() = %q, want %q", got, want)
	}
}

func TestCanonicalLinkStripsCommentFragment(t *testing.T) {
	got := CanonicalLink("https://habr.com/ru/articles/1042858/?utm_source=rss#comments")
	want := "https://habr.com/ru/articles/1042858/"

	if got != want {
		t.Fatalf("CanonicalLink() = %q, want %q", got, want)
	}
}
