package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"zaelix/store/sqlstore"
)

func newTestContainer(t *testing.T) *sqlstore.Container {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	c := sqlstore.NewWithDB(db, "sqlite", nil)
	if err := c.Upgrade(context.Background()); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	return c
}

func TestContainer_Upgrade(t *testing.T) {
	
	newTestContainer(t)
}

func TestContainer_GetFirstDevice_EmptyDB(t *testing.T) {
	c := newTestContainer(t)
	dev, err := c.GetFirstDevice(context.Background())
	if err != nil {
		t.Fatalf("GetFirstDevice: %v", err)
	}
	if dev == nil {
		t.Fatal("expected a new device, got nil")
	}
	if dev.NoiseKey == nil || dev.IdentityKey == nil {
		t.Error("new device should have noise/identity keys")
	}
}

func TestContainer_NewDevice(t *testing.T) {
	c := newTestContainer(t)
	dev := c.NewDevice()
	if dev == nil {
		t.Fatal("NewDevice returned nil")
	}
	if dev.NoiseKey == nil || dev.IdentityKey == nil || dev.SignedPreKey == nil {
		t.Error("NewDevice missing keys")
	}
	if dev.RegistrationID == 0 {
		t.Error("expected non-zero RegistrationID")
	}
}

func TestContainer_GetAllDevices_Empty(t *testing.T) {
	c := newTestContainer(t)
	devs, err := c.GetAllDevices(context.Background())
	if err != nil {
		t.Fatalf("GetAllDevices: %v", err)
	}
	if len(devs) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devs))
	}
}

func TestContainer_Close(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	c := sqlstore.NewWithDB(db, "sqlite", nil)
	if err := c.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestContainer_DB_NotNil(t *testing.T) {
	c := newTestContainer(t)
	if c.DB() == nil {
		t.Error("DB() should return a non-nil *sql.DB")
	}
}
