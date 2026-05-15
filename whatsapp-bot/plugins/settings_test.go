package plugins

import (
	"testing"
)

func newSettings() *Settings {
	return &Settings{
		Prefixes: []string{"."},
		Sudoers:  []string{},
		Mode:     ModePublic,
		Language: "en",
	}
}

// IsSudo ──────────────────────────────────────────────────────────────────────

func TestIsSudo_EmptySudoers(t *testing.T) {
	s := newSettings()
	if s.IsSudo("123") {
		t.Fatal("expected false with empty sudoers")
	}
}

func TestIsSudo_Found(t *testing.T) {
	s := newSettings()
	s.Sudoers = []string{"111", "222"}
	if !s.IsSudo("222") {
		t.Fatal("expected true for existing sudoer")
	}
}

func TestIsSudo_NotFound(t *testing.T) {
	s := newSettings()
	s.Sudoers = []string{"111"}
	if s.IsSudo("999") {
		t.Fatal("expected false for non-sudoer")
	}
}

// AddSudo / RemoveSudo ────────────────────────────────────────────────────────

func TestAddSudo_AddsEntry(t *testing.T) {
	s := newSettings()
	s.AddSudo("555")
	if !s.IsSudo("555") {
		t.Fatal("expected sudoer after AddSudo")
	}
}

func TestAddSudo_NoDuplicate(t *testing.T) {
	s := newSettings()
	s.AddSudo("555")
	s.AddSudo("555")
	count := 0
	for _, p := range s.Sudoers {
		if p == "555" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 entry, got %d", count)
	}
}

func TestRemoveSudo_Removes(t *testing.T) {
	s := newSettings()
	s.AddSudo("555")
	removed := s.RemoveSudo("555")
	if !removed {
		t.Fatal("expected RemoveSudo to return true")
	}
	if s.IsSudo("555") {
		t.Fatal("expected sudoer to be gone after removal")
	}
}

func TestRemoveSudo_NotPresent(t *testing.T) {
	s := newSettings()
	if s.RemoveSudo("999") {
		t.Fatal("expected false when phone not in sudoers")
	}
}

// Prefixes ────────────────────────────────────────────────────────────────────

func TestGetPrefixes_Default(t *testing.T) {
	s := newSettings()
	p := s.GetPrefixes()
	if len(p) != 1 || p[0] != "." {
		t.Fatalf("unexpected default prefixes: %v", p)
	}
}

func TestSetPrefixes_Multiple(t *testing.T) {
	s := newSettings()
	s.SetPrefixes(". / #")
	p := s.GetPrefixes()
	if len(p) != 3 {
		t.Fatalf("expected 3 prefixes, got %v", p)
	}
}

func TestSetPrefixes_EmptyToken(t *testing.T) {
	s := newSettings()
	s.SetPrefixes("empty")
	p := s.GetPrefixes()
	if len(p) != 1 || p[0] != "" {
		t.Fatalf("expected one empty prefix, got %v", p)
	}
}

// Mode ────────────────────────────────────────────────────────────────────────

func TestGetMode_Default(t *testing.T) {
	s := newSettings()
	if s.GetMode() != ModePublic {
		t.Fatalf("expected ModePublic, got %q", s.GetMode())
	}
}

func TestSetMode(t *testing.T) {
	s := newSettings()
	s.SetMode(ModePrivate)
	if s.GetMode() != ModePrivate {
		t.Fatal("expected ModePrivate after SetMode")
	}
}

// Language ────────────────────────────────────────────────────────────────────

func TestGetLanguage_Default(t *testing.T) {
	s := newSettings()
	if s.GetLanguage() != "en" {
		t.Fatalf("expected 'en', got %q", s.GetLanguage())
	}
}

func TestGetLanguage_EmptyFallsBackToEn(t *testing.T) {
	s := newSettings()
	s.Language = ""
	if s.GetLanguage() != "en" {
		t.Fatal("expected fallback to 'en' when Language is empty")
	}
}

func TestSetLanguage(t *testing.T) {
	s := newSettings()
	s.SetLanguage("ar")
	if s.GetLanguage() != "ar" {
		t.Fatalf("expected 'ar', got %q", s.GetLanguage())
	}
}
