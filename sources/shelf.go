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

	"gopkg.in/yaml.v3"
)

// A Shelf is where fetched sources are kept on this machine: a
// directory outside any repository, and how to reach the sources.
type Shelf struct {
	dir    string
	client *http.Client
	hash   func(path string) (string, error)
	// progress, when set, is told how each download goes.
	progress Progress
}

// NewShelf is a shelf in a directory, fetching through a client.
func NewShelf(dir string, client *http.Client) Shelf {
	return Shelf{dir: dir, client: client}
}

// HashedBy is the shelf checking a file it is asked to read (Reads,
// Check) against its pin by the sha256 a function gives for its path:
// one that keeps the sha256s it has taken, so a large file is not read
// whole on every run. Fetching and tending still read every byte. A nil
// function is the sha256 of the file's bytes, each time.
func (shelf Shelf) HashedBy(hash func(path string) (string, error)) Shelf {
	shelf.hash = hash
	return shelf
}

// hashed is the sha256 of the file at a path, by the shelf's function.
func (shelf Shelf) hashed(at string) (string, error) {
	if shelf.hash == nil {
		return hashOf(at)
	}
	return shelf.hash(at)
}

// Placed is the variable that says where a person's shelf is: a
// directory of their own that lasts past a reboot, so a dem of an
// island is fetched once, and not a temporary one (since 2026-10-16).
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
	return filepath.Join(shelf.dir, fileName(source))
}

// fileName is the name a source's bytes have on the shelf: its key, and
// the ending its url's path has.
func fileName(source Source) string {
	name := source.Key()
	if from, err := url.Parse(source.From()); err == nil {
		if ending := path.Ext(from.Path); endings[ending] {
			name += ending
		}
	}
	return name
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
// with no pin they are kept, and the hash is said for pinning. a source
// with a pinned member of its archive has the member unpacked, or kept
// if it is there as pinned.
func (shelf Shelf) Fetch(ctx context.Context, source Source,
	now time.Time) (string, error) {
	said, err := shelf.fetched(ctx, source, now)
	if err != nil || source.row.Unpack == nil {
		return said, err
	}
	more, err := shelf.unpackedFor(source)
	return said + "; " + more, err
}

// fetched is Fetch of the source's own bytes, without its member.
func (shelf Shelf) fetched(ctx context.Context, source Source,
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
		err = held(source.row.Sha256, source.row.Bytes, sum, size)
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

// held is what is wrong with fetched bytes, their sha256 and size, against
// the list's pin and size (empty and zero where it gives none), or nothing.
func held(pin string, bytes int64, sum string, size int64) error {
	if pin != "" && sum != pin {
		return fmt.Errorf("%w: sha256 %s, the list pins %s", errFetch, sum,
			pin)
	}
	if bytes != nothing && size != bytes {
		return fmt.Errorf("%w: %d bytes, the list says %d", errFetch, size,
			bytes)
	}
	return nil
}

// download writes a source's bytes to a path, hashing them as they come.
func (shelf Shelf) download(ctx context.Context, source Source,
	to string) (string, int64, error) {
	if err := os.MkdirAll(shelf.dir, 0o755); err != nil {
		return "", nothing, err
	}
	key := source.Key()
	return downloadVia(ctx, shelf.client, source.From(), to,
		func(body io.Reader, total int64) io.Reader {
			return shelf.watched(key, body, total)
		})
}

// download writes the bytes at a url to a path through a getter, hashing
// them as they come, and gives their sha256 and size.
func download(ctx context.Context, get getter, from, to string) (string,
	int64, error) {
	return downloadVia(ctx, get, from, to, nil)
}

// downloadVia is download with the body read through a wrapper, given
// the body and the length the server announced (-1 when it did not). A
// nil wrapper reads the body as it is.
func downloadVia(ctx context.Context, get getter, from, to string,
	wrap func(body io.Reader, total int64) io.Reader) (string,
	int64, error) {
	asking, err := http.NewRequestWithContext(ctx, http.MethodGet, from, nil)
	if err != nil {
		return "", nothing, err
	}
	answer, err := get.Do(asking)
	if err != nil {
		return "", nothing, err
	}
	defer answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return "", nothing, fmt.Errorf("%w: %s answered %s", errFetch, from,
			answer.Status)
	}
	file, err := os.Create(to)
	if err != nil {
		return "", nothing, err
	}
	hash := sha256.New()
	var body io.Reader = answer.Body
	if wrap != nil {
		body = wrap(body, answer.ContentLength)
	}
	size, err := io.Copy(io.MultiWriter(file, hash), body)
	if closing := file.Close(); err == nil {
		err = closing
	}
	return hex.EncodeToString(hash.Sum(nil)), size, err
}

// sidecar writes what is known of fetched bytes beside them.
func (shelf Shelf) sidecar(source Source, at, sum string, size int64,
	now time.Time) error {
	out, err := os.OpenFile(at+".source", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	encoder := yaml.NewEncoder(out)
	err = encoder.Encode(side{Key: source.Key(),
		From: source.From(), Licence: source.Licence(), Sha256: sum,
		Bytes: size, Fetched: now.UTC().Format(time.RFC3339)})
	return errors.Join(err, encoder.Close(), out.Close())
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
	sum, err := shelf.hashed(shelf.Path(source))
	if err != nil {
		return "", fmt.Errorf("%s: not on the shelf: %w", source.Key(), err)
	}
	if sum != source.row.Sha256 {
		return "", fmt.Errorf("%s: sha256 %s on the shelf, the list pins %s",
			source.Key(), sum, source.row.Sha256)
	}
	return fmt.Sprintf("%s: as pinned", source.Key()), nil
}
