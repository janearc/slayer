package identity

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestInvalidUTF8(t *testing.T) {
	s := fixture(t)
	for _, change := range []func(*Spec){func(s *Spec) { s.Device = "\xff" }, func(s *Spec) { s.Backend = "\xff" }, func(s *Spec) { s.BinaryPath += "\xff" }, func(s *Spec) { s.Inputs = map[string]Input{"\xff": {Path: s.BinaryPath}} }, func(s *Spec) { s.Inputs = map[string]Input{"input": {Path: s.BinaryPath, Choice: "\xff"}} }, func(s *Spec) { s.Inputs = map[string]Input{"input": {Path: "\xff"}} }} {
		copy := s
		change(&copy)
		if _, err := Build(copy); err == nil {
			t.Fatal("accepted invalid UTF-8")
		}
	}
}

func chunk(kind string, data []byte) []byte {
	out := make([]byte, len(data)+12)
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	copy(out[4:], kind)
	copy(out[8:], data)
	binary.BigEndian.PutUint32(out[len(out)-4:], crc32.ChecksumIEEE(out[4:len(out)-4]))
	return out
}

func TestStrictPNGStream(t *testing.T) {
	data := tiny(t, 1)
	s := fixture(t)
	k := build(t, s)
	var ihdr, idat []byte
	for rest := data[8:]; len(rest) > 0; {
		n := int(binary.BigEndian.Uint32(rest))
		switch string(rest[4:8]) {
		case "IHDR":
			ihdr = bytes.Clone(rest[:n+12])
		case "IDAT":
			idat = append(idat, rest[8:n+8]...)
		}
		rest = rest[n+12:]
	}
	wrap := func(chunks ...[]byte) []byte {
		out := bytes.Clone(data[:8])
		out = append(out, ihdr...)
		for _, c := range chunks {
			out = append(out, c...)
		}
		return append(out, chunk("IEND", nil)...)
	}
	cases := [][]byte{
		wrap(chunk("ABCD", nil), chunk("IDAT", idat)),
		wrap(chunk("PLTE", []byte{0, 0, 0}), chunk("IDAT", idat)),
		wrap(chunk("tRNS", []byte{0, 0}), chunk("IDAT", idat)),
		wrap(chunk("tEXt", []byte("x\x00y")), chunk("IDAT", idat)),
		wrap(chunk("IDAT", append(bytes.Clone(idat), 1, 2, 3))),
		wrap(chunk("IDAT", append(bytes.Clone(idat), idat...))),
		wrap(chunk("IDAT", idat[:len(idat)/2]), chunk("tEXt", []byte("x\x00y")), chunk("IDAT", idat[len(idat)/2:])),
		wrap(chunk("IDAT", idat), chunk("IHDR", ihdr[8:len(ihdr)-4])),
	}
	for i, b := range cases {
		if err := validatePNG(b, k); err == nil {
			t.Fatalf("accepted malformed PNG %d", i)
		}
	}
}

func TestDecodedBudgetAnd16Bit(t *testing.T) {
	s := fixture(t)
	s.Width = 4096
	s.Height = 2160
	k := build(t, s)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA64(image.Rect(0, 0, s.Width, s.Height))); err != nil {
		t.Fatal(err)
	}
	if err := validatePNG(encoded.Bytes(), k); err != nil {
		t.Fatalf("4K 16-bit rejected: %v", err)
	}
	s.Width = 32768
	s.Height = 8192
	if _, err := Build(s); err == nil {
		t.Fatal("unbounded decoded allocation accepted")
	}
}

func TestCancellationWaitingForSameStore(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	k := build(t, fixture(t))
	data := tiny(t, 1)
	store.mu.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := store.PublishContext(ctx, k, data, false); done <- err }()
	select {
	case err := <-done:
		store.mu.Unlock()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		store.mu.Unlock()
		t.Fatal("in-process lock ignored context")
	}
	if _, err := store.Verify(k); !errors.Is(err, ErrMissing) {
		t.Fatalf("cancelled publication exists: %v", err)
	}
}

func TestPublicationCancellation(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	k := build(t, fixture(t))
	data := tiny(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.PublishContext(ctx, k, data, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled publication: %v", err)
	}
	if _, err := store.Verify(k); !errors.Is(err, ErrMissing) {
		t.Fatalf("cancelled publication visible: %v", err)
	}
	lock, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := store.PublishContext(ctx, k, data, false); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("lock wait ignored context")
	}
	if _, err := os.Lstat(filepath.Join(dir, slot(k))); !os.IsNotExist(err) {
		t.Fatalf("cancelled slot exists: %v", err)
	}
}

func TestAnchoredDirectoryAfterParentReplacement(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "parent")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "output")
	if err := os.Mkdir(output, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	moved := filepath.Join(parent, "moved")
	if err := os.Rename(dir, moved); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(outside, "output"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, dir); err != nil {
		t.Fatal(err)
	}
	k := build(t, fixture(t))
	if _, err := store.Publish(k, tiny(t, 1), false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(k); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(outside, "output", slot(k))); !os.IsNotExist(err) {
		t.Fatalf("escaped anchored directory: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(moved, "output", slot(k))); err != nil {
		t.Fatal(err)
	}
	readFlat(t, filepath.Join(moved, "output"), k, tiny(t, 1))
	if _, err := store.Publish(k, tiny(t, 2), true); err != nil {
		t.Fatal(err)
	}
	readFlat(t, filepath.Join(moved, "output"), k, tiny(t, 2))
	if _, err := os.Lstat(filepath.Join(outside, "output", slot(k))); !os.IsNotExist(err) {
		t.Fatalf("redraw escaped anchored directory: %v", err)
	}
}
