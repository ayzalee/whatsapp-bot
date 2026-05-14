package plugins

import (
	"testing"
)

// parseCommand ────────────────────────────────────────────────────────────────

func TestParseCommand_BasicMatch(t *testing.T) {
	prefix, name, rest, ok := parseCommand(".ping", []string{"."})
	if !ok || prefix != "." || name != "ping" || rest != "" {
		t.Fatalf("got prefix=%q name=%q rest=%q ok=%v", prefix, name, rest, ok)
	}
}

func TestParseCommand_WithArgs(t *testing.T) {
	_, name, rest, ok := parseCommand(".meta hello world", []string{"."})
	if !ok || name != "meta" || rest != "hello world" {
		t.Fatalf("name=%q rest=%q ok=%v", name, rest, ok)
	}
}

func TestParseCommand_SpaceBetweenPrefixAndName(t *testing.T) {
	_, name, _, ok := parseCommand(". ping", []string{"."})
	if !ok || name != "ping" {
		t.Fatalf("name=%q ok=%v", name, ok)
	}
}

func TestParseCommand_NoMatch(t *testing.T) {
	_, _, _, ok := parseCommand("ping", []string{"."})
	if ok {
		t.Fatal("expected no match for unprefixed text when prefix is required")
	}
}

func TestParseCommand_EmptyPrefix(t *testing.T) {
	prefix, name, _, ok := parseCommand("ping", []string{""})
	if !ok || prefix != "" || name != "ping" {
		t.Fatalf("prefix=%q name=%q ok=%v", prefix, name, ok)
	}
}

func TestParseCommand_SecondPrefixMatches(t *testing.T) {
	_, name, _, ok := parseCommand("/ping", []string{".", "/"})
	if !ok || name != "ping" {
		t.Fatalf("name=%q ok=%v", name, ok)
	}
}

func TestParseCommand_EmptyText(t *testing.T) {
	_, _, _, ok := parseCommand("", []string{"."})
	if ok {
		t.Fatal("expected no match for empty text")
	}
}

func TestParseCommand_CaseInsensitive(t *testing.T) {
	_, name, _, ok := parseCommand(".PING", []string{"."})
	if !ok || name != "ping" {
		t.Fatalf("name=%q ok=%v", name, ok)
	}
}

// findCommand / Register ──────────────────────────────────────────────────────

func TestFindCommand_ByPattern(t *testing.T) {
	if findCommand("ping") == nil {
		t.Fatal("expected to find 'ping' command")
	}
}

func TestFindCommand_ByAlias(t *testing.T) {
	cmd := findCommand("help")
	if cmd == nil {
		t.Fatal("expected to find command via alias 'help'")
	}
	if cmd.Pattern != "menu" {
		t.Fatalf("expected pattern 'menu', got %q", cmd.Pattern)
	}
}

func TestFindCommand_HelpAlias(t *testing.T) {
	cmd := findCommand("help")
	if cmd == nil {
		t.Fatal("expected to find command via alias 'help'")
	}
	if cmd.Pattern != "menu" {
		t.Fatalf("expected pattern 'menu', got %q", cmd.Pattern)
	}
}

func TestFindCommand_NotFound(t *testing.T) {
	if findCommand("doesnotexist") != nil {
		t.Fatal("expected nil for unknown command")
	}
}

func TestRegister_MapsPopulated(t *testing.T) {
	// registryMap and categoryMap are built during init(); verify both.
	if len(registryMap) == 0 {
		t.Fatal("registryMap is empty after init")
	}
	if len(categoryMap) == 0 {
		t.Fatal("categoryMap is empty after init")
	}
}

func TestCategoryMap_ContainsKnownCategories(t *testing.T) {
	for _, cat := range []string{"utility", "ai", "settings"} {
		if len(categoryMap[cat]) == 0 {
			t.Errorf("categoryMap missing category %q", cat)
		}
	}
}
