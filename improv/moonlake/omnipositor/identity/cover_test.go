package identity

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
)

// abcDigest is the SHA-256 of the three bytes "abc", from FIPS 180-4.
const abcDigest = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

// TestKeyStringIsTheHexDigestOfTheRecipe checks that a key prints as 64
// lowercase hex characters, that a key built from files equals the one built
// from their hashes, and that the frame number changes it.
func TestKeyStringIsTheHexDigestOfTheRecipe(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "abc")
	if err := os.WriteFile(file, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := HashFile(file); err != nil || got != abcDigest {
		t.Fatalf("hash of abc: %q, %v", got, err)
	}
	spec := Spec{BinaryPath: file, Settings: []byte("s"), Inputs: map[string]Input{"in": {Path: file, Choice: "c"}},
		FPS: 24, Frame: 1, Device: "d", Backend: "b", Width: 2, Height: 3}
	built := build(t, spec)
	pre, err := BuildPrehashed(PrehashedSpec{BinarySHA256: abcDigest, Settings: []byte("s"),
		Inputs: map[string]HashedInput{"in": {SHA256: abcDigest, Choice: "c"}},
		FPS:    24, Frame: 1, Device: "d", Backend: "b", Width: 2, Height: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(built.String()) {
		t.Errorf("key %q is not 64 lowercase hex characters", built.String())
	}
	if built.String() != pre.String() {
		t.Errorf("file key %s differs from prehashed key %s", built, pre)
	}
	spec.Frame = 2
	if build(t, spec).String() == built.String() {
		t.Error("the frame number did not change the key")
	}
}

// TestBuildRefusesWhatItCannotHash checks the refusals that name an input,
// an empty path and a file that cannot be opened.
func TestBuildRefusesWhatItCannotHash(t *testing.T) {
	spec := fixture(t)
	spec.Inputs = map[string]Input{"fire.steady": {Path: spec.BinaryPath + "-gone"}}
	if _, err := Build(spec); err == nil || !strings.Contains(err.Error(), `identity: input "fire.steady"`) {
		t.Errorf("a missing input: %v", err)
	}
	if _, err := HashFile(""); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Errorf("an empty path: %v", err)
	}
	if os.Geteuid() != 0 {
		locked := filepath.Join(t.TempDir(), "locked")
		if err := os.WriteFile(locked, []byte("x"), 0o000); err != nil {
			t.Fatal(err)
		}
		if _, err := HashFile(locked); err == nil || !strings.Contains(err.Error(), "cannot open regular input") {
			t.Errorf("an unreadable file: %v", err)
		}
	}
}

// TestPrehashedRefusesABadInput checks an input hash or name that is not
// acceptable.
func TestPrehashedRefusesABadInput(t *testing.T) {
	good := PrehashedSpec{BinarySHA256: abcDigest, Settings: []byte("s"), FPS: 24, Device: "d", Backend: "b", Width: 2, Height: 3}
	for name, input := range map[string]struct {
		key  string
		hash HashedInput
	}{
		"short hash":     {"in", HashedInput{SHA256: "abc"}},
		"upper case":     {"in", HashedInput{SHA256: strings.ToUpper(abcDigest)}},
		"empty name":     {" ", HashedInput{SHA256: abcDigest}},
		"bad utf8":       {"in", HashedInput{SHA256: abcDigest, Choice: "\xff"}},
		"bad utf8 name":  {"\xff", HashedInput{SHA256: abcDigest}},
		"unclosed digit": {"in", HashedInput{SHA256: abcDigest[:63] + "g"}},
	} {
		spec := good
		spec.Inputs = map[string]HashedInput{input.key: input.hash}
		if _, err := BuildPrehashed(spec); err == nil || !strings.Contains(err.Error(), "invalid prehashed input") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestOpenRefusesWhatIsNotADirectory checks that a regular file is not a
// store.
func TestOpenRefusesWhatIsNotADirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(file); err == nil || !strings.Contains(err.Error(), "cannot open output directory") {
		t.Errorf("a file opened as a store: %v", err)
	}
}

// openStore opens a store on a fresh directory and returns both.
func openStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store, dir
}

// TestLockHoldsTheMutexUntilUnlock checks the context-free lock.
func TestLockHoldsTheMutexUntilUnlock(t *testing.T) {
	store, _ := openStore(t)
	if err := store.lock(); err != nil {
		t.Fatal(err)
	}
	if store.mu.TryLock() {
		t.Error("the mutex was free while the store was locked")
	}
	store.unlock()
	if !store.mu.TryLock() {
		t.Fatal("the mutex stayed held after unlock")
	}
	store.mu.Unlock()
}

// TestLockFailsWhenTheDirectoryHandleIsGone checks that a flock error that
// is not contention is returned and leaves the mutex free.
func TestLockFailsWhenTheDirectoryHandleIsGone(t *testing.T) {
	store, _ := openStore(t)
	store.dir.Close()
	err := store.lockContext(context.Background())
	if !errors.Is(err, syscall.EBADF) {
		t.Fatalf("lock on a closed handle: %v", err)
	}
	if !store.mu.TryLock() {
		t.Error("the mutex stayed held after a failed lock")
	}
	store.mu.Unlock()
}

// scripted is a context whose Err answers by call number, and which can act
// on the call: it makes a failure land at a chosen step of an operation.
type scripted struct {
	context.Context
	calls int
	step  func(call int) error
}

// Err counts the call and returns what the script says for it.
func (c *scripted) Err() error { c.calls++; return c.step(c.calls) }

// failAt is a scripted context that returns err at one call and nil otherwise.
func failAt(call int, err error) *scripted {
	return &scripted{Context: context.Background(), step: func(n int) error {
		if n == call {
			return err
		}
		return nil
	}}
}

// doAt is a scripted context that runs act at one call and returns nil.
func doAt(call int, act func()) *scripted {
	return &scripted{Context: context.Background(), step: func(n int) error {
		if n == call {
			act()
		}
		return nil
	}}
}

// TestVerifyContextStopsWhereTheContextEnds checks the three places after
// the lock where it asks the context, and the directory sync between them.
func TestVerifyContextStopsWhereTheContextEnds(t *testing.T) {
	store, _ := openStore(t)
	key := build(t, fixture(t))
	if _, err := store.Verify(Key{}); err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Errorf("a zero key: %v", err)
	}
	if _, err := store.Publish(key, tiny(t, 1), false); err != nil {
		t.Fatal(err)
	}
	for call := 2; call <= 3; call++ {
		if _, err := store.VerifyContext(failAt(call, context.Canceled), key); !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled at call %d: %v", call, err)
		}
	}
	if _, err := store.VerifyContext(doAt(2, func() { store.dir.Close() }), key); err == nil || errors.Is(err, ErrCorrupt) {
		t.Errorf("a directory that cannot sync: %v", err)
	}
}

