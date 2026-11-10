package sources

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShelfReportsEveryRealFileAndNetworkFailure(t *testing.T) {
	for _, fault := range []string{"rename", "sidecar", "request", "network", "shelf path", "part path"} {
		t.Run(fault, func(t *testing.T) {
			server, _ := served(t)
			source := one(t, server, "")
			shelf := NewShelf(t.TempDir(), server.Client())
			expected := ""
			switch fault {
			case "rename":
				expected = shelf.Path(source)
				if err := os.Mkdir(expected, 0700); err != nil {
					t.Fatal(err)
				}
			case "sidecar":
				expected = shelf.Path(source) + ".source"
				if err := os.Mkdir(expected, 0700); err != nil {
					t.Fatal(err)
				}
			case "request":
				source.row.From = "http://bad\x00url"
				expected = "invalid control character"
			case "network":
				server.Close()
				expected = "connection refused"
			case "shelf path":
				shelf.dir = filepath.Join(t.TempDir(), "ordinary-file")
				if err := os.WriteFile(shelf.dir, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
				expected = shelf.dir
			case "part path":
				expected = shelf.Path(source) + ".part"
				if err := os.Mkdir(expected, 0700); err != nil {
					t.Fatal(err)
				}
			}
			_, err := shelf.Fetch(context.Background(), source, time.Now())
			if err == nil || !strings.Contains(err.Error(), expected) {
				t.Fatalf("%s: %v, want %q", fault, err, expected)
			}
			if fault == "network" {
				var requestError *url.Error
				if !errors.As(err, &requestError) || requestError.Op != "Get" {
					t.Fatalf("network cause: %v", err)
				}
			}
		})
	}
	if _, err := hashOf(t.TempDir()); err == nil {
		t.Fatal("directory hashed as a source")
	}
	source := Source{row: row{Key: "withdrawn", Never: "removed from the catalogue"}}
	said, err := NewShelf(t.TempDir(), nil).Check(source)
	if err != nil || said != "withdrawn: never fetched" {
		t.Fatalf("withdrawn check: %q %v", said, err)
	}
}
