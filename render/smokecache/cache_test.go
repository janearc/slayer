package smokecache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janearc/slayer/scene/smoke"
)

// key is a whole key, every field set.
func key() Key {
	return Key{Exe: "aaaa", Inputs: "$SLAYER_MARBLE nothing", Smoke: 5,
		Size: "text 358x150", Sampling: "text 3", Sensor: "pinhole"}
}

// The key's text names every field, and a change to any one of them is a
// different name in the cache.
func TestEveryFieldIsInTheKey(t *testing.T) {
	k := key()
	for _, want := range []string{"executable aaaa",
		"inputs $SLAYER_MARBLE nothing", "smoke 5", "size text 358x150",
		"sampling text 3", "sensor pinhole"} {
		if !strings.Contains(k.Text(), want+"\n") {
			t.Errorf("the key's text has no %q:\n%s", want, k.Text())
		}
	}
	for name, change := range map[string]func(*Key){
		"executable": func(k *Key) { k.Exe = "x" },
		"inputs":     func(k *Key) { k.Inputs = "x" },
		"smoke":      func(k *Key) { k.Smoke = 6 },
		"size":       func(k *Key) { k.Size = "x" },
		"sampling":   func(k *Key) { k.Sampling = "x" },
		"sensor":     func(k *Key) { k.Sensor = "x" },
	} {
		other := key()
		change(&other)
		if other.Name() == k.Name() {
			t.Errorf("a change to %s kept the same name", name)
		}
	}
}

// The cache is under $XDG_CACHE_HOME where that is set, and under the
// user's cache directory where it is not.
func TestRoot(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/somewhere")
	if got, _ := Root(); got != "/somewhere/smokecheck" {
		t.Errorf("with XDG_CACHE_HOME, %s", got)
	}
	t.Setenv("XDG_CACHE_HOME", "")
	dir, _ := os.UserCacheDir()
	if got, _ := Root(); got != filepath.Join(dir, "smokecheck") {
		t.Errorf("without, %s; want %s", got, filepath.Join(dir, "smokecheck"))
	}
}

// The key's executable is the sha256 of the binary that is running.
func TestTheBinaryIsTheKey(t *testing.T) {
	k, err := Here()
	if err != nil {
		t.Fatal(err)
	}
	path, _ := os.Executable()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	if want := hex.EncodeToString(sum[:]); k.Exe != want {
		t.Errorf("executable %s, want %s", k.Exe, want)
	}
}