// fixedRandom is a reader of a repeated byte, so a generation name is known.
type fixedRandom byte

// Read fills p with the byte.
func (f fixedRandom) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(f)
	}
	return len(p), nil
}

// failingRandom is a reader that always fails.
type failingRandom struct{}

// Read fails.
func (failingRandom) Read([]byte) (int, error) { return 0, errors.New("no randomness") }

// useRandom replaces the random source for the rest of the test.
func useRandom(t *testing.T, source io.Reader) {
	t.Helper()
	previous := rand.Reader
	rand.Reader = source
	t.Cleanup(func() { rand.Reader = previous })
}

// generation is the generation name a fixedRandom(1) source gives.
const generation = ".generation-01010101010101010101010101010101"

// entries is the names in a directory.
func entries(t *testing.T, dir string) []string {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, each := range list {
		names = append(names, each.Name())
	}
	return names
}

// TestPublishRefusesBadKeysAndHugeData checks the checks made before the lock.
func TestPublishRefusesBadKeysAndHugeData(t *testing.T) {
	store, dir := openStore(t)
	key := build(t, fixture(t))
	if _, err := store.Publish(Key{}, tiny(t, 1), false); err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Errorf("a zero key: %v", err)
	}
	if _, err := store.Publish(key, make([]byte, maxPNGBytes+1), false); !errors.Is(err, ErrCorrupt) {
		t.Errorf("data over the limit: %v", err)
	}
	if names := entries(t, dir); len(names) != 0 {
		t.Errorf("a refused publish left %v", names)
	}
}

