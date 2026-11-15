//go:build !(darwin && cgo)

package movie

import (
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// on macOS the bridge must be built: a build without cgo there cannot
// test what the package is for, and fails rather than skips.
func TestTheBridgeIsBuiltOnMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Fatal("this is macOS built without cgo: the AVFoundation " +
			"bridge is not tested; run with CGO_ENABLED=1")
	}
}

// elsewhere a checked request is opened as ErrNeedsMacOS, and leaves
// nothing behind.
func TestElsewhereWritingNeedsMacOS(t *testing.T) {
	request := smallRequest(t)
	_, err := New(context.Background(), mustSettings(t, request))
	if !errors.Is(err, ErrNeedsMacOS) {
		t.Errorf("New: %v", err)
	}
	nothingLeft(t, request.Path)
}

// the anti-test: a bad request is refused for what is wrong with it,
// not for the platform.
func TestElsewhereABadRequestIsRefusedFirst(t *testing.T) {
	request := smallRequest(t)
	request.Width++
	_, err := NewSettings(request)
	if err == nil || errors.Is(err, ErrNeedsMacOS) {
		t.Errorf("an odd width: %v", err)
	}
}

// elsewhere reading a file that is there is ErrNeedsMacOS too.
func TestElsewhereReadingNeedsMacOS(t *testing.T) {
	path := filepath.Join(t.TempDir(), movieName)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Probe(path); !errors.Is(err, ErrNeedsMacOS) {
		t.Errorf("Probe: %v", err)
	}
	if _, err := ReadSound(path); !errors.Is(err, ErrNeedsMacOS) {
		t.Errorf("ReadSound: %v", err)
	}
	if _, err := OpenFilm(path, 0, image.Pt(16, 16)); !errors.Is(err, ErrNeedsMacOS) {
		t.Errorf("OpenFilm: %v", err)
	}
}
