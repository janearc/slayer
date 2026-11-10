// Package smokecache caches smoke renders and their comparison results.
//
// The key includes the executable's hash, the contents of declared input files,
// smoke number, dimensions, sampling, and sensor. The executable captures the
// compiled dependencies, Go version, and platform. Checkout commits are
// recorded for inspection but do not affect the key.
//
// One machine-wide cache can therefore serve identical binaries, including
// reproducible builds made with -trimpath. Entries are written in temporary
// directories and renamed into place atomically. Readers use one directory
// handle, and concurrent stores of the same key can both succeed.
package smokecache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// Root is where the cache lives: $XDG_CACHE_HOME/smokecheck where that is
// set, and the user's cache directory otherwise, ~/Library/Caches/smokecheck
// on a Mac.
func Root() (string, error) {
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "smokecheck"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "smokecheck"), nil
}

// Key is everything a smoke's pictures and verdict depend on.
type Key struct {
	Exe      string // the running executable's sha256
	Inputs   string // what the smoke reads from disk, in words; see Inputs
	Smoke    int
	Size     string // the pictures' sizes, in words
	Sampling string // the rays each takes, in words
	Sensor   string // the camera and the film, in words
}

// Text is the key as lines a person can read, which is also what is hashed.
func (k Key) Text() string {
	return fmt.Sprintf("executable %s\ninputs %s\nsmoke %d\nsize %s\n"+
		"sampling %s\nsensor %s\n", k.Exe, k.Inputs, k.Smoke, k.Size,
		k.Sampling, k.Sensor)
}

// Name is the key's directory name in the cache.
func (k Key) Name() string {
	sum := sha256.Sum256([]byte(k.Text()))
	return hex.EncodeToString(sum[:16])
}

// Here is the key's part that is this run's, the executable, or why it
// cannot be known. The smoke's inputs and the picture are the caller's.
func Here() (Key, error) {
	exe, err := executable()
	if err != nil {
		return Key{}, err
	}
	return Key{Exe: exe}, nil
}

// executable is the sha256 of the binary that is running.
func executable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Trimmed is whether the running binary was built with -trimpath. One
// built without it holds its tree's path, so it shares nothing with a
// binary built alike in another tree.
func Trimmed() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, s := range info.Settings {
		if s.Key == "-trimpath" {
			return s.Value == "true"
		}
	}
	return false
}

// Entry is a whole entry held open: its pictures, its verdict and its
// key, all opened through one handle on one directory. What it holds stays
// what it was when opened, whatever another run does to the cache after.
type Entry struct {
	files map[string]*os.File
}

// Text, Real and Verdict are the files an entry holds; Record is what it
// was drawn from, for people, and not part of the key.
const (
	Text    = "text.png"
	Real    = "real.png"
	Verdict = "verdict.json"
	Record  = "record.txt"
	keyFile = "key.txt"
)

// whole is the files an entry must have to be served.
var whole = []string{Text, Real, Verdict, keyFile}

// Bytes is one of the entry's files.
func (e *Entry) Bytes(name string) ([]byte, error) {
	f, ok := e.files[name]
	if !ok {
		return nil, fmt.Errorf("an entry holds no %s", name)
	}
	return io.ReadAll(io.NewSectionReader(f, 0, 1<<62))
}

// Close lets go of the entry's files.
func (e *Entry) Close() error {
	var errs []error
	for _, f := range e.files {
		errs = append(errs, f.Close())
	}
	return errors.Join(errs...)
}

// Hold is the files named in dir, held open, all opened through one
// handle on dir: if dir is moved, or another put where it was, they are
// still its files.
func Hold(dir string, names ...string) (*Entry, error) {
	r, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	e := &Entry{files: map[string]*os.File{}}
	for _, name := range names {
		f, err := r.Open(name)
		if err != nil {
			e.Close()
			return nil, err
		}
		e.files[name] = f
	}
	return e, nil
}

// open is the entry in dir for k, held open, or why it is not whole.
func open(dir string, k Key) (*Entry, error) {
	e, err := Hold(dir, whole...)
	if err != nil {
		return nil, err
	}
	b, err := e.Bytes(keyFile)
	if err == nil && string(b) != k.Text() {
		err = fmt.Errorf("%s holds another key", dir)
	}
	if err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}

// Open is the entry for k under root, held open, when the cache has it
// whole. Anything less, a file gone included, is not served.
func Open(root string, k Key) (*Entry, bool) {
	e, err := open(filepath.Join(root, "v1", k.Name()), k)
	return e, err == nil
}

// Store keeps an entry for k under root, whole or not at all, and is the
// entry, held open. fill writes the pictures and the verdict into the
// directory it is given, which is renamed into place once it has. record
// is kept beside them.
//
// An entry already there and whole is kept, since it is the same key's,
// unless replace is set, which is how a forced redraw mends one. A run
// that loses a race to store a key is not an error: another stored it.
func Store(root string, k Key, record string, replace bool,
	fill func(dir string) error) (*Entry, error) {
	dir := filepath.Join(root, "v1", k.Name())
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp(parent, ".new-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	if err := fill(tmp); err != nil {
		return nil, err
	}
	for name, s := range map[string]string{keyFile: k.Text(),
		Record: record} {
		err := os.WriteFile(filepath.Join(tmp, name), []byte(s), 0o644)
		if err != nil {
			return nil, err
		}
	}
	e, err := open(tmp, k)
	if err != nil {
		return nil, err
	}
	if err := put(root, tmp, dir, k, replace); err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}

// put renames tmp into place as dir, the entry for k.
func put(root, tmp, dir string, k Key, replace bool) error {
	err := os.Rename(tmp, dir)
	if err == nil || !errors.Is(err, fs.ErrExist) {
		return err
	}
	if !replace {
		if there, ok := Open(root, k); ok {
			there.Close()
			return nil
		}
	}
	// move what is there aside, put this one in, and let the old one
	// go. a reader between the two renames finds nothing and renders. a
	// reader that opened the old one keeps it until it closes it.
	aside, err := os.MkdirTemp(filepath.Dir(dir), ".old-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(aside)
	err = os.Rename(dir, filepath.Join(aside, "entry"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// another run may have put its own in since: the same key's, and as
	// new as this one, so it stays
	if err := os.Rename(tmp, dir); err != nil &&
		!errors.Is(err, fs.ErrExist) {
		return err
	}
	return nil
}

// oneLine is the first line of s, for an error.
func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i] + " (and more)"
	}
	return s
}