// write puts s in dir/name, making the folders it needs.
func write(t *testing.T, dir, name, s string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// reads is scene/smoke's own Reads, for one input.
func reads(in string) ([]string, error) {
	return smoke.Smoke{Inputs: []string{in}}.Reads()
}

// What a smoke reads is in its key by what it declared and the contents
// of each file: the same files in another folder, or through a link, are
// the same key, and a changed byte, a renamed file or one more file is
// another. A smoke that declares nothing reads nothing.
func TestInputsAreKeyedByWhatTheSmokeReads(t *testing.T) {
	if in, err := Inputs(nil, reads); err != nil || in != "none" {
		t.Errorf("declaring nothing: %q, %v", in, err)
	}
	t.Setenv("SLAYER_MARBLE", "")
	unset, err := Inputs([]string{"$SLAYER_MARBLE"}, reads)
	if err != nil || unset != "$SLAYER_MARBLE nothing" {
		t.Errorf("unset: %q, %v", unset, err)
	}
	marbles := func(dir string, files map[string]string) string {
		t.Helper()
		for name, s := range files {
			write(t, dir, name, s)
		}
		t.Setenv("SLAYER_MARBLE", dir)
		in, err := Inputs([]string{"$SLAYER_MARBLE"}, reads)
		if err != nil {
			t.Fatal(err)
		}
		return in
	}
	two := map[string]string{"blue.jpg": "day", "black.jpg": "night"}
	first := marbles(t.TempDir(), two)
	if !strings.HasPrefix(first, `$SLAYER_MARBLE "black.jpg" 5 `) ||
		!strings.Contains(first, `, "blue.jpg" 3 `) {
		t.Fatalf("inputs %q", first)
	}
	if again := marbles(t.TempDir(), two); again != first {
		t.Errorf("the same files elsewhere: %q, want %q", again, first)
	}
	real, link := t.TempDir(), filepath.Join(t.TempDir(), "marbles")
	marbles(real, two)
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if got := marbles(link, nil); got != first {
		t.Errorf("the same files through a link: %q", got)
	}
	for name, files := range map[string]map[string]string{
		"a changed byte": {"blue.jpg": "dax", "black.jpg": "night"},
		"a renamed file": {"blue.jpeg": "day", "black.jpg": "night"},
		"one more file": {"blue.jpg": "day", "black.jpg": "night",
			"red.jpg": ""},
	} {
		if marbles(t.TempDir(), files) == first {
			t.Errorf("%s kept the same inputs", name)
		}
	}
	t.Setenv("SLAYER_MARBLE", filepath.Join(t.TempDir(), "gone"))
	if _, err := Inputs([]string{"$SLAYER_MARBLE"}, reads); err == nil {
		t.Error("a folder that is not there was keyed")
	}
}

// A plain path is keyed as the smoke declared it, and by its bytes: the
// same bytes at another declared path are another key.
func TestAPlainPathIsKeyedAsDeclared(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.jpg", "day")
	write(t, dir, "b.jpg", "day")
	a, err := Inputs([]string{filepath.Join(dir, "a.jpg")}, reads)
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("%s %q 3 ", filepath.Join(dir, "a.jpg"),
		filepath.Join(dir, "a.jpg")); !strings.HasPrefix(a, want) {
		t.Errorf("%q, want it to open %q", a, want)
	}
	b, err := Inputs([]string{filepath.Join(dir, "b.jpg")}, reads)
	if err != nil || b == a {
		t.Errorf("another path, the same bytes: %q, %v", b, err)
	}
	if _, err := Inputs([]string{filepath.Join(dir, "c.jpg")},
		reads); err == nil {
		t.Error("a path that is not there was keyed")
	}
}

// git runs git in dir, failing the test if it fails.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=smokecheck",
		"GIT_AUTHOR_EMAIL=smokecheck@here", "GIT_COMMITTER_NAME=smokecheck",
		"GIT_COMMITTER_EMAIL=smokecheck@here")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// The record says a tree's commit, and what is uncommitted in it, an
// untracked file included and an ignored one not. It never stops a run.
func TestTheRecordSaysWhatIsUncommitted(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "a.go", "package a\n")
	write(t, dir, ".gitignore", "bin/\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "--no-verify", "-m", "one")
	clean := commit(dir)
	if len(clean) != 40 {
		t.Fatalf("a clean tree: %q", clean)
	}
	write(t, dir, "bin/smokecheck", "built")
	if got := commit(dir); got != clean {
		t.Errorf("an ignored file: %q", got)
	}
	write(t, dir, "b.go", "package a\n")
	if got := commit(dir); !strings.HasPrefix(got, clean+", with "+
		"uncommitted changes: ?? b.go") {
		t.Errorf("an untracked file: %q", got)
	}
	if got := commit(t.TempDir()); !strings.HasPrefix(got, "unknown: ") {
		t.Errorf("not a git tree: %q", got)
	}
}

