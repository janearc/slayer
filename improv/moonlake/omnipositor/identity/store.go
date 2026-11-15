package identity

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image/png"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var ErrMissing = errors.New("identity: frame missing")
var ErrConflict = errors.New("identity: frame identity conflict")
var ErrCorrupt = errors.New("identity: corrupt frame")

const maxPNGBytes = 128 << 20
const maxDecodedBytes = 128 << 20

type Store struct {
	mu     sync.Mutex
	root   *os.Root
	dir    *os.File
	closed bool
}
type manifest struct {
	Version int
	Key     string
	Frame   int64
	Width   int
	Height  int
	Bytes   int
	SHA256  string
}

// Open opens the existing directory at path as a Store. It refuses a path
// that is a symlink, is missing, or is not the same directory when opened
// again, which would mean it was swapped during the open.
func Open(path string) (*Store, error) {
	absolute, err := safePath(path)
	if err != nil {
		return nil, err
	}
	dir, err := os.OpenFile(absolute, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("identity: cannot open output directory")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		dir.Close()
		return nil, errors.New("identity: cannot anchor output directory")
	}
	a, e1 := dir.Stat()
	b, e2 := root.Stat(".")
	if e1 != nil || e2 != nil || !os.SameFile(a, b) {
		root.Close()
		dir.Close()
		return nil, errors.New("identity: output directory changed")
	}
	return &Store{root: root, dir: dir}, nil
}

// Close lets go of the store's root and directory handles. Closing again does
// nothing and returns nil.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return errors.Join(s.root.Close(), s.dir.Close())
}

// lock takes the store's lock with no deadline; see lockContext.
func (s *Store) lock() error { return s.lockContext(context.Background()) }

