package store_test

import (
	"context"
	"errors"
	"testing"

	"zaelix/store"
)

func TestNoopStore_ReturnsError(t *testing.T) {
	sentinel := errors.New("sentinel")
	n := &store.NoopStore{Error: sentinel}
	ctx := context.Background()

	if err := n.PutIdentity(ctx, "addr", [32]byte{}); !errors.Is(err, sentinel) {
		t.Errorf("PutIdentity: expected sentinel error, got %v", err)
	}
	if _, err := n.IsTrustedIdentity(ctx, "addr", [32]byte{}); !errors.Is(err, sentinel) {
		t.Errorf("IsTrustedIdentity: expected sentinel error, got %v", err)
	}
	if _, err := n.GetSession(ctx, "addr"); !errors.Is(err, sentinel) {
		t.Errorf("GetSession: expected sentinel error, got %v", err)
	}
	if err := n.PutSession(ctx, "addr", []byte("data")); !errors.Is(err, sentinel) {
		t.Errorf("PutSession: expected sentinel error, got %v", err)
	}
	if err := n.DeleteSession(ctx, "addr"); !errors.Is(err, sentinel) {
		t.Errorf("DeleteSession: expected sentinel error, got %v", err)
	}
}

func TestNoopDevice_NotNil(t *testing.T) {
	d := store.NoopDevice
	if d == nil {
		t.Fatal("NoopDevice should not be nil")
	}
	if d.ID == nil {
		t.Error("NoopDevice.ID should not be nil")
	}
	if d.NoiseKey == nil {
		t.Error("NoopDevice.NoiseKey should not be nil")
	}
	if d.IdentityKey == nil {
		t.Error("NoopDevice.IdentityKey should not be nil")
	}
}

func TestNoopStore_ImplementsAllStores(t *testing.T) {
	
	var _ store.AllStores = (*store.NoopStore)(nil)
	var _ store.DeviceContainer = (*store.NoopStore)(nil)
}
