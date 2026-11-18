package diskcache

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Usage is what a kind's folder holds: its entries, their bytes, and
// when the least and most recently used of them was last used.
type Usage struct {
	kind           Kind
	folder         string
	entries        int
	bytes          int64
	oldest, newest time.Time
}

// Kind is the kind the usage is of.
func (usage Usage) Kind() Kind { return usage.kind }

// Folder is the kind's folder.
func (usage Usage) Folder() string { return usage.folder }

// Entries is how many entries the kind holds.
func (usage Usage) Entries() int { return usage.entries }

// Bytes is how many bytes they are.
func (usage Usage) Bytes() int64 { return usage.bytes }

// Oldest is when the least recently used entry was last used; the zero
// time when there are none.
func (usage Usage) Oldest() time.Time { return usage.oldest }

// Newest is when the most recently used entry was last used.
func (usage Usage) Newest() time.Time { return usage.newest }

// held is an entry on disk: where, how big, when last used.
type held struct {
	path string
	size int64
	used time.Time
}

// listed is every entry the store wrote, least recently used first, and
// every .part file of one, each apart. An entry is a regular file at
// folder/FAN/NAME, NAME a key's name and FAN its first fanDigits; a part is
// such a name, a dot and more, ending in .part.
//
// Nothing else is listed: a file at the folder's root, a file of another
// name, a key's name in another fan folder, a file below a fan folder and a
// symbolic link are strangers, which the store never counts or removes.
func (store *Store) listed() ([]held, []held, error) {
	var entries, parts []held
	fans, err := os.ReadDir(store.folder)
	if err != nil {
		return nil, nil, ignoreGone(err)
	}
	for _, fan := range fans {
		if !fan.IsDir() || len(fan.Name()) != fanDigits {
			continue
		}
		more, extra, err := fanned(store.folder, fan.Name())
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, more...)
		parts = append(parts, extra...)
	}
	slices.SortFunc(entries, func(one, other held) int {
		return one.used.Compare(other.used)
	})
	return entries, parts, nil
}

// fanned is the entries and the parts in one fan folder of a store's
// folder, as listed says.
func fanned(folder, fan string) ([]held, []held, error) {
	var entries, parts []held
	files, err := os.ReadDir(filepath.Join(folder, fan))
	if err != nil {
		return nil, nil, ignoreGone(err)
	}
	for _, each := range files {
		name := each.Name()
		if !each.Type().IsRegular() || !strings.HasPrefix(name, fan) {
			continue
		}
		info, err := each.Info()
		if err != nil {
			if ignoreGone(err) != nil {
				return nil, nil, err
			}
			continue
		}
		found := held{path: filepath.Join(folder, fan, name),
			size: info.Size(), used: info.ModTime()}
		if isName(name) {
			entries = append(entries, found)
		} else if isPartName(name) {
			parts = append(parts, found)
		}
	}
	return entries, parts, nil
}

// isName is whether a file name is a key's: a sha256 in lower-case hex.
func isName(name string) bool {
	return len(name) == digestLength && strings.Trim(name, hexLetters) == ""
}

// isPartName is whether a file name is a writer's .part of an entry: a
// key's name, a dot, anything, and the suffix.
func isPartName(name string) bool {
	return len(name) >= digestLength+len(partSuffix) &&
		isName(name[:digestLength]) && name[digestLength] == '.' &&
		strings.HasSuffix(name, partSuffix)
}

// ignoreGone is nil for a file another run removed meanwhile.
func ignoreGone(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// Usage is what the store holds now.
func (store *Store) Usage() (Usage, error) {
	entries, _, err := store.listed()
	made := Usage{kind: store.kind, folder: store.folder,
		entries: len(entries)}
	for _, each := range entries {
		made.bytes += each.size
	}
	if len(entries) > nothing {
		made.oldest = entries[0].used
		made.newest = entries[len(entries)-1].used
	}
	return made, err
}

// trimmed removes .part files a killed writer left, then the least
// recently used entries while the kind is over its cap, never one used
// within heldFor.
func (store *Store) trimmed() error {
	entries, parts, err := store.listed()
	if err != nil {
		return err
	}
	now := store.now()
	for _, part := range parts {
		if now.Sub(part.used) > staleParts {
			_ = os.Remove(part.path)
		}
	}
	var total int64
	for _, each := range entries {
		total += each.size
	}
	for _, each := range entries {
		if total <= store.kind.cap || now.Sub(each.used) < heldFor {
			break
		}
		if err := os.Remove(each.path); ignoreGone(err) != nil {
			return err
		}
		total -= each.size
	}
	return nil
}

// Clear removes every entry the kind holds and says how many; a .part
// file being written is left to its writer.
func (store *Store) Clear() (int, error) {
	entries, _, err := store.listed()
	if err != nil {
		return nothing, err
	}
	removed := nothing
	for _, each := range entries {
		if err := os.Remove(each.path); ignoreGone(err) != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
