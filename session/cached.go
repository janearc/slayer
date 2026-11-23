package session

// What staging works out from numbers and files alone is kept on disk
// (package diskcache), so the next run reads it instead: today the
// moonlit sky's table, which took 8 to 13 seconds of every moonlit
// still (measured 2026-11-12, the fire, moonlake and the cloud sea at
// 640 by 360). Each thing kept is keyed by every number it is made from
// and the build, so a hit is the same to the bit as working it out.
//
// It lives under the cache home, SLAYER_CACHE, and nowhere else.
//
// What came from the cache and what was worked out is noted, one line a
// thing, and the session gives the lines to whoever drew it.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/airlight"
	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/titan"
)

// skyKind keeps moonlit and twilit skies' tables. Its cap is picked by
// the disk caches slice's author, 2026-11-12: a table is 7,512 bytes,
// so 64 MiB keeps about nine thousand skies, more moons and heights
// than a month of films stands under.
var skyKind = diskcache.MustKind("skies", skyCap, "a sky is 7.5 KB; "+
	"this keeps about nine thousand")

// skyCap is the skies' cap in bytes; see skyKind.
const skyCap = 64 << 20

// cacheNote is one thing staging took from a cache or worked out: what,
// whether it was a hit, when the entry was written, how long it took,
// and on a miss why, and whether it was then kept.
//
// A note told is no cache's: a line staging says, its what, as it is.
type cacheNote struct {
	what    string
	hit     bool
	told    bool
	written time.Time
	spent   time.Duration
	why     string
	keptErr error
}

// String is the note as the progress line says it.
func (note cacheNote) String() string {
	if note.told {
		return note.what
	}
	if note.hit {
		return fmt.Sprintf("%s: from the cache, worked out %s, read "+
			"in %.3fs", note.what,
			note.written.Local().Format(time.DateTime), note.spent.Seconds())
	}
	line := fmt.Sprintf("%s: worked out in %.3fs (%s)", note.what,
		note.spent.Seconds(), note.why)
	if note.keptErr != nil {
		return line + ", not kept: " + note.keptErr.Error()
	}
	return line + ", kept"
}

// keeping is how one thing goes through the cache: its kind, the key of
// what it is made from, its name on the progress line, how to work it
// out, and how to make it bytes and read it back.
type keeping[T any] struct {
	key   diskcache.Key
	what  string
	work  func() (T, error)
	bytes func(T) []byte
	from  func([]byte) (T, error)
}

// throughCache is the thing from the cache where it is there and reads back,
// and otherwise worked out and kept; and a note of which.
func throughCache[T any](given keeping[T]) (T, cacheNote, error) {
	began := time.Now()
	note := cacheNote{what: given.what}
	store, err := storeOf(given.key.Kind())
	if err == nil {
		var made T
		made, note, err = fromStore(store, given, note)
		if err == nil && note.hit {
			note.spent = time.Since(began)
			return made, note, nil
		}
	}
	if err != nil {
		note.why = "the cache could not be read: " + err.Error()
	}
	made, err := given.work()
	note.spent = time.Since(began)
	if err != nil {
		return made, note, err
	}
	note.keptErr = errUnkept(store)
	if store != nil {
		note.keptErr = store.Put(given.key, given.bytes(made))
	}
	return made, note, nil
}

// fromStore is the thing read from a store, with the note's hit and
// written said, or the note's why on a miss.
func fromStore[T any](store *diskcache.Store, given keeping[T],
	note cacheNote) (T, cacheNote, error) {
	var made T
	hit, err := store.Get(given.key)
	if err != nil {
		return made, note, err
	}
	if !hit.Found() {
		note.why = hit.Why()
		return made, note, nil
	}
	made, err = given.from(hit.Payload())
	if err != nil {
		note.why = "the entry does not read: " + err.Error()
		return made, note, nil
	}
	note.hit, note.written = true, hit.Written()
	return made, note, nil
}

// errUnkept is why nothing is kept where there is no store.
func errUnkept(store *diskcache.Store) error {
	if store == nil {
		return fmt.Errorf("no cache")
	}
	return nil
}

// stores is each kind's store, opened once a process, or why not.
var stores = struct {
	sync.Mutex
	opened map[string]*diskcache.Store
}{opened: map[string]*diskcache.Store{}}

