package gpu

// Two more ways to keep trees, beside the one bbolt file, so the three
// can be measured against each other on the same scenes before one is
// chosen (2026-11-12).
//
// SLAYER_TREE_STORE picks the form, not the place: every form lives in
// the cache home (SLAYER_CACHE), the one file beside the trees kind's
// folder.
//
// Unset or "bolt" is the one file as it has always been; "files" keeps
// each tree as a file of its own in diskcache; "pids": every run reads
// one main bolt file read-only, writes what it grows to a bolt file of its
// own named by its process, and at its end merges into a new main file
// renamed into place, so no reader ever waits for a writer.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/treebolt"
)

// storeVariable names the way trees are kept.
const storeVariable = "SLAYER_TREE_STORE"

// The ways trees are kept, as storeVariable says them.
const (
	inOneFile  = "bolt"
	asFiles    = "files"
	perProcess = "pids"
)

// treeKind keeps trees as files. Its cap is picked by the disk caches
// slice's author, 2026-11-12: moonlake's trees are about 588 MB, so 8
// GiB holds a dozen scenes' trees for one program.
var treeKind = diskcache.MustKind(treebolt.KindName, treeCap,
	"moonlake's trees are about 588 MB; this holds a dozen scenes'")

// treeCap is the trees' cap in bytes, the bolt file's as well; see
// treeKind and treebolt.Cap.
const treeCap = treebolt.Cap

// The names of a per-process file and the merge's, beside the main one.
const (
	ownSuffix    = ".pid-"
	mergeSuffix  = ".merging"
	mergeLock    = ".merge-lock"
	staleMerging = 10 * time.Minute // picked: a merge takes seconds
)

// formOf is the way trees are kept that the environment says, or a
// refusal naming the ways there are.
func formOf(said string) (string, error) {
	switch said {
	case "", inOneFile:
		return inOneFile, nil
	case asFiles, perProcess:
		return said, nil
	}
	return "", fmt.Errorf("tree cache: %s is %q; it is %s, %s or %s",
		storeVariable, said, inOneFile, asFiles, perProcess)
}

// lookAs is a look by the shelf's way of keeping.
func (kept *treeShelf) lookAs(key programKey) ([]byte, error) {
	switch kept.form {
	case asFiles:
		return kept.lookFile(key)
	case perProcess:
		return kept.lookOwnThenMain(key)
	}
	return kept.look(key)
}

// storeAs is a store by the shelf's way of keeping.
func (kept *treeShelf) storeAs(key programKey, value []byte) error {
	switch kept.form {
	case asFiles:
		return kept.storeFile(key, value)
	case perProcess:
		return storeIn(kept.own(), key, value, kept.wait)
	}
	return kept.store(key, value)
}

// fileKey is a tree's key as diskcache keys it.
func fileKey(key programKey) diskcache.Key {
	return diskcache.NewKey(treeKind).Bytes("tree", key[:])
}

// treeFiles is the trees' diskcache store, under the shelf's folder.
func (kept *treeShelf) treeFiles() (*diskcache.Store, error) {
	return diskcache.Open(filepath.Dir(kept.path), treeKind)
}

// lookFile is a tree from its own file, or nil.
func (kept *treeShelf) lookFile(key programKey) ([]byte, error) {
	files, err := kept.treeFiles()
	if err != nil {
		return nil, err
	}
	hit, err := files.Get(fileKey(key))
	if err != nil || !hit.Found() {
		return nil, err
	}
	return hit.Payload(), nil
}

// storeFile keeps a tree as a file of its own.
func (kept *treeShelf) storeFile(key programKey, value []byte) error {
	files, err := kept.treeFiles()
	if err != nil {
		return err
	}
	return files.Put(fileKey(key), value)
}

// own is this process's own bolt file, beside the main one.
func (kept *treeShelf) own() string {
	return kept.path + ownSuffix + strconv.Itoa(os.Getpid())
}

// lookOwnThenMain is a tree from this process's own file, or else from
// the main file, opened read-only.
func (kept *treeShelf) lookOwnThenMain(key programKey) ([]byte, error) {
	for _, path := range []string{kept.own(), kept.path} {
		value, used, err := lookIn(path, key, kept.wait)
		if value != nil && path == kept.path {
			kept.want(key, used)
		}
		if value != nil || err != nil {
			return value, err
		}
	}
	return nil, nil
}

