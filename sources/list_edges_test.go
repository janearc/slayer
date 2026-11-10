package sources

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestListFindsOnlyItsNamedSource(t *testing.T) {
	list, err := listed(t, "sources:\n  - {key: gone, never: withdrawn}\n")
	if err != nil {
		t.Fatal(err)
	}
	got, found := list.Find("gone")
	if !found || got.Key() != "gone" || got.Never() != "withdrawn" {
		t.Fatalf("named source: %+v %v", got, found)
	}
	if got, found := list.Find("other"); found || got.Key() != "" {
		t.Fatalf("absent source: %+v %v", got, found)
	}
	_, err = Read(filepath.Join(t.TempDir(), "missing.yaml"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing list cause: %v", err)
	}
}
