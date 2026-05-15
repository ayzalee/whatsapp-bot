package plugins

import (
	"strings"
	"testing"
)

// T() ─────────────────────────────────────────────────────────────────────────

func TestT_DefaultIsEnglish(t *testing.T) {
	BotSettings.SetLanguage("en")
	s := T()
	if s == nil {
		t.Fatal("T() returned nil for 'en'")
	}
	if s.Pong != "Pong" {
		t.Fatalf("unexpected Pong value: %q", s.Pong)
	}
}

func TestT_KnownLanguage(t *testing.T) {
	BotSettings.SetLanguage("es")
	defer BotSettings.SetLanguage("en")
	s := T()
	if s == nil {
		t.Fatal("T() returned nil for 'es'")
	}
	if s.Pong != "Pong" {
		t.Fatalf("unexpected Pong: %q", s.Pong)
	}
}

func TestT_FallbackToEnglishOnUnknown(t *testing.T) {
	BotSettings.SetLanguage("xx")
	defer BotSettings.SetLanguage("en")
	s := T()
	if s == nil {
		t.Fatal("T() returned nil for unknown language")
	}
	if s != translations["en"] {
		t.Fatal("expected English fallback for unknown language code")
	}
}

func TestT_AllLanguagesHaveRequiredFields(t *testing.T) {
	for code, s := range translations {
		if s.Pong == "" {
			t.Errorf("lang %q: Pong is empty", code)
		}
		if s.SudoOnly == "" {
			t.Errorf("lang %q: SudoOnly is empty", code)
		}
		if s.MenuGreeting == "" {
			t.Errorf("lang %q: MenuGreeting is empty", code)
		}
		if s.LangSet == "" {
			t.Errorf("lang %q: LangSet is empty", code)
		}
	}
}

// availableLangs ──────────────────────────────────────────────────────────────

func TestAvailableLangs_ContainsAllCodes(t *testing.T) {
	result := availableLangs()
	for code := range translations {
		if !strings.Contains(result, code) {
			t.Errorf("availableLangs missing code %q", code)
		}
	}
}

func TestAvailableLangs_IsSorted(t *testing.T) {
	result := availableLangs()
	parts := strings.Split(result, ", ")
	for i := 1; i < len(parts); i++ {
		if parts[i] < parts[i-1] {
			t.Fatalf("availableLangs not sorted: %q before %q", parts[i-1], parts[i])
		}
	}
}

// langList ────────────────────────────────────────────────────────────────────

func TestLangList_Format(t *testing.T) {
	result := langList()
	if !strings.Contains(result, " - ") {
		t.Fatalf("langList should contain ' - ' separator, got: %s", result)
	}
}

func TestLangList_ContainsEnglish(t *testing.T) {
	result := langList()
	if !strings.Contains(result, "English - en") {
		t.Fatalf("langList should contain 'English - en', got: %s", result)
	}
}

func TestLangList_NoTrailingNewline(t *testing.T) {
	result := langList()
	if strings.HasSuffix(result, "\n") {
		t.Fatal("langList should not end with a newline")
	}
}

// LangNames ───────────────────────────────────────────────────────────────────

func TestLangNames_MatchTranslations(t *testing.T) {
	for code := range translations {
		if _, ok := LangNames[code]; !ok {
			t.Errorf("LangNames missing entry for language code %q", code)
		}
	}
}