// lookIn is the value under a key in a bolt file, read-only, and when
// it was last used, or nil where the file or the key is not there.
func lookIn(path string, key programKey, wait time.Duration) ([]byte,
	time.Time, error) {
	var used time.Time
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, used, nil
	}
	file, err := treebolt.Open(path, wait, true)
	if err != nil {
		return nil, used, err
	}
	var value []byte
	err = file.View(func(reading *bolt.Tx) error {
		value, used = treebolt.Get(reading, key[:])
		return nil
	})
	return value, used, errors.Join(err, file.Close())
}

// storeIn keeps a value under a key in a bolt file, used now.
func storeIn(path string, key programKey, value []byte,
	wait time.Duration) error {
	file, err := bolt.Open(path, fileMode, &bolt.Options{Timeout: wait})
	if err != nil {
		return err
	}
	err = file.Update(func(writing *bolt.Tx) error {
		return treebolt.Put(writing, key[:], value, time.Now())
	})
	return errors.Join(err, file.Close())
}

// MergeTrees, for the per-process way, folds every finished run's own
// file into a new main file and renames it over the old one, unless
// another merge is in hand; it says what it merged. It does nothing for
// the other ways.
func MergeTrees() (string, error) {
	kept := shelf.Load()
	if kept == nil || kept.form != perProcess {
		return "", nil
	}
	return mergeInto(kept.path, os.Getpid(), kept.wait, nil)
}

// mergeInto is MergeTrees for a main file, the merging process, and a
// step that runs before the rename, which a test makes fail.
func mergeInto(main string, self int, wait time.Duration,
	beforeRename func() error) (string, error) {
	lock, taken, err := mergeTaken(main)
	if err != nil || !taken {
		return "another merge is in hand", err
	}
	defer os.Remove(lock)
	merged, err := finished(main, self)
	if err != nil || len(merged) == 0 {
		return "nothing to merge", err
	}
	next := main + mergeSuffix
	if err := mergedInto(append([]string{main}, merged...), next,
		wait); err != nil {
		return "", errors.Join(err, os.Remove(next))
	}
	if beforeRename != nil {
		if err := beforeRename(); err != nil {
			return "", errors.Join(err, os.Remove(next))
		}
	}
	if err := os.Rename(next, main); err != nil {
		return "", errors.Join(err, os.Remove(next))
	}
	for _, each := range merged {
		_ = os.Remove(each)
	}
	return fmt.Sprintf("merged %d runs' trees", len(merged)), nil
}

// mergeTaken takes the merge's lock, a file made only where it is not
// there; a lock older than staleMerging is a killed merge's, and taken.
func mergeTaken(main string) (string, bool, error) {
	lock := main + mergeLock
	if info, err := os.Stat(lock); err == nil &&
		time.Since(info.ModTime()) > staleMerging {
		_ = os.Remove(lock)
	}
	file, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		fileMode)
	if errors.Is(err, os.ErrExist) {
		return lock, false, nil
	}
	if err != nil {
		return lock, false, err
	}
	return lock, true, file.Close()
}

// finished is every own file beside the main one whose process has
// ended, or is the one merging.
func finished(main string, self int) ([]string, error) {
	found, err := filepath.Glob(main + ownSuffix + "*")
	var done []string
	for _, each := range found {
		pid, bad := strconv.Atoi(strings.TrimPrefix(each,
			main+ownSuffix))
		if bad == nil && (pid == self || !alive(pid)) {
			done = append(done, each)
		}
	}
	return done, err
}

// alive is whether a process of that number is running.
func alive(pid int) bool {
	process, err := os.FindProcess(pid)
	return err == nil && process.Signal(syscall.Signal(0)) == nil
}

// mergedInto writes every tree of the files, read-only and in order, into a
// new bolt file, compact, made fresh.
func mergedInto(from []string, into string, wait time.Duration) error {
	_ = os.Remove(into)
	made, err := bolt.Open(into, fileMode, &bolt.Options{Timeout: wait})
	if err != nil {
		return err
	}
	for _, each := range from {
		if err = copyTrees(each, made, wait); err != nil {
			break
		}
	}
	return errors.Join(err, made.Close())
}

// copyTrees puts every tree of a file into an open one; a file not
// there gives nothing.
func copyTrees(from string, into *bolt.DB, wait time.Duration) error {
	if _, err := os.Stat(from); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	source, err := bolt.Open(from, fileMode,
		&bolt.Options{Timeout: wait, ReadOnly: true})
	if err != nil {
		return err
	}
	err = source.View(func(reading *bolt.Tx) error {
		return into.Update(func(writing *bolt.Tx) error {
			return treebolt.Absorb(writing, reading)
		})
	})
	return errors.Join(err, source.Close())
}
