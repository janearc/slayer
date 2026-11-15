package hillside

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
)

// The sky the cache's tests stand under: the fire's moon from the
// fire's lowest ground, at a picture's exposure. Picked, from the night
// fire's set, by the disk caches slice's author, 2026-11-12.
const (
	cachedHeight   = 1100.0
	cachedExposure = 1e5
)

// cachedToward is the way to the test's moon.
var cachedToward = maths.V(0.6, 0.47, 0.64).Unit()

// cachedHome is a fresh cache home for a test.
func cachedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("SLAYER_CACHE", home)
	return home
}

func TestASkyOneBitApartIsAMiss(t *testing.T) {
	cachedHome(t)
	height, light := measure.Metres(cachedHeight), moonlight.Full()
	kept := skyKey(height, cachedToward, light, cachedExposure)
	store, err := storeOf(skyKind)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(kept, []byte("a sky")); err != nil {
		t.Fatal(err)
	}
	brighter := moonlight.Spectrum{}
	for nm, power := range light {
		brighter[nm] = power
	}
	brighter[550] = math.Nextafter(brighter[550], math.Inf(1))
	apart := cachedToward
	apart.X = math.Nextafter(apart.X, 1)
	for name, key := range map[string]diskcache.Key{
		"height": skyKey(measure.Metres(math.Nextafter(cachedHeight, 0)),
			cachedToward, light, cachedExposure),
		"toward":   skyKey(height, apart, light, cachedExposure),
		"light":    skyKey(height, cachedToward, brighter, cachedExposure),
		"exposure": skyKey(height, cachedToward, light, cachedExposure*2),
	} {
		hit, err := store.Get(key)
		if err != nil || hit.Found() {
			t.Errorf("a sky of another %s was served: %v, %v", name, hit,
				err)
		}
	}
	if hit, _ := store.Get(kept); !hit.Found() {
		t.Fatal("the sky put was not found, so the misses prove nothing")
	}
}

// stringly is a keeping of a string, worked out by work.
func stringly(t *testing.T, name string, work func() (string,
	error)) keeping[string] {
	t.Helper()
	kind, err := diskcache.NewKind("hillside-"+name, skyCap, "a test's")
	if err != nil {
		t.Fatal(err)
	}
	return keeping[string]{key: diskcache.NewKey(kind).Text("a", "1"),
		what: name, work: work,
		bytes: func(made string) []byte { return []byte(made) },
		from: func(raw []byte) (string, error) {
			if string(raw) == "unreadable" {
				return "", errors.New("unreadable")
			}
			return string(raw), nil
		}}
}

func TestWorkThatFailsIsNotKept(t *testing.T) {
	cachedHome(t)
	failing := stringly(t, "failing", func() (string, error) {
		return "", errors.New("no sky")
	})
	if _, _, err := throughCache(failing); err == nil {
		t.Fatal("work that failed was not said to")
	}
	store, _ := storeOf(failing.key.Kind())
	if hit, _ := store.Get(failing.key); hit.Found() {
		t.Fatal("work that failed was kept")
	}
}

func TestAnEntryThatDoesNotReadIsWorkedOutAgain(t *testing.T) {
	cachedHome(t)
	worked := 0
	given := stringly(t, "unread", func() (string, error) {
		worked++
		return "fresh", nil
	})
	store, _ := storeOf(given.key.Kind())
	if err := store.Put(given.key, []byte("unreadable")); err != nil {
		t.Fatal(err)
	}
	made, note, err := throughCache(given)
	if err != nil || made != "fresh" || worked != 1 || note.hit ||
		!strings.Contains(note.why, "does not read") {
		t.Fatalf("%q, %v, %v; worked %d", made, note, err, worked)
	}
	again, note, _ := throughCache(given)
	if again != "fresh" || !note.hit || worked != 1 {
		t.Fatalf("the entry worked out again was not kept: %v", note)
	}
}

func TestWithNoCacheTheWorkIsStillDone(t *testing.T) {
	t.Setenv("SLAYER_CACHE", "relative")
	given := stringly(t, "homeless", func() (string, error) {
		return "made", nil
	})
	made, note, err := throughCache(given)
	if err != nil || made != "made" || note.hit ||
		!strings.Contains(note.String(), "not kept") ||
		!strings.Contains(note.why, "could not be read") {
		t.Fatalf("%q, %v, %v", made, note, err)
	}
}

func TestAKeyWithARefusalIsWorkedOutAndNotKept(t *testing.T) {
	cachedHome(t)
	given := stringly(t, "refused", func() (string, error) {
		return "made", nil
	})
	given.key = given.key.Text("a", "again")
	made, note, err := throughCache(given)
	if err != nil || made != "made" || note.keptErr == nil {
		t.Fatalf("%q, %v, %v", made, note, err)
	}
}

func TestANoteSaysWhereItCameFrom(t *testing.T) {
	written := time.Date(2026, 11, 12, 14, 2, 0, 0, time.Local)
	for _, each := range []struct {
		note cacheNote
		says string
	}{
		{cacheNote{what: "the sky", hit: true, written: written,
			spent: time.Millisecond},
			"the sky: from the cache, worked out 2026-11-12 14:02:00, " +
				"read in 0.001s"},
		{cacheNote{what: "the sky", spent: time.Second, why: "no entry"},
			"the sky: worked out in 1.000s (no entry), kept"},
		{cacheNote{what: "the sky", spent: time.Second, why: "no entry",
			keptErr: errors.New("disk full")},
			"the sky: worked out in 1.000s (no entry), not kept: disk full"},
	} {
		if said := each.note.String(); said != each.says {
			t.Errorf("said %q, want %q", said, each.says)
		}
	}
}

func TestASessionSaysWhatItsStagingKept(t *testing.T) {
	session := &Session{given: asked{cached: []cacheNote{
		{what: "the sky", why: "no entry"}}}}
	lines := session.Cached()
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "the sky: worked") {
		t.Fatalf("lines %q", lines)
	}
	if lines := (&Session{}).Cached(); len(lines) != nothing {
		t.Fatalf("a session that kept nothing said %q", lines)
	}
}
