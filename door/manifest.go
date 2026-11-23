package door

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ledger is a frames folder's manifest: the key of the frames, from the
// job's picture and the build that drew them, the sha256 of each frame
// written, by file name, and the movie made from them. It is written
// beside itself and renamed into place when the folder is made and after
// every frame, so a run stopped at any moment leaves a folder the next
// run resumes: a frame written and not yet recorded is drawn again.
//
// Rewriting it whole costs 3.6 ms at 6,000 frames on this machine
// (2026-11-12), against seconds a frame, so it is written every frame.
type ledger struct {
	folder string
	record manifest
}

// manifest is the ledger as manifest.json holds it: the frames' key,
// each frame written by name and sum, the movie working frames are for,
// and the movie made from them.
type manifest struct {
	Key     string            `json:"key"`
	Frames  map[string]string `json:"frames"`
	Planned string            `json:"planned,omitempty"`
	Movie   *filmed           `json:"movie,omitempty"`
}

// filmed is the movie made from a folder's frames: its file, the key of
// the job and build it was made for, sound and all, and the sha256 of
// its bytes.
type filmed struct {
	Path string `json:"path"`
	Key  string `json:"key"`
	Sum  string `json:"sum"`
}

// openLedger is the ledger of a frames folder for frames of key: the
// folder's own where it has one of that key, or a new one, written at
// once, and whether the folder held stale frames, of another key, which
// were removed to draw again. A manifest that does not read is refused,
// and so is a folder holding pictures the manifest does not list, since
// they are another run's; nothing is removed then.
func openLedger(folder, key string) (*ledger, bool, error) {
	found, exists, err := readManifest(folder)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		book, err := newLedger(folder, key)
		return book, false, err
	}
	if found.Key == key {
		return &ledger{folder: folder, record: found}, false, nil
	}
	stale := &ledger{folder: folder, record: found}
	if err := stale.discard(); err != nil {
		return nil, false, err
	}
	book, err := newLedger(folder, key)
	return book, true, err
}

// peekLedger is what openLedger would find in a frames folder, read
// and nothing changed: whether its frames are stale, of another key, or
// the refusal openLedger would make. A folder not there is fine.
func peekLedger(folder, key string) (bool, error) {
	found, exists, err := readManifest(folder)
	if err != nil || !exists {
		return false, errors.Join(err, emptyOf(folder))
	}
	if found.Key == key {
		return false, nil
	}
	stale := &ledger{folder: folder, record: found}
	return true, stale.unlisted()
}

// readManifest is a frames folder's manifest, and whether it has one. A
// manifest that does not read is refused.
func readManifest(folder string) (manifest, bool, error) {
	raw, err := os.ReadFile(filepath.Join(folder, manifestName))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest{}, false, nil
	}
	if err != nil {
		return manifest{}, false, err
	}
	var found manifest
	if err := json.Unmarshal(raw, &found); err != nil {
		return manifest{}, false, fmt.Errorf("%s: its %s cannot be read "+
			"(%w), so its frames cannot be trusted; empty the folder, or "+
			"draw to a new one", folder, manifestName, err)
	}
	if found.Frames == nil {
		found.Frames = map[string]string{}
	}
	return found, true, nil
}

// discard removes the frames the ledger lists and the ledger itself,
// and only those: a name that is not a frame's is left, and a folder
// holding a picture the ledger does not list is refused by that
// picture's name before anything is removed.
func (book *ledger) discard() error {
	if err := book.unlisted(); err != nil {
		return err
	}
	return book.removeAll()
}

// unlisted refuses a folder holding a picture the ledger does not list,
// by that picture's name: it is not this door's.
func (book *ledger) unlisted() error {
	pictures, err := filepath.Glob(filepath.Join(book.folder,
		"*"+pictureExtension))
	if err != nil {
		return err
	}
	for _, picture := range pictures {
		if _, listed := book.record.Frames[filepath.Base(picture)]; !listed {
			return fmt.Errorf("%s holds %s, which its %s does not list, "+
				"so it is not this door's; empty the folder or draw to a "+
				"new one", book.folder, filepath.Base(picture),
				manifestName)
		}
	}
	return nil
}

