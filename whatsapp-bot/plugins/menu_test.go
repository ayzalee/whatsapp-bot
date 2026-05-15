package plugins

import (
	"strings"
	"testing"
)

// toFancy ─────────────────────────────────────────────────────────────────────

func TestToFancy_Letters(t *testing.T) {
	got := toFancy("ping")
	if got != "ᴘɪɴɢ" {
		t.Fatalf("got %q", got)
	}
}

func TestToFancy_UppercaseNormalised(t *testing.T) {
	if toFancy("PING") != toFancy("ping") {
		t.Fatal("toFancy should be case-insensitive")
	}
}

func TestToFancy_Digits(t *testing.T) {
	got := toFancy("123")
	if got != "𝟷𝟸𝟹" {
		t.Fatalf("got %q", got)
	}
}

func TestToFancy_Mixed(t *testing.T) {
	got := toFancy("ping1")
	if !strings.HasPrefix(got, "ᴘɪɴɢ") {
		t.Fatalf("unexpected result: %q", got)
	}
}

func TestToFancy_NonAlpha_PassThrough(t *testing.T) {
	got := toFancy("!")
	if got != "!" {
		t.Fatalf("non-alpha char should pass through unchanged, got %q", got)
	}
}

// CategoryMenu ────────────────────────────────────────────────────────────────

func TestCategoryMenu_Known(t *testing.T) {
	result := CategoryMenu("utility")
	if result == "" {
		t.Fatal("expected non-empty menu for 'utility'")
	}
	if !strings.Contains(result, "ᴍᴇɴᴜ") {
		t.Fatalf("expected 'ᴍᴇɴᴜ' in result, got: %s", result)
	}
}

func TestCategoryMenu_CaseInsensitive(t *testing.T) {
	lower := CategoryMenu("utility")
	upper := CategoryMenu("UTILITY")
	if lower != upper {
		t.Fatal("CategoryMenu should be case-insensitive")
	}
}

func TestCategoryMenu_Unknown(t *testing.T) {
	if CategoryMenu("doesnotexist") != "" {
		t.Fatal("expected empty string for unknown category")
	}
}

func TestCategoryMenu_ContainsBoldHeader(t *testing.T) {
	result := CategoryMenu("settings")
	if !strings.HasPrefix(result, "*") {
		t.Fatalf("expected bold header (starts with *), got: %s", result)
	}
}

func TestCategoryMenu_ListsCommands(t *testing.T) {
	result := CategoryMenu("utility")
	if !strings.Contains(result, "·") {
		t.Fatalf("expected bullet points in menu, got: %s", result)
	}
}