// storeOf is a kind's store under the home the environment says now,
// each opened once.
func storeOf(kind diskcache.Kind) (*diskcache.Store, error) {
	home, err := diskcache.Home(os.LookupEnv)
	if err != nil {
		return nil, err
	}
	folder := filepath.Join(home, kind.Name())
	stores.Lock()
	defer stores.Unlock()
	if store, has := stores.opened[folder]; has {
		return store, nil
	}
	store, err := diskcache.Open(home, kind)
	if err != nil {
		return nil, err
	}
	stores.opened[folder] = store
	return store, nil
}

// exactly is a float64 as a key says it: the shortest decimal that reads
// back as the same float64, so two numbers one bit apart differ.
func exactly(number float64) string {
	return strconv.FormatFloat(number, 'g', -1, 64)
}

// keptSky is moonlight's sky seen from a height, lit from a way by a
// light at an exposure, from the cache where it was worked out before.
func keptSky(height measure.Length, toward maths.Vec,
	light moonlight.Spectrum, exposure float64, what string) (*moonlight.Sky,
	cacheNote, error) {
	key := skyKey(height, toward, light, exposure)
	return throughCache(keeping[*moonlight.Sky]{key: key, what: what,
		work: func() (*moonlight.Sky, error) {
			return moonlight.NewSky(height, toward, light, exposure)
		},
		bytes: (*moonlight.Sky).Bytes, from: moonlight.SkyFromBytes})
}

// skyKey is the key of a sky: the build and every number NewSky takes.
func skyKey(height measure.Length, toward maths.Vec,
	light moonlight.Spectrum, exposure float64) diskcache.Key {
	return diskcache.NewKey(skyKind).Built().
		Text("height", exactly(height.InMetres())).
		Text("toward", exactly(toward.X)+" "+exactly(toward.Y)+" "+
			exactly(toward.Z)).
		Bytes("light", light.Bytes()).
		Text("exposure", exactly(exposure))
}

// airKind keeps the light worked out in a named air, by package
// airlight, an entry for each air, ground height and pair of the sun's
// zenith bins. Its cap is picked: an Earth entry is 6,254,880 bytes, 21
// wavelengths of 36 by 32 by 16 bins, so 512 MiB keeps about eighty.
var airKind = diskcache.MustKind("air-fields", airCap, "an Earth air's "+
	"fields are about 6.3 MB; this keeps about eighty")

// airCap is the air fields' cap in bytes; see airKind.
const airCap = 512 << 20

// keptFields is the fields a plan decides, from the cache where they
// were worked out before. The key is the plan's every part and not the
// build: a walk takes minutes, and the plan's solver version says when
// the code that walks it has changed what it gives.
func keptFields(plan airlight.Plan) (*airlight.Fields, cacheNote, error) {
	return throughCache(keeping[*airlight.Fields]{
		key: plan.Keyed(diskcache.NewKey(airKind)), what: plan.Said(),
		work: plan.Solve, bytes: (*airlight.Fields).Bytes,
		from: airlight.FieldsFromBytes})
}

// pinKind keeps the sha256 a pinned file was found to have, under its
// path, size, time, inode and device: a file none of those has changed
// on is not read whole again to check its pin.
//
// The coordinator chose
// this over the content (2026-11-12): the content is what is being
// checked, and the 1.8 GB elevation model took a second or more to
// hash on every run. Picked by the disk caches slice's author: an entry
// is about 400 bytes, so 4 MiB is ten thousand files.
var pinKind = diskcache.MustKind("pin-checks", pinCap, "an entry is "+
	"about 400 bytes; this is ten thousand files")

// pinCap is the pin checks' cap in bytes; see pinKind.
const pinCap = 4 << 20

// groundKind keeps the ground a set cuts from its elevation model and
// smooths. Picked by the same author, the same day: moonlake's window is
// tens of megabytes, so 2 GiB keeps dozens of sets' grounds.
var groundKind = diskcache.MustKind("ground", groundCap, "a set's "+
	"ground is tens of megabytes; this keeps dozens")

// groundCap is the grounds' cap in bytes; see groundKind.
const groundCap = 2 << 30

// rememberedHash is a file's sha256, from the cache where the file's
// path, size, time, inode and device are those it was hashed at, and
// otherwise read whole and kept.
func rememberedHash(path string) (string, cacheNote, error) {
	key, err := statKey(path)
	if err != nil {
		hash, hashErr := diskcache.FileDigest(path)
		return hash, cacheNote{what: "the sha256 of " +
			filepath.Base(path), why: err.Error(),
			keptErr: err}, hashErr
	}
	return throughCache(keeping[string]{key: key,
		what: "the sha256 of " + filepath.Base(path) + " (kept by its " +
			"size and time)",
		work:  func() (string, error) { return diskcache.FileDigest(path) },
		bytes: func(sum string) []byte { return []byte(sum) },
		from:  func(raw []byte) (string, error) { return string(raw), nil }})
}

