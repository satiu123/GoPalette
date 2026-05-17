package data

import (
	"testing"
	"time"
)

func TestParseIndexedTime(t *testing.T) {
	want := time.Date(2026, 5, 17, 2, 32, 50, 283000000, time.UTC)
	raw := map[string]any{
		"created_at": want.Format(time.RFC3339Nano),
	}

	got := parseIndexedTime(raw, "created_at", "createdAt")
	if !got.Equal(want) {
		t.Fatalf("parseIndexedTime() = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

func TestParseIndexedTimeSupportsCamelCase(t *testing.T) {
	want := time.Date(2026, 5, 17, 2, 32, 50, 0, time.UTC)
	raw := map[string]any{
		"createdAt": want.Format(time.RFC3339),
	}

	got := parseIndexedTime(raw, "created_at", "createdAt")
	if !got.Equal(want) {
		t.Fatalf("parseIndexedTime() = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