// fill writes an entry's files, each saying which run wrote it.
func fill(who string) func(string) error {
	return func(dir string) error {
		for _, f := range []string{Text, Real, Verdict} {
			if err := os.WriteFile(filepath.Join(dir, f),
				[]byte(who), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
}

// says is what an entry's files say, all three, or why they cannot be
// read.
func says(e *Entry) (string, error) {
	var got []string
	for _, f := range []string{Text, Real, Verdict} {
		b, err := e.Bytes(f)
		if err != nil {
			return "", err
		}
		got = append(got, string(b))
	}
	if got[0] != got[1] || got[1] != got[2] {
		return "", fmt.Errorf("an entry from more than one run: %q", got)
	}
	return got[0], nil
}

// opened is what the entry for k under root says, failing the test when
// it is not whole.
func opened(t *testing.T, root string, k Key) string {
	t.Helper()
	e, ok := Open(root, k)
	if !ok {
		t.Fatal("no whole entry")
	}
	defer e.Close()
	s, err := says(e)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// An entry stored is found whole and kept, unless replaced; one missing a
// file or holding another key is not served, and the next store mends it;
// and nothing is left behind.
func TestStoreAndOpen(t *testing.T) {
	root, k := t.TempDir(), key()
	if _, ok := Open(root, k); ok {
		t.Fatal("found an entry in an empty cache")
	}
	store := func(who string, replace bool) {
		t.Helper()
		e, err := Store(root, k, "a record", replace, fill(who))
		if err != nil {
			t.Fatal(err)
		}
		if s, err := says(e); err != nil || s != who {
			t.Errorf("stored %q, and is %q, %v", who, s, err)
		}
		e.Close()
	}
	store("first", false)
	if got := opened(t, root, k); got != "first" {
		t.Errorf("stored first, opened %q", got)
	}
	store("second", false)
	if got := opened(t, root, k); got != "first" {
		t.Errorf("a whole entry was not kept: %q", got)
	}
	store("third", true)
	if got := opened(t, root, k); got != "third" {
		t.Errorf("replacing left %q", got)
	}
	dir := filepath.Join(root, "v1", k.Name())
	if b, _ := os.ReadFile(filepath.Join(dir, Record)); string(b) !=
		"a record" {
		t.Errorf("the record is %q", b)
	}
	os.Remove(filepath.Join(dir, Real))
	if _, ok := Open(root, k); ok {
		t.Error("an entry missing a picture was served")
	}
	store("fourth", false)
	if got := opened(t, root, k); got != "fourth" {
		t.Errorf("a broken entry was not mended: %q", got)
	}
	os.WriteFile(filepath.Join(dir, keyFile), []byte("another"), 0o644)
	if _, ok := Open(root, k); ok {
		t.Error("an entry holding another key was served")
	}
	if left, _ := filepath.Glob(filepath.Join(root, "v1", ".*")); len(
		left) > 0 {
		t.Errorf("left behind: %v", left)
	}
}

// An entry held open is still what it was after another run replaces it,
// and the cache serves the new one.
func TestAHeldEntryOutlivesItsReplacement(t *testing.T) {
	root, k := t.TempDir(), key()
	e, err := Store(root, k, "", false, fill("old"))
	if err != nil {
		t.Fatal(err)
	}
	e.Close()
	held, ok := Open(root, k)
	if !ok {
		t.Fatal("no whole entry")
	}
	defer held.Close()
	if e, err := Store(root, k, "", true, fill("new")); err != nil {
		t.Fatal(err)
	} else {
		e.Close()
	}
	if s, err := says(held); err != nil || s != "old" {
		t.Errorf("the held entry says %q, %v", s, err)
	}
	if got := opened(t, root, k); got != "new" {
		t.Errorf("the cache serves %q", got)
	}
}

// Runs storing and replacing one key at once for a second, with others
// reading it, all succeed: no store fails, no reader gets an entry made by
// two runs, and one whole entry is left, with nothing else.
func TestRunsAtOnce(t *testing.T) {
	root, k := t.TempDir(), key()
	const writers, readers = 8, 4
	stop := time.Now().Add(time.Second)
	var wg sync.WaitGroup
	var stores, reads atomic.Int64
	for w := range writers {
		wg.Go(func() {
			for r := 0; time.Now().Before(stop); r++ {
				who := fmt.Sprintf("writer %d round %d", w, r)
				e, err := Store(root, k, "", w > 0, fill(who))
				if err != nil {
					t.Error(err)
					return
				}
				if s, err := says(e); err != nil || s != who {
					t.Errorf("%s stored, holds %q, %v", who, s, err)
				}
				e.Close()
				stores.Add(1)
			}
		})
	}
	for range readers {
		wg.Go(func() {
			for time.Now().Before(stop) {
				e, ok := Open(root, k)
				if !ok {
					continue
				}
				if _, err := says(e); err != nil {
					t.Error(err)
				}
				e.Close()
				reads.Add(1)
			}
		})
	}
	wg.Wait()
	t.Logf("%d stores, %d whole reads", stores.Load(), reads.Load())
	opened(t, root, k)
	all, _ := os.ReadDir(filepath.Join(root, "v1"))
	if len(all) != 1 {
		var names []string
		for _, d := range all {
			names = append(names, d.Name())
		}
		t.Errorf("v1 holds %v, want the one entry", names)
	}
}