// TestPublishStopsCleanlyWhereTheContextEnds checks each place a publish asks
// its context, that the error is the context's, and that no generation or
// slot is left behind.
func TestPublishStopsCleanlyWhereTheContextEnds(t *testing.T) {
	key := build(t, fixture(t))
	for _, call := range []int{2, 3, 4, 5} {
		store, dir := openStore(t)
		_, err := store.PublishContext(failAt(call, context.Canceled), key, tiny(t, 1), false)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("call %d: %v", call, err)
		}
		if names := entries(t, dir); len(names) != 0 {
			t.Errorf("call %d left %v", call, names)
		}
	}
}

// TestPublishReportsADirectorySyncFailure checks each sync of the store's
// directory: after a good frame is found, after the link is made, and after
// it is installed.
func TestPublishReportsADirectorySyncFailure(t *testing.T) {
	key := build(t, fixture(t))
	data := tiny(t, 1)
	store, _ := openStore(t)
	if _, err := store.Publish(key, data, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishContext(doAt(3, func() { store.dir.Close() }), key, data, false); err == nil {
		t.Error("a good frame was reported with a directory that cannot sync")
	}
	fresh, dir := openStore(t)
	if _, err := fresh.PublishContext(doAt(4, func() { fresh.dir.Close() }), key, data, false); err == nil {
		t.Error("a link was kept with a directory that cannot sync")
	}
	if names := entries(t, dir); len(names) != 0 {
		t.Errorf("a failed sync left %v", names)
	}
	last, dir := openStore(t)
	_, err := last.PublishContext(doAt(5, func() { last.dir.Close() }), key, data, false)
	if err == nil {
		t.Error("an installed frame was reported synced with a directory that cannot sync")
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got, err := reopened.Verify(key); err != nil || !bytes.Equal(got, data) {
		t.Errorf("the installed frame after a late failure: %v", err)
	}
}

// TestPublishFailsWhenItCannotMakeItsGeneration checks a source of
// randomness that fails, and a generation name that is already taken.
func TestPublishFailsWhenItCannotMakeItsGeneration(t *testing.T) {
	key := build(t, fixture(t))
	store, _ := openStore(t)
	useRandom(t, failingRandom{})
	if _, err := store.Publish(key, tiny(t, 1), false); err == nil || !strings.Contains(err.Error(), "no randomness") {
		t.Errorf("failing randomness: %v", err)
	}
	useRandom(t, fixedRandom(1))
	taken, dir := openStore(t)
	if err := os.Mkdir(filepath.Join(dir, generation), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := taken.Publish(key, tiny(t, 1), false); !errors.Is(err, os.ErrExist) {
		t.Errorf("a taken generation: %v", err)
	}
}

// TestPublishCleansUpWhenItsLinkIsTaken checks that a link that cannot be
// made removes the generation it wrote, and returns the error.
func TestPublishCleansUpWhenItsLinkIsTaken(t *testing.T) {
	useRandom(t, fixedRandom(1))
	key := build(t, fixture(t))
	store, dir := openStore(t)
	if err := os.WriteFile(filepath.Join(dir, generation+".link"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(key, tiny(t, 1), false); !errors.Is(err, os.ErrExist) {
		t.Errorf("a taken link: %v", err)
	}
	if names := entries(t, dir); len(names) != 0 {
		t.Errorf("a failed publish left %v", names)
	}
}

// TestPublishSlotTakenDuringTheWriteIsRefused checks that a slot that turns
// up between the check and the final link is not overwritten.
func TestPublishSlotTakenDuringTheWriteIsRefused(t *testing.T) {
	key := build(t, fixture(t))
	store, dir := openStore(t)
	taking := doAt(5, func() {
		slotPath := filepath.Join(dir, slot(key))
		if err := os.WriteFile(slotPath, []byte("not a frame"), 0o600); err != nil {
			t.Error(err)
		}
	})
	if _, err := store.PublishContext(taking, key, tiny(t, 1), false); !errors.Is(err, os.ErrExist) {
		t.Errorf("a slot taken during the publish: %v", err)
	}
	slotPath := filepath.Join(dir, slot(key))
	if raw, err := os.ReadFile(slotPath); err != nil || string(raw) != "not a frame" {
		t.Errorf("the slot was changed: %q, %v", raw, err)
	}
	if names := entries(t, dir); len(names) != 1 {
		t.Errorf("a failed publish left %v", names)
	}
}

// TestRedrawKeepsTheOldFrameWhenItFails checks each failure of a redraw after
// its new generation is written: the old frame still verifies.
func TestRedrawKeepsTheOldFrameWhenItFails(t *testing.T) {
	key := build(t, fixture(t))
	old, fresh := tiny(t, 1), tiny(t, 2)
	setup := func(t *testing.T) (*Store, string) {
		store, dir := openStore(t)
		if _, err := store.Publish(key, old, false); err != nil {
			t.Fatal(err)
		}
		return store, dir
	}
	t.Run("pending name taken", func(t *testing.T) {
		store, dir := setup(t)
		useRandom(t, fixedRandom(1))
		if err := os.WriteFile(filepath.Join(dir, generation+".pending"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Publish(key, fresh, true); !errors.Is(err, os.ErrExist) {
			t.Errorf("a taken pending name: %v", err)
		}
		if got, err := store.Verify(key); err != nil || !bytes.Equal(got, old) {
			t.Errorf("the old frame after a failed redraw: %v", err)
		}
	})
	t.Run("context ends after the pending link", func(t *testing.T) {
		store, _ := setup(t)
		if _, err := store.PublishContext(failAt(5, context.Canceled), key, fresh, true); !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled: %v", err)
		}
		if got, err := store.Verify(key); err != nil || !bytes.Equal(got, old) {
			t.Errorf("the old frame after a cancelled redraw: %v", err)
		}
	})
	t.Run("slot lost before the rename", func(t *testing.T) {
		store, dir := setup(t)
		losing := doAt(5, func() {
			slotPath := filepath.Join(dir, slot(key))
			if err := os.Remove(slotPath); err != nil {
				t.Error(err)
			}
		})
		if _, err := store.PublishContext(losing, key, fresh, true); !errors.Is(err, ErrMissing) {
			t.Errorf("a slot lost during the redraw: %v", err)
		}
	})
	t.Run("redraw replaces", func(t *testing.T) {
		store, _ := setup(t)
		again, err := store.Publish(key, fresh, true)
		if err != nil || again {
			t.Fatalf("redraw: %v %v", again, err)
		}
		if got, err := store.Verify(key); err != nil || !bytes.Equal(got, fresh) {
			t.Errorf("the redrawn frame: %v", err)
		}
	})
}

// TestVerifyRefusesAMalformedSlotAndManifestFile checks a slot whose link does
// not name a generation, and a manifest that is not a regular file.
func TestVerifyRefusesAMalformedSlotAndManifestFile(t *testing.T) {
	key := build(t, fixture(t))
	store, dir := openStore(t)
	for _, target := range []string{"bogus/frame.png", ".generation-abc/frame.png", ".generation-" + strings.Repeat("g", 32) + "/frame.png"} {
		link := filepath.Join(dir, slot(key))
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Verify(key); !errors.Is(err, ErrCorrupt) {
			t.Errorf("slot linked to %q: %v", target, err)
		}
		os.Remove(link)
	}
	if _, err := store.Publish(key, tiny(t, 1), false); err != nil {
		t.Fatal(err)
	}
	slotPath := filepath.Join(dir, slot(key))
	target, err := os.Readlink(slotPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, filepath.Dir(target), "manifest.json")
	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(manifestPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(key); !errors.Is(err, ErrCorrupt) {
		t.Errorf("a manifest that is a directory: %v", err)
	}
	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, make([]byte, 16385), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(key); !errors.Is(err, ErrCorrupt) {
		t.Errorf("a manifest over its limit: %v", err)
	}
}

// TestWriteRefusesAnExistingFile checks that write creates and never replaces.
func TestWriteRefusesAnExistingFile(t *testing.T) {
	store, dir := openStore(t)
	if err := store.write("a", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := store.write("a", []byte("two")); !errors.Is(err, os.ErrExist) {
		t.Errorf("a second write: %v", err)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "a")); string(raw) != "one" {
		t.Errorf("the file holds %q", raw)
	}
}

// pngChunk is one PNG chunk of the kind and payload, with its checksum.
func pngChunk(kind string, payload []byte) []byte {
	body := append([]byte(kind), payload...)
	size := uint32(len(payload))
	out := binary.BigEndian.AppendUint32(nil, size)
	out = append(out, body...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(body))
}

// pngHeader is an IHDR payload for the size, bit depth and colour type.
func pngHeader(width, height int, depth, colour byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(width))
	out = binary.BigEndian.AppendUint32(out, uint32(height))
	return append(out, depth, colour, 0, 0, 0)
}

// zlibOf is the zlib stream of raw.
func zlibOf(raw []byte) []byte {
	var out bytes.Buffer
	w := zlib.NewWriter(&out)
	w.Write(raw)
	w.Close()
	return out.Bytes()
}

// pngSignature is the eight bytes every PNG starts with.
var pngSignature = []byte{137, 80, 78, 71, 13, 10, 26, 10}

// pngOf joins the signature and the chunks.
func pngOf(chunks ...[]byte) []byte {
	return append(append([]byte{}, pngSignature...), bytes.Join(chunks, nil)...)
}

// TestPublishTakesEveryColourTypeItCanCount checks that two by three images
// of gray, RGB and gray with alpha are stored and read back byte for byte.
func TestPublishTakesEveryColourTypeItCanCount(t *testing.T) {
	var gray, rgb bytes.Buffer
	blank := image.NewGray(image.Rect(0, 0, 2, 3))
	if err := png.Encode(&gray, blank); err != nil {
		t.Fatal(err)
	}
	opaque := image.NewRGBA(image.Rect(0, 0, 2, 3))
	for i := 3; i < len(opaque.Pix); i += 4 {
		opaque.Pix[i] = 255
	}
	if err := png.Encode(&rgb, opaque); err != nil {
		t.Fatal(err)
	}
	rows := bytes.Repeat([]byte{0, 1, 2, 3, 4}, 3)
	alphaHeader := pngChunk("IHDR", pngHeader(2, 3, 8, 4))
	alphaData := pngChunk("IDAT", zlibOf(rows))
	alphaEnd := pngChunk("IEND", nil)
	grayAlpha := pngOf(alphaHeader, alphaData, alphaEnd)
	for name, data := range map[string][]byte{"gray": gray.Bytes(), "rgb": rgb.Bytes(), "gray with alpha": grayAlpha} {
		store, _ := openStore(t)
		key := build(t, fixture(t))
		if _, err := store.Publish(key, data, false); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got, err := store.Verify(key); err != nil || !bytes.Equal(got, data) {
			t.Errorf("%s read back: %v", name, err)
		}
	}
}

// TestPublishRefusesMalformedPNGs checks that each way of building a PNG
// wrongly is refused as corrupt and nothing is stored.
func TestPublishRefusesMalformedPNGs(t *testing.T) {
	header := pngChunk("IHDR", pngHeader(2, 3, 8, 0))
	grayRows := bytes.Repeat([]byte{0, 1, 2}, 3)
	idat := pngChunk("IDAT", zlibOf(grayRows))
	end := pngChunk("IEND", nil)
	flipped := pngOf(header, idat, end)
	flipped[8+4+4+13] ^= 1
	badFilter := zlibOf([]byte{5, 1, 2, 0, 1, 2, 0, 1, 2})
	colourFive := pngChunk("IHDR", pngHeader(2, 3, 8, 5))
	badZlib := pngChunk("IDAT", []byte("xx"))
	unknown := pngChunk("tEXt", []byte("a"))
	interlacedBody := append(pngHeader(2, 3, 8, 0)[:12], 1)
	interlaced := pngChunk("IHDR", interlacedBody)
	endWithData := pngChunk("IEND", []byte("x"))
	cases := map[string][]byte{
		"idat before the header": pngOf(idat, header, end),
		"bad checksum":           flipped,
		"no end chunk":           pngOf(header, idat),
		"colour type five":       pngOf(colourFive, idat, end),
		"bad zlib header":        pngOf(header, badZlib, end),
		"filter byte of five":    pngOf(header, pngChunk("IDAT", badFilter), end),
		"row too short":          pngOf(header, pngChunk("IDAT", zlibOf([]byte{0, 1})), end),
		"unknown chunk":          pngOf(header, unknown, idat, end),
		"second header":          pngOf(header, header, idat, end),
		"interlaced header":      pngOf(interlaced, idat, end),
		"end with data":          pngOf(header, idat, endWithData),
		"end before data":        pngOf(header, end),
		"trailing bytes":         append(pngOf(header, idat, end), 0),
		"too short":              {137, 80},
	}
	key := build(t, fixture(t))
	for name, data := range cases {
		store, dir := openStore(t)
		if _, err := store.Publish(key, data, false); !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: %v", name, err)
		}
		if names := entries(t, dir); len(names) != 0 {
			t.Errorf("%s left %v", name, names)
		}
	}
}
