package main

import (
	"os"
	"strings"
	"testing"
)

func TestDbConfig_Default(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	dialect, addr := dbConfig()
	if dialect != "sqlite" {
		t.Fatalf("expected 'sqlite', got %q", dialect)
	}
	if !strings.Contains(addr, "database.db") {
		t.Fatalf("expected default db path in addr, got %q", addr)
	}
}

func TestDbConfig_SQLiteExplicit(t *testing.T) {
	os.Setenv("DATABASE_URL", "mybot.db")
	defer os.Unsetenv("DATABASE_URL")
	dialect, addr := dbConfig()
	if dialect != "sqlite" {
		t.Fatalf("expected 'sqlite', got %q", dialect)
	}
	if !strings.Contains(addr, "mybot.db") {
		t.Fatalf("expected 'mybot.db' in addr, got %q", addr)
	}
}

func TestDbConfig_SQLitePragmas(t *testing.T) {
	os.Setenv("DATABASE_URL", "test.db")
	defer os.Unsetenv("DATABASE_URL")
	_, addr := dbConfig()
	for _, pragma := range []string{"foreign_keys", "journal_mode", "busy_timeout"} {
		if !strings.Contains(addr, pragma) {
			t.Errorf("expected pragma %q in sqlite addr, got %q", pragma, addr)
		}
	}
}

func TestDbConfig_FilePrefix(t *testing.T) {
	os.Setenv("DATABASE_URL", "file:data/bot.db")
	defer os.Unsetenv("DATABASE_URL")
	dialect, addr := dbConfig()
	if dialect != "sqlite" {
		t.Fatalf("expected 'sqlite', got %q", dialect)
	}
	// Should not double the file: prefix
	if strings.Count(addr, "file:") != 1 {
		t.Fatalf("expected exactly one 'file:' in addr, got %q", addr)
	}
}

func TestDbConfig_Postgres(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/mydb")
	defer os.Unsetenv("DATABASE_URL")
	dialect, addr := dbConfig()
	if dialect != "postgres" {
		t.Fatalf("expected 'postgres', got %q", dialect)
	}
	if addr != "postgres://user:pass@localhost:5432/mydb" {
		t.Fatalf("addr should be passed through unchanged, got %q", addr)
	}
}

func TestDbConfig_PostgresqlScheme(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
	defer os.Unsetenv("DATABASE_URL")
	dialect, _ := dbConfig()
	if dialect != "postgres" {
		t.Fatalf("expected 'postgres' for postgresql:// scheme, got %q", dialect)
	}
}
