package regionref

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func lightBytes(values ...float32) []byte {
	b := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b
}

func TestPublishedReferenceIsReadAtItsIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference")
	id := Identity{1, 2, 3}
	if err := Publish(path, id, 1, 1, lightBytes(2, 3, 4)); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path, id, 1, 1, 1024)
	if err != nil || got == nil || got.Pix[0].R != 2 || got.Pix[0].G != 3 || got.Pix[0].B != 4 {
		t.Fatalf("read %+v, %v", got, err)
	}
}

func TestReferenceRefusesWrongIdentityDimensionsAndLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference")
	id := Identity{1}
	if err := Publish(path, id, 1, 1, lightBytes(1, 2, 3)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id            Identity
		width, height int
		limit         int64
	}{
		{Identity{2}, 1, 1, 1024}, {id, 2, 1, 1024}, {id, 1, 1, 10},
	} {
		if got, err := Read(path, tc.id, tc.width, tc.height, tc.limit); err == nil || got != nil {
			t.Fatalf("accepted mismatch: %+v, %v", got, err)
		}
	}
}

func TestChangedOrTruncatedContentIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference")
	id := Identity{1}
	if err := Publish(path, id, 1, 1, lightBytes(1, 2, 3)); err != nil {
		t.Fatal(err)
	}
	whole, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), whole...)
	changed[len(changed)-1] ^= 1
	for _, bytes := range [][]byte{changed, whole[:len(whole)-1], append(whole, 0)} {
		if err := os.WriteFile(path, bytes, 0600); err != nil {
			t.Fatal(err)
		}
		if got, err := Read(path, id, 1, 1, 1024); err == nil || got != nil {
			t.Fatalf("accepted changed reference: %+v, %v", got, err)
		}
	}
}

func TestPublishNeverReplacesAndRefusesBadLight(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reference")
	id := Identity{1}
	if err := Publish(path, id, 1, 1, lightBytes(1, 2, 3)); err != nil {
		t.Fatal(err)
	}
	if err := Publish(path, id, 1, 1, lightBytes(9, 9, 9)); err == nil {
		t.Fatal("replaced reference")
	}
	got, err := Read(path, id, 1, 1, 1024)
	if err != nil || got.Pix[0].R != 1 {
		t.Fatalf("original changed: %+v, %v", got, err)
	}
	bad := filepath.Join(dir, "bad")
	if err := Publish(bad, id, 1, 1, lightBytes(float32(math.NaN()), 2, 3)); err == nil {
		t.Fatal("accepted NaN")
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Fatalf("bad reference exists: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary entry left: %v, %v", entries, err)
	}
}
