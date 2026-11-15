package identity

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestVerifyContextSameStoreContention(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := build(t, fixture(t))
	data := tiny(t, 1)
	if _, err := store.Publish(key, data, false); err != nil {
		t.Fatal(err)
	}
	if err := store.lockContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			store.unlock()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := store.VerifyContext(ctx, key); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("contended verification: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("verification ignored cancellation while same store was locked")
	}
	store.unlock()
	locked = false
	for _, verify := range []func() ([]byte, error){func() ([]byte, error) { return store.VerifyContext(context.Background(), key) }, func() ([]byte, error) { return store.Verify(key) }} {
		got, err := verify()
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("verification after contention: %v", err)
		}
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if _, err := store.VerifyContext(ctx, key); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled verification: %v", err)
	}
}