// statKey is the key of a file by what the system says of it.
func statKey(path string) (diskcache.Key, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return diskcache.Key{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return diskcache.Key{}, err
	}
	system, held := info.Sys().(*syscall.Stat_t)
	if !held {
		return diskcache.Key{}, fmt.Errorf("%s: no inode", path)
	}
	return diskcache.NewKey(pinKind).
		Bytes("path", []byte(absolute)).
		Text("size", strconv.FormatInt(info.Size(), 10)).
		Text("modified", strconv.FormatInt(info.ModTime().UnixNano(), 10)).
		Text("inode", strconv.FormatUint(system.Ino, 10)).
		Text("device", strconv.FormatInt(int64(system.Dev), 10)), nil
}

// keptGround is a set's ground cut from the elevation model whose pinned
// sha256 is sum, on its window, smoothed: from the cache where it was
// cut before.
func keptGround(path, sum string, cut groundCut) (terrain.Window,
	cacheNote, error) {
	key := groundKey(sum, cut)
	return throughCache(keeping[terrain.Window]{key: key,
		what: "the ground", work: func() (terrain.Window, error) {
			return cut.from(path)
		},
		bytes: terrain.Window.Bytes, from: terrain.WindowFromBytes})
}

// groundKey is the key of a ground: the pin of the model it is cut from,
// the window, and the smoothing, and where the ground is laid on a sphere,
// the place, its size in metres, whether titan makes it finer than the map
// and the seed it does it by, and the relief it is asked for where one is.
// Everything that decides a height is in it.
func groundKey(sum string, cut groundCut) diskcache.Key {
	key := diskcache.NewKey(groundKind).Built().Digest("model", sum).
		Text("window", exactly(cut.west)+" "+exactly(cut.north)+" "+
			strconv.Itoa(cut.across)+" "+strconv.Itoa(cut.down)+" "+
			exactly(cut.step)).
		Text("smooth", exactly(cut.smooth))
	place := cut.place
	if place == nil {
		return key
	}
	key = key.Text("place", exactly(place.latitude)+" "+
		exactly(place.longitude)+" "+exactly(place.across)+" "+
		exactly(place.down)+" "+strconv.FormatBool(place.detailed)+" "+
		strconv.FormatUint(place.seed, 10))
	if place.relieved {
		key = key.Text("relief", exactly(place.slope)+" "+
			exactly(place.longest))
	}
	return key
}

// keptPlace is a set's ground laid on a sphere, cut from the map whose
// pinned sha256 is sum: from the cache where it was cut before, with how
// much of it was made.
func keptPlace(path, sum string, cut groundCut) (placedGround, cacheNote,
	error) {
	return throughCache(keeping[placedGround]{key: groundKey(sum, cut),
		what: "the ground on a sphere", work: func() (placedGround, error) {
			return cut.placed(path)
		},
		bytes: placedGround.bytes, from: placedFromBytes})
}

// placed is the ground laid on a sphere, cut from the raster at a path.
func (cut groundCut) placed(path string) (placedGround, error) {
	raster, err := terrain.Open(path)
	if err != nil {
		return placedGround{}, err
	}
	defer raster.Close()
	var sphere titan.Sphere = raster
	if earth, isEarth := raster.Earth(); isEarth {
		sphere = earth
	}
	found, err := cut.onSphere(sphere)
	if err != nil {
		return placedGround{}, err
	}
	window, err := found.window.Smoothed(cut.smooth)
	found.window = window
	return found, err
}

// groundCut is where a set's ground is cut from its model, how finely,
// and how much it is smoothed. place is where the ground is laid on a
// sphere, or nil where it is cut by the model's own metres.
type groundCut struct {
	west, north, step, smooth float64
	across, down              int
	place                     *groundPlace
}

// from is the ground cut from the model at a path and smoothed.
func (cut groundCut) from(path string) (terrain.Window, error) {
	raster, err := terrain.Open(path)
	if err != nil {
		return terrain.Window{}, err
	}
	defer raster.Close()
	window, err := raster.Window(cut.west, cut.north, cut.across,
		cut.down, cut.step)
	if err != nil {
		return terrain.Window{}, err
	}
	return window.Smoothed(cut.smooth)
}