// removeAll removes every frame the ledger lists whose name is a
// frame's, then the ledger: nothing it cannot show it wrote.
func (book *ledger) removeAll() error {
	var failed []error
	for name := range book.record.Frames {
		if !frameNamed(name) {
			continue
		}
		err := os.Remove(filepath.Join(book.folder, name))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			failed = append(failed, err)
		}
	}
	manifestPath := filepath.Join(book.folder, manifestName)
	failed = append(failed, os.Remove(manifestPath))
	return errors.Join(failed...)
}

// frameNamed is whether a name is one the door gives a frame, or a
// scene of maps gives a map, and nothing else: no folder, no other file.
func frameNamed(name string) bool {
	for _, pattern := range []string{framePattern, mapPattern} {
		if matched, err := filepath.Match(pattern, name); err == nil &&
			matched {
			return true
		}
	}
	return false
}

// plan records the movie working frames are for, where it is not
// recorded already, so a film resumed on another day finds them.
func (book *ledger) plan(movie string) error {
	if book.record.Planned == movie {
		return nil
	}
	book.record.Planned = movie
	return book.write()
}

// newLedger is a new ledger for a folder that has none, the folder made
// where it is not there and the ledger written in it before any frame.
// A folder holding pictures is refused.
func newLedger(folder, key string) (*ledger, error) {
	if err := emptyOf(folder); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(folder, folderMode); err != nil {
		return nil, err
	}
	book := &ledger{folder: folder, record: manifest{Key: key,
		Frames: map[string]string{}}}
	return book, book.write()
}

// emptyOf refuses a folder with no manifest that holds pictures: they
// are not this job's. A folder not there holds none.
func emptyOf(folder string) error {
	pictures, err := filepath.Glob(filepath.Join(folder,
		"*"+pictureExtension))
	if err != nil {
		return err
	}
	if len(pictures) > nothing {
		return fmt.Errorf("%s holds pictures and no %s, so they are not "+
			"this job's; empty it or draw to a new folder", folder,
			manifestName)
	}
	return nil
}

// kept is whether a frame was written and its file still holds the
// bytes the ledger recorded.
func (book *ledger) kept(name string) bool {
	sum, recorded := book.record.Frames[name]
	return recorded && fileSum(filepath.Join(book.folder, name)) == sum
}

// complete is whether every one of a film's frames is kept.
func (book *ledger) complete(frames int) bool {
	for index := range frames {
		if !book.kept(fmt.Sprintf(frameName, index)) {
			return false
		}
	}
	return true
}

// picture is a kept frame's picture, read back.
func (book *ledger) picture(name string) (*image.RGBA, error) {
	return decoded(filepath.Join(book.folder, name))
}

// decoded is a png file's picture.
func decoded(path string) (*image.RGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	bounds := picture.Bounds()
	made := image.NewRGBA(bounds)
	draw.Draw(made, bounds, picture, bounds.Min, draw.Src)
	return made, nil
}

// note records a frame written and its sum, and writes the ledger.
func (book *ledger) note(name string, sum [sha256.Size]byte) error {
	book.record.Frames[name] = hex.EncodeToString(sum[:])
	return book.write()
}

// screened is whether the movie at path was made from these frames for
// the job and build of key, and still holds the bytes it was made with.
func (book *ledger) screened(path, key string) bool {
	made := book.record.Movie
	return made != nil && made.Path == path && made.Key == key &&
		fileSum(path) == made.Sum
}

// filmed records the movie at path, made for key, and writes the ledger.
func (book *ledger) filmed(path, key string) error {
	book.record.Movie = &filmed{Path: path, Key: key, Sum: fileSum(path)}
	return book.write()
}

// write writes the ledger beside itself and renames it into place.
func (book *ledger) write() error {
	raw, err := json.MarshalIndent(book.record, "", "  ")
	if err != nil {
		return err
	}
	return replaced(filepath.Join(book.folder, manifestName),
		append(raw, '\n'))
}

// fileSum is the sha256 of a file's bytes in hex, or nothing for a file
// that cannot be read, which matches no recorded sum.
func fileSum(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return ""
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// replaced writes a file beside itself and renames it into place, so it
// is never seen half written. Only files this run alone writes are
// written so: never a file another writer appends to.
func replaced(path string, raw []byte) error {
	part := path + partSuffix
	if err := os.WriteFile(part, raw, fileMode); err != nil {
		return err
	}
	return os.Rename(part, path)
}
