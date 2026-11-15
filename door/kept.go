package door

// A still whose job, files and build are unchanged is not drawn again:
// its picture is kept on disk (package diskcache) under the job's
// picture key, the build, and the sources list nearest its set, whose
// pins are the content of every file the set reads from the shelf.
//
// Only a set's still is kept; a smoke reads folders the environment names,
// and a film's frames have their own ledger. --fresh draws regardless,
// and keeps what it drew.
//
// A kept still is served only where SLAYER_KEPT_STILLS is "on". The key
// does not yet hold every file a set reads from beside itself (its hud
// theme, for one: review of 2026-11-12), so a still served by default
// could be an old picture of a changed set. Stills are kept either way.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/job"
)

// pictureKind keeps finished stills. Its cap is picked by the disk
// caches slice's author, 2026-11-12: a full-size still's picture is a
// few megabytes, so 4 GiB keeps about a thousand.
var pictureKind = diskcache.MustKind("pictures", pictureCap, "a "+
	"full-size still is a few MB; this keeps about a thousand")

// pictureCap is the pictures' cap in bytes; see pictureKind.
const pictureCap = 4 << 30

// keptStillsVariable, set to keptStillsOn, serves a kept still in place
// of drawing it; see the comment at the head of this file.
const (
	keptStillsVariable = "SLAYER_KEPT_STILLS"
	keptStillsOn       = "on"
)

// sourcesList is the file a set's pinned sources are listed in.
const sourcesList = "sources.yaml"

// pictureKey is the key of a job's still: its picture key, the build,
// and the sources list nearest its set. A job that is not a set's still
// is refused, so it is neither read nor kept.
func pictureKey(wanted job.Job) diskcache.Key {
	key := diskcache.NewKey(pictureKind).Built().
		Digest("picture", wanted.PictureKey())
	set, isSet := wanted.Scene().Set()
	if !isSet || !wanted.Clock().Still() {
		return key.Text("kept", "only a set's still").
			Text("kept", "is kept")
	}
	listed, err := nearestList(filepath.Dir(set))
	if err != nil {
		// with no list a set reads nothing from the shelf: staging
		// refuses one that would, and nothing is kept
		return key.Text("sources", "none")
	}
	sum, err := diskcache.FileDigest(listed)
	if err != nil {
		return key.Text("sources", "unread").Text("sources", "unread")
	}
	return key.Digest("sources", sum)
}

// nearestList is the sources list in a folder or the nearest above it.
func nearestList(folder string) (string, error) {
	for {
		listed := filepath.Join(folder, sourcesList)
		if _, err := os.Stat(listed); err == nil {
			return listed, nil
		}
		above := filepath.Dir(folder)
		if above == folder {
			return "", fmt.Errorf("no %s above %s", sourcesList, folder)
		}
		folder = above
	}
}

// keptPicture is the still's picture from the cache, written where the
// places say, and whether it was there; nothing is written on a miss.
func (run runner) keptPicture() (bool, error) {
	if served, _ := run.env.lookup(keptStillsVariable); run.fresh ||
		served != keptStillsOn {
		return false, nil
	}
	store, err := pictureStore(run.env)
	if err != nil {
		return false, nil
	}
	hit, err := store.Get(pictureKey(run.wanted))
	if err != nil || !hit.Found() {
		return false, nil
	}
	if err := writtenBeside(run.at.picture, hit.Payload()); err != nil {
		return false, err
	}
	_, err = fmt.Fprintf(run.out, "wrote %s from the cache, drawn %s\n",
		run.at.picture, hit.Written().Local().Format(time.DateTime))
	return true, err
}

// keepPicture keeps the picture just written, and says whether it could.
func (run runner) keepPicture() error {
	key := pictureKey(run.wanted)
	if key.Err() != nil {
		return nil
	}
	store, err := pictureStore(run.env)
	if err == nil {
		var written []byte
		if written, err = os.ReadFile(run.at.picture); err == nil {
			err = store.Put(key, written)
		}
	}
	if err != nil {
		_, err = fmt.Fprintf(run.out, "the picture is not kept: %v\n", err)
		return err
	}
	_, err = fmt.Fprintln(run.out, "kept the picture for the next draw")
	return err
}

// pictureStore is the pictures' store under the env's cache home.
func pictureStore(env Env) (*diskcache.Store, error) {
	home, err := diskcache.Home(env.lookup)
	if err != nil {
		return nil, err
	}
	return diskcache.Open(home, pictureKind)
}

// writtenBeside writes bytes beside a path and renames them onto it, so
// a reader sees the old file or the new, never half of one.
func writtenBeside(path string, written []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return err
	}
	part, err := os.CreateTemp(filepath.Dir(path),
		"."+filepath.Base(path)+"-*"+partSuffix)
	if err != nil {
		return err
	}
	_, err = part.Write(written)
	if err == nil {
		err = part.Chmod(fileMode)
	}
	err = errors.Join(err, part.Close())
	if err == nil {
		err = os.Rename(part.Name(), path)
	}
	if err != nil {
		return errors.Join(err, os.Remove(part.Name()))
	}
	return nil
}
