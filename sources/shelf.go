package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

// A Shelf is where fetched sources are kept on this machine: a
// directory outside any repository, and how to reach the sources.
type Shelf struct {
	dir    string
	client *http.Client
}

// NewShelf is a shelf in a directory, fetching through a client.
func NewShelf(dir string, client *http.Client) Shelf {
	return Shelf{dir: dir, client: client}
}

// Placed is the variable that says where a person's shelf is: a
// directory of their own that lasts past a reboot, so a dem of an
// island is fetched once, and not a temporary one (ruled 2026-10-16).
const Placed = "SLAYER_SHELF"

var errUnplaced = errors.New("sources: no shelf; set " + Placed +
	" to a directory of your own that lasts, as $HOME/slayer/sources")

// Here is the shelf's directory from the environment, or a refusal that
// says how to give it. it is never guessed.
func Here() (string, error) {
	dir := os.Getenv(Placed)
	if dir == "" || !filepath.IsAbs(dir) {
		return "", errUnplaced
	}
	return dir, nil
}

// endings a file's name keeps from its url: the kinds of file sources
// come as. a page's ending, as a script's .cfm, says nothing of what it
// sends, so it is not kept.
var endings = map[string]bool{".pdf": true, ".tif": true, ".tiff": true,
	".csv": true, ".txt": true, ".zip": true, ".nc": true, ".json": true,
	".xml": true, ".png": true, ".jpg": true, ".gz": true}

// Path is where a source's bytes are on the shelf: its key, and the
// ending its url's path has, if it is one of the kinds sources come as.
func (shelf Shelf) Path(source Source) string {
	name := source.Key()
	if from, err := url.Parse(source.From()); err == nil {
		if ending := path.Ext(from.Path); endings[ending] {
			name += ending
		}
	}
	return filepath.Join(shelf.dir, name)
}

// side is what the sidecar beside a source's bytes says: where they
// came from, under what licence, their hash and size, and when.
type side struct {
	Key     string `yaml:"key"`
	From    string `yaml:"from"`
	Licence string `yaml:"licence"`
	Sha256  string `yaml:"sha256"`
	Bytes   int64  `yaml:"bytes"`
	Fetched string `yaml:"fetched"`
}

var errFetch = errors.New("sources: not fetched")

// Fetch brings a source onto the shelf, written beside its sidecar,
// and says what it did. bytes already there that match the pin are
// kept. bytes fetched that do not match the pin are refused and removed;
// with no pin they are kept, and the hash is said for pinning.
func (shelf Shelf) Fetch(ctx context.Context, source Source,
	now time.Time) (string, error) {
	if source.Never() != "" {
		return fmt.Sprintf("%s: never fetched: %s", source.Key(),
			source.Never()), nil
	}
	at := shelf.Path(source)
	if source.Pinned() && matches(at, source.row.Sha256) {
		return fmt.Sprintf("%s: kept, %s", source.Key(), at), nil
	}
	sum, size, err := shelf.download(ctx, source, at+".part")
	if err == nil {
		err = held(source, sum, size)
	}
	if err != nil {
		os.Remove(at + ".part")
		return "", fmt.Errorf("%s: %w", source.Key(), err)
	}
	if err := os.Rename(at+".part", at); err != nil {
		return "", err
	}
	if err := shelf.sidecar(source, at, sum, size, now); err != nil {
		return "", err
	}
	if !source.Pinned() {
		return fmt.Sprintf("%s: fetched, not pinned: sha256 %s, %d bytes",
			source.Key(), sum, size), nil
	}
	return fmt.Sprintf("%s: fetched, %d bytes, %s", source.Key(), size,
		at), nil
}

// held is what is wrong with fetched bytes against the list's pin and
// size, or nothing.
func held(source Source, sum string, size int64) error {
	if source.Pinned() && sum != source.row.Sha256 {
		return fmt.Errorf("%w: sha256 %s, the list pins %s", errFetch, sum,
			source.row.Sha256)
	}
	if source.row.Bytes != 0 && size != source.row.Bytes {
		return fmt.Errorf("%w: %d bytes, the list says %d", errFetch, size,
			source.row.Bytes)
	}
	return nil
}

// download writes a source's bytes to a path, hashing them as they come.
func (shelf Shelf) download(ctx context.Context, source Source,
	to string) (string, int64, error) {
	asking, err := http.NewRequestWithContext(ctx, http.MethodGet,
		source.From(), nil)
	if err != nil {
		return "", 0, err
	}
	answer, err := shelf.client.Do(asking)
	if err != nil {
		return "", 0, err
	}
	defer answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("%w: %s answered %s", errFetch,
			source.From(), answer.Status)
	}
	if err := os.MkdirAll(shelf.dir, 0o755); err != nil {
		return "", 0, err
	}
	file, err := os.Create(to)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), answer.Body)
	if closing := file.Close(); err == nil {
		err = closing
	}
	return hex.EncodeToString(hash.Sum(nil)), size, err
}

// sidecar writes what is known of fetched bytes beside them.
func (shelf Shelf) sidecar(source Source, at, sum string, size int64,
	now time.Time) error {
	text, err := yaml.Marshal(side{Key: source.Key(),
		From: source.From(), Licence: source.Licence(), Sha256: sum,
		Bytes: size, Fetched: now.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	return os.WriteFile(at+".source", text, 0o644)
}

// matches is whether a file is there and its sha256 is the one given.
func matches(at, want string) bool {
	sum, err := hashOf(at)
	return err == nil && sum == want
}

// hashOf is a file's sha256.
func hashOf(at string) (string, error) {
	file, err := os.Open(at)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Check is whether a pinned source is on the shelf as its pin says: an
// error saying what is wrong, or nothing. an unpinned source is said to
// be so, and a tombstone is never checked.
func (shelf Shelf) Check(source Source) (string, error) {
	switch {
	case source.Never() != "":
		return fmt.Sprintf("%s: never fetched", source.Key()), nil
	case !source.Pinned():
		return "", fmt.Errorf("%s: not pinned in the list", source.Key())
	}
	sum, err := hashOf(shelf.Path(source))
	if err != nil {
		return "", fmt.Errorf("%s: not on the shelf: %w", source.Key(), err)
	}
	if sum != source.row.Sha256 {
		return "", fmt.Errorf("%s: sha256 %s on the shelf, the list pins %s",
			source.Key(), sum, source.row.Sha256)
	}
	return fmt.Sprintf("%s: as pinned", source.Key()), nil
}