// pause waits 5 milliseconds, or less if ctx ends first, in which case it
// returns the error of ctx.
func pause(ctx context.Context) error {
	timer := time.NewTimer(5 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// lockContext takes the in-process mutex and then an exclusive flock on the
// directory, which keeps other processes out too. It polls every 5
// milliseconds, stops with the error of ctx if ctx ends, and refuses a closed
// store. On success the caller must call unlock.
func (s *Store) lockContext(ctx context.Context) error {
	for !s.mu.TryLock() {
		if err := pause(ctx); err != nil {
			return err
		}
	}
	if s.closed {
		s.mu.Unlock()
		return errors.New("identity: store closed")
	}
	for {
		if err := ctx.Err(); err != nil {
			s.mu.Unlock()
			return err
		}
		err := syscall.Flock(int(s.dir.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != syscall.EINTR && err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			s.mu.Unlock()
			return err
		}
		if err := pause(ctx); err != nil {
			s.mu.Unlock()
			return err
		}
	}
}

// unlock drops the flock on the directory and then the mutex.
func (s *Store) unlock() { syscall.Flock(int(s.dir.Fd()), syscall.LOCK_UN); s.mu.Unlock() }

// slot returns the file name for the frame of k: the frame number padded to
// five digits, then .png.
func slot(k Key) string { return fmt.Sprintf("%05d.png", k.frame) }

// valid reports whether k has a 64 character digest, a positive size and a
// frame of zero or more; a zero Key is not valid.
func valid(k Key) bool { return len(k.digest) == 64 && k.width > 0 && k.height > 0 && k.frame >= 0 }

// Verify is VerifyContext with a context that never ends.
func (s *Store) Verify(k Key) ([]byte, error) { return s.VerifyContext(context.Background(), k) }

// VerifyContext returns the PNG stored for k once it has checked it, and syncs
// the directory. It refuses an invalid key, returns ErrMissing, ErrConflict or
// ErrCorrupt from verify, and returns the error of ctx if ctx ends at any
// point.
func (s *Store) VerifyContext(ctx context.Context, k Key) ([]byte, error) {
	if !valid(k) {
		return nil, errors.New("identity: invalid key")
	}
	if err := s.lockContext(ctx); err != nil {
		return nil, err
	}
	defer s.unlock()
	data, err := s.verify(k)
	if contextErr := ctx.Err(); contextErr != nil {
		return nil, contextErr
	}
	if err != nil {
		return nil, err
	}
	if err := s.dir.Sync(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

// read returns the contents of the file name under the root. It does not
// follow symlinks. It returns ErrCorrupt if the file cannot be opened, is not
// a regular file, or is longer than limit bytes.
func (s *Store) read(name string, limit int64) ([]byte, error) {
	f, err := s.root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrCorrupt
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, ErrCorrupt
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, ErrCorrupt
	}
	return b, nil
}

// owned returns the name of the generation directory that the slot of k links
// to. The slot must be a symlink to a .generation-ID/frame.png with a 32 hex
// character ID, and a .link file beside that generation must be the same
// file. It returns ErrMissing if the slot is absent and ErrCorrupt otherwise.
func (s *Store) owned(k Key) (string, error) {
	info, err := s.root.Lstat(slot(k))
	if os.IsNotExist(err) {
		return "", ErrMissing
	}
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return "", ErrCorrupt
	}
	target, err := s.root.Readlink(slot(k))
	if err != nil || !strings.HasSuffix(target, "/frame.png") {
		return "", ErrCorrupt
	}
	generation := strings.TrimSuffix(target, "/frame.png")
	id := strings.TrimPrefix(generation, ".generation-")
	if generation == id || len(id) != 32 || strings.Trim(id, "0123456789abcdef") != "" {
		return "", ErrCorrupt
	}
	witness, err := s.root.Lstat(generation + ".link")
	if err != nil || witness.Mode()&os.ModeSymlink == 0 || !os.SameFile(info, witness) {
		return "", ErrCorrupt
	}
	return generation, nil
}

// verify checks the stored frame of k and returns its PNG bytes. It needs
// ownership, a real generation directory, and a manifest of at most 16384
// bytes that decodes with no unknown fields as version 1. A manifest for
// another key or frame gives ErrConflict. A size, byte count, hash or PNG
// that does not match gives ErrCorrupt. The caller must hold the lock.
func (s *Store) verify(k Key) ([]byte, error) {
	target, err := s.owned(k)
	if err != nil {
		return nil, err
	}
	info, err := s.root.Lstat(target)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrCorrupt
	}
	raw, err := s.read(target+"/manifest.json", 16384)
	if err != nil {
		return nil, err
	}
	var m manifest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&m) != nil || dec.Decode(new(any)) != io.EOF || m.Version != 1 {
		return nil, ErrCorrupt
	}
	if m.Key != k.digest || m.Frame != k.frame {
		return nil, ErrConflict
	}
	if m.Width != k.width || m.Height != k.height {
		return nil, ErrCorrupt
	}
	data, err := s.read(target+"/frame.png", maxPNGBytes)
	if err != nil {
		return nil, err
	}
	if m.Bytes != len(data) || m.SHA256 != digest(data) {
		return nil, ErrCorrupt
	}
	if err := validatePNG(data, k); err != nil {
		return nil, ErrCorrupt
	}
	return data, nil
}

// validatePNG returns nil if data is a PNG of the size of k, else ErrCorrupt.
// It checks the signature, each chunk's CRC, IHDR first, only IHDR, IDAT and
// IEND chunks, IDATs in one run, nothing after IEND, a decompressed stream
// of exactly one valid filter byte and row per line, and then a full decode.
// It also refuses a size of k over 32768 or over the decoded-bytes budget.
func validatePNG(data []byte, k Key) error {
	if k.width <= 0 || k.height <= 0 || k.width > 32768 || k.height > 32768 || int64(k.width)*int64(k.height) > maxDecodedBytes/8 || len(data) > maxPNGBytes || len(data) < 8 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return ErrCorrupt
	}
	rest := data[8:]
	ended := false
	seenHeader := false
	seenIDAT := false
	afterIDAT := false
	var compressed bytes.Buffer
	bits := 0
	for len(rest) >= 12 {
		n := uint64(binary.BigEndian.Uint32(rest[:4]))
		if n > uint64(len(rest)-12) {
			return ErrCorrupt
		}
		end := int(n) + 8
		if crc32.ChecksumIEEE(rest[4:end]) != binary.BigEndian.Uint32(rest[end:end+4]) {
			return ErrCorrupt
		}
		kind := string(rest[4:8])
		payload := rest[8:end]
		rest = rest[end+4:]
		if !seenHeader && kind != "IHDR" {
			return ErrCorrupt
		}
		if seenIDAT && kind != "IDAT" {
			afterIDAT = true
		}
		switch kind {
		case "IHDR":
			if seenHeader || n != 13 || payload[12] != 0 {
				return ErrCorrupt
			}
			seenHeader = true
			channels := 0
			switch payload[9] {
			case 0:
				channels = 1
			case 2:
				channels = 3
			case 4:
				channels = 2
			case 6:
				channels = 4
			default:
				return ErrCorrupt
			}
			bits = int(payload[8]) * channels
		case "IDAT":
			if afterIDAT {
				return ErrCorrupt
			}
			seenIDAT = true
			compressed.Write(payload)
		case "IEND":
			if n != 0 || len(rest) != 0 || !seenIDAT {
				return ErrCorrupt
			}
			ended = true
		default:
			return ErrCorrupt
		}
		if ended {
			break
		}
	}
	if !ended {
		return ErrCorrupt
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width != k.width || cfg.Height != k.height || bits <= 0 || bits > 64 {
		return ErrCorrupt
	}
	source := bytes.NewReader(compressed.Bytes())
	stream, err := zlib.NewReader(source)
	if err != nil {
		return ErrCorrupt
	}
	defer stream.Close()
	rowBytes := (int64(k.width)*int64(bits) + 7) / 8
	for row := 0; row < k.height; row++ {
		var filter [1]byte
		if _, err := io.ReadFull(stream, filter[:]); err != nil || filter[0] > 4 {
			return ErrCorrupt
		}
		if _, err := io.CopyN(io.Discard, stream, rowBytes); err != nil {
			return ErrCorrupt
		}
	}
	var extra [1]byte
	if n, err := stream.Read(extra[:]); n != 0 || err != io.EOF || source.Len() != 0 {
		return ErrCorrupt
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil || im.Bounds().Dx() != k.width || im.Bounds().Dy() != k.height {
		return ErrCorrupt
	}
	return nil
}

// write creates the file name under the root, failing if it exists, with
// mode 0600 and no symlink following. It writes data, syncs and closes, and
// returns all the errors joined.
func (s *Store) write(name string, data []byte) error {
	f, err := s.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	_, e1 := f.Write(data)
	e2 := f.Sync()
	e3 := f.Close()
	return errors.Join(e1, e2, e3)
}

// Publish is PublishContext with a context that never ends.
func (s *Store) Publish(k Key, data []byte, redraw bool) (bool, error) {
	return s.PublishContext(context.Background(), k, data, redraw)
}

// PublishContext stores data as the frame of k and reports whether a good
// frame was already there and left alone, which it does unless redraw is
// true. It refuses an invalid key, data over maxPNGBytes, data that
// validatePNG refuses, a slot with bad ownership, and, without redraw, a
// frame that fails verify. It writes a new generation directory and links
// the slot to it, with a rename when redraw replaces a frame.
func (s *Store) PublishContext(ctx context.Context, k Key, data []byte, redraw bool) (bool, error) {
	if !valid(k) {
		return false, errors.New("identity: invalid key")
	}
	if len(data) > maxPNGBytes {
		return false, ErrCorrupt
	}
	if err := s.lockContext(ctx); err != nil {
		return false, err
	}
	defer s.unlock()
	data = bytes.Clone(data)
	if err := validatePNG(data, k); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, ownershipErr := s.owned(k)
	if ownershipErr != nil && !errors.Is(ownershipErr, ErrMissing) {
		return false, ownershipErr
	}
	_, err := s.verify(k)
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err == nil && !redraw {
		if err := s.dir.Sync(); err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil && !errors.Is(err, ErrMissing) && !redraw {
		return false, err
	}
	random := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return false, err
	}
	name := fmt.Sprintf(".generation-%x", random)
	if err := s.root.Mkdir(name, 0700); err != nil {
		return false, err
	}
	installed := false
	defer func() {
		if !installed {
			s.root.Remove(name + ".link")
			s.root.RemoveAll(name)
		}
	}()
	m := manifest{Version: 1, Key: k.digest, Frame: k.frame, Width: k.width, Height: k.height, Bytes: len(data), SHA256: digest(data)}
	raw, err := json.Marshal(m)
	if err != nil {
		return false, err
	}
	if err := s.write(name+"/frame.png", data); err != nil {
		return false, err
	}
	if err := s.write(name+"/manifest.json", raw); err != nil {
		return false, err
	}
	generation, err := s.root.Open(name)
	if err != nil {
		return false, err
	}
	syncErr := generation.Sync()
	closeErr := generation.Close()
	if err := errors.Join(syncErr, closeErr); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := s.root.Symlink(name+"/frame.png", name+".link"); err != nil {
		return false, err
	}
	if err := s.dir.Sync(); err != nil {
		return false, err
	}
	if redraw && !errors.Is(ownershipErr, ErrMissing) {
		temporary := name + ".pending"
		if err := s.root.Link(name+".link", temporary); err != nil {
			return false, err
		}
		defer s.root.Remove(temporary)
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if _, err := s.owned(k); err != nil {
			return false, err
		}
		if err := s.root.Rename(temporary, slot(k)); err != nil {
			return false, err
		}
	} else {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := s.root.Link(name+".link", slot(k)); err != nil {
			return false, err
		}
	}
	installed = true
	if err := s.dir.Sync(); err != nil {
		return false, err
	}
	return false, nil
}
