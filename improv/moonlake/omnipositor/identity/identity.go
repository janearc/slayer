package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"
)

type Input struct {
	Path   string
	Choice string
}
type Spec struct {
	BinaryPath string
	Settings   []byte
	Inputs     map[string]Input
	FPS        float64
	Frame      int64
	Device     string
	Backend    string
	Width      int
	Height     int
}
type HashedInput struct {
	SHA256 string
	Choice string
}
type PrehashedSpec struct {
	BinarySHA256 string
	Settings     []byte
	Inputs       map[string]HashedInput
	FPS          float64
	Frame        int64
	Device       string
	Backend      string
	Width        int
	Height       int
}
type Key struct {
	digest        string
	frame         int64
	width, height int
}

// String returns the key's digest, 64 hex characters of SHA-256.
func (k Key) String() string { return k.digest }

type namedHash struct {
	Name   string
	SHA256 string
	Choice string
}
type recipe struct {
	Version  int
	Binary   string
	Settings string
	Inputs   []namedHash
	FPS      float64
	Frame    int64
	Device   string
	Backend  string
	Width    int
	Height   int
}

// digest returns the SHA-256 of b as lowercase hex.
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Build returns the Key for spec s. It hashes the binary and each input file
// by content and the settings by value, in input name order, so the same
// spec gives the same key. It refuses invalid text, empty settings, a
// negative frame, an FPS that is not positive and finite, an empty device,
// backend or input name, a size outside 1 to 32768 or over the decoded-bytes
// budget, and any file hashFile refuses.
func Build(s Spec) (Key, error) {
	if !utf8.ValidString(s.BinaryPath) || !utf8.ValidString(s.Device) || !utf8.ValidString(s.Backend) || len(s.Settings) == 0 || s.Frame < 0 || s.FPS <= 0 || math.IsNaN(s.FPS) || math.IsInf(s.FPS, 0) || strings.TrimSpace(s.Device) == "" || strings.TrimSpace(s.Backend) == "" || s.Width <= 0 || s.Height <= 0 || s.Width > 32768 || s.Height > 32768 || int64(s.Width)*int64(s.Height) > maxDecodedBytes/8 {
		return Key{}, errors.New("identity: invalid specification")
	}
	binary, err := hashFile(s.BinaryPath)
	if err != nil {
		return Key{}, fmt.Errorf("identity: binary: %w", err)
	}
	r := recipe{Version: 1, Binary: binary, Settings: digest(s.Settings), FPS: s.FPS, Frame: s.Frame, Device: s.Device, Backend: s.Backend, Width: s.Width, Height: s.Height, Inputs: []namedHash{}}
	names := make([]string, 0, len(s.Inputs))
	for name := range s.Inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !utf8.ValidString(name) || !utf8.ValidString(s.Inputs[name].Path) || !utf8.ValidString(s.Inputs[name].Choice) || strings.TrimSpace(name) == "" {
			return Key{}, errors.New("identity: empty input name")
		}
		input := s.Inputs[name]
		h, err := hashFile(input.Path)
		if err != nil {
			return Key{}, fmt.Errorf("identity: input %q: %w", name, err)
		}
		r.Inputs = append(r.Inputs, namedHash{Name: name, SHA256: h, Choice: input.Choice})
	}
	b, err := json.Marshal(r)
	if err != nil {
		return Key{}, err
	}
	return Key{digest: digest(b), frame: s.Frame, width: s.Width, height: s.Height}, nil
}

// HashFile returns the SHA-256 of the file at path as hex. It refuses what
// hashFile refuses.
func HashFile(path string) (string, error) { return hashFile(path) }

// BuildPrehashed is Build for hashes already known: no file is read. Each
// hash must be 64 lowercase hex characters. It builds the same recipe as
// Build, so a key matches Build's for files with those hashes. It refuses the
// same bad specs as Build.
func BuildPrehashed(s PrehashedSpec) (Key, error) {
	validHash := func(value string) bool {
		if len(value) != 64 {
			return false
		}
		for _, c := range value {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return false
			}
		}
		return true
	}
	if !validHash(s.BinarySHA256) || !utf8.ValidString(s.Device) || !utf8.ValidString(s.Backend) || len(s.Settings) == 0 || s.Frame < 0 || s.FPS <= 0 || math.IsNaN(s.FPS) || math.IsInf(s.FPS, 0) || strings.TrimSpace(s.Device) == "" || strings.TrimSpace(s.Backend) == "" || s.Width <= 0 || s.Height <= 0 || s.Width > 32768 || s.Height > 32768 || int64(s.Width)*int64(s.Height) > maxDecodedBytes/8 {
		return Key{}, errors.New("identity: invalid prehashed specification")
	}
	r := recipe{Version: 1, Binary: s.BinarySHA256, Settings: digest(s.Settings), FPS: s.FPS, Frame: s.Frame, Device: s.Device, Backend: s.Backend, Width: s.Width, Height: s.Height, Inputs: []namedHash{}}
	names := make([]string, 0, len(s.Inputs))
	for name := range s.Inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		input := s.Inputs[name]
		if !utf8.ValidString(name) || strings.TrimSpace(name) == "" || !utf8.ValidString(input.Choice) || !validHash(input.SHA256) {
			return Key{}, errors.New("identity: invalid prehashed input")
		}
		r.Inputs = append(r.Inputs, namedHash{Name: name, SHA256: input.SHA256, Choice: input.Choice})
	}
	b, err := json.Marshal(r)
	if err != nil {
		return Key{}, err
	}
	return Key{digest: digest(b), frame: s.Frame, width: s.Width, height: s.Height}, nil
}

// safePath returns the absolute form of path with the symlinks in its parent
// directory resolved. It refuses an empty path, a path that cannot be
// examined, and one whose last element is a symlink.
func safePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("empty path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("invalid path")
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", errors.New("path unavailable")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("symlink path rejected")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", errors.New("parent unavailable")
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

// hashFile returns the SHA-256 of the regular file at path as hex. It opens
// without following symlinks or blocking, refuses anything not a regular
// file, and refuses a file that was replaced, resized or touched while it was
// being read.
func hashFile(path string) (string, error) {
	absolute, err := safePath(path)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(absolute, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", errors.New("cannot open regular input")
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return "", errors.New("input is not regular")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, before.Size()+1))
	if err != nil {
		return "", errors.New("cannot hash input")
	}
	after, err := f.Stat()
	if err != nil {
		return "", errors.New("cannot stat input")
	}
	atPath, err := os.Lstat(absolute)
	if err != nil || !os.SameFile(before, atPath) || !os.SameFile(before, after) || n != before.Size() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return "", errors.New("input changed while hashing")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
