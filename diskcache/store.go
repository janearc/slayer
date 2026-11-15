package diskcache

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Store is one kind's folder under a home.
type Store struct {
	kind   Kind
	folder string
	now    func() time.Time
}

// Hit is what a Get found: an entry's payload and when it was written,
// or no entry and why.
type Hit struct {
	found   bool
	payload []byte
	written time.Time
	why     string
}

// Found is whether the entry was there and checked whole.
func (hit Hit) Found() bool { return hit.found }

// Payload is the entry's bytes; nothing on a miss.
func (hit Hit) Payload() []byte { return hit.payload }

// Written is when the entry was written; the zero time on a miss.
func (hit Hit) Written() time.Time { return hit.written }

// Why is, on a miss, why: no entry, or what did not check.
func (hit Hit) Why() string { return hit.why }

// Open is a kind's store under a home, its folder made where it is not
// there yet. The home must be absolute.
func Open(home string, kind Kind) (*Store, error) {
	if !filepath.IsAbs(home) {
		return nil, fmt.Errorf("diskcache: the home %q is not an absolute "+
			"folder", home)
	}
	if _, err := KindNamed(kind.name); err != nil {
		return nil, err
	}
	return opened(filepath.Join(home, kind.name), kind, time.Now)
}

// opened is a store of a folder, made, read by a clock.
func opened(folder string, kind Kind, now func() time.Time) (*Store,
	error) {
	if err := os.MkdirAll(folder, folderMode); err != nil {
		return nil, fmt.Errorf("diskcache: kind %s's folder: %w",
			kind.name, err)
	}
	return &Store{kind: kind, folder: folder, now: now}, nil
}

// Folder is the store's folder.
func (store *Store) Folder() string { return store.folder }

// path is where a key's entry is.
func (store *Store) path(key Key) string {
	name := key.Name()
	return filepath.Join(store.folder, name[:fanDigits], name)
}

// Get is the key's entry, read whole and checked, or a miss saying why.
// A key of another kind or one with a refusal is an error; anything on
// disk that does not check is only a miss, never served.
func (store *Store) Get(key Key) (Hit, error) {
	if err := store.keyed(key); err != nil {
		return Hit{}, err
	}
	path := store.path(key)
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Hit{why: "no entry"}, nil
	}
	if err != nil {
		return Hit{why: "unreadable: " + err.Error()}, nil
	}
	read, err := decoded(raw, key.Record())
	if err != nil {
		return Hit{why: err.Error()}, nil
	}
	now := store.now()
	_ = os.Chtimes(path, now, now)
	return Hit{found: true, payload: read.payload, written: read.written},
		nil
}

// Put keeps a payload under a key: written whole beside its place and
// renamed into it, then the kind brought under its cap.
func (store *Store) Put(key Key, payload []byte) error {
	if err := store.keyed(key); err != nil {
		return err
	}
	path := store.path(key)
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return fmt.Errorf("diskcache: kind %s: %w", store.kind.name, err)
	}
	entry := encoded(key.Record(), store.now(), payload)
	if err := written(path, entry); err != nil {
		return fmt.Errorf("diskcache: kind %s: %w", store.kind.name, err)
	}
	return store.trimmed()
}

// keyed refuses a key the store cannot take.
func (store *Store) keyed(key Key) error {
	if key.err != nil {
		return key.err
	}
	if key.kind != store.kind {
		return fmt.Errorf("diskcache: a key of kind %s given to kind %s's "+
			"store", key.kind.name, store.kind.name)
	}
	return nil
}

// written is bytes written to a .part file of a name no other writer
// has, beside path, and renamed onto it.
func written(path string, entry []byte) error {
	part, err := partName(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(part, entry, fileMode); err != nil {
		_ = os.Remove(part)
		return err
	}
	if err := os.Rename(part, path); err != nil {
		_ = os.Remove(part)
		return err
	}
	return nil
}

// partName is a .part name beside a path that no other writer picks:
// the process and sixteen random bytes.
func partName(path string) (string, error) {
	var unique [uniqueBytes]byte
	if _, err := rand.Read(unique[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%d-%s%s", path, os.Getpid(),
		hex.EncodeToString(unique[:]), partSuffix), nil
}
