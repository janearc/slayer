package job

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// whole is a job that says something of every part, for the key tests
// to change one part at a time.
const whole = film + `film:
  {width: 640, height: 360, rays: 2, wide: 50, exposure: 0.5}
renderer: gpu
quality: preview
start: {value: 4, unit: s}
scene-duration: {value: 8, unit: s}
shutter: {angle: {value: 180, unit: deg}, glimpses: 2}
stock: {name: dusk, seed: 7, rated: 1}
camera:
  name: rim
` + zoom + `titles:
  rows: 24
  elements:
    first: {kind: text, words: A, ink: ink, across: 0.1, down: 0.1}
    second: {kind: text, words: B, ink: ink, across: 0.2, down: 0.2}
sound: {file: take.wav, from: {value: 1, unit: s}}
outputs: {frames: frames, movie: film.mp4, note: film.note}
limits: {memory: {value: 8, unit: GiB}, time: {value: 60, unit: s}}
`

// keyOf is the key of the job text says, written in the directory all
// of a test's jobs are, so their relative paths are the same files.
func keyOf(t *testing.T, text string) string {
	t.Helper()
	directory := filepath.Join(os.TempDir(), "job-keys-"+
		strings.ReplaceAll(t.Name(), "/", "-"))
	if err := os.MkdirAll(directory, private|executable); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	job, err := Read(writtenIn(t, directory, "job.yaml", text))
	if err != nil {
		t.Fatal(err)
	}
	return job.Key()
}

// changed is whole with before replaced by after, which must be there.
func changed(t *testing.T, before, after string) string {
	t.Helper()
	if !strings.Contains(whole, before) {
		t.Fatalf("the whole job has no %q", before)
	}
	return strings.Replace(whole, before, after, one)
}

// TestTheKeyIgnoresWhatDoesNotChangeThePicture holds the key steady
// across outputs, the note, limits, key order, and defaults said.
func TestTheKeyIgnoresWhatDoesNotChangeThePicture(t *testing.T) {
	key := keyOf(t, whole)
	for name, text := range map[string]string{
		"outputs": changed(t, "frames: frames, ", ""),
		"note":    changed(t, ", note: film.note", ""),
		"limits": changed(t, "limits: {memory: {value: 8, unit: GiB}, "+
			"time: {value: 60, unit: s}}\n", ""),
		"order": changed(t, "renderer: gpu\nquality: preview\n",
			"quality: preview\nrenderer: gpu\n"),
		"inner order": changed(t, "{name: dusk, seed: 7, rated: 1}",
			"{rated: 1, name: dusk, seed: 7}"),
		"ease said": changed(t, "unit: s}}\n", "unit: s}, ease: linear}\n"),
	} {
		if got := keyOf(t, text); got != key {
			t.Errorf("changing the %s changed the key", name)
		}
	}
}

// TestTheKeyFillsInDefaults holds that saying a default and leaving it
// unsaid give the same key, and that a preset is its size.
func TestTheKeyFillsInDefaults(t *testing.T) {
	for _, pair := range [][2]string{
		{still, still + "quality: exact\n"},
		{film, film + "start: {value: 0, unit: s}\n"},
		{still + "film: {preset: 720}\n",
			still + "film: {width: 1280, height: 720}\n"},
		{still + "print: {knee: 0.5, top: 8}\n",
			still + "print: {knee: 0.5, top: 8, bloom: 0, stops: 0}\n"},
		{still + "stock: {name: ash}\n",
			still + "stock: {name: ash, seed: 0, rated: 0}\n"},
		{still + pathCamera, still + pathCamera +
			"  up: {x: 0, y: 1, z: 0}\n"},
		{film + "sound: {file: a.wav}\n",
			film + "sound: {file: a.wav, from: {value: 0, unit: s}}\n"},
		{retitled("      align: centre\n", ""),
			retitled("align: centre", "align: left")},
		{retitled("value: 0.25", "value: {from: 0.25, to: 0.25}"),
			titled},
	} {
		if keyOf(t, pair[nothing]) != keyOf(t, pair[one]) {
			t.Errorf("%q and %q have different keys", pair[nothing],
				pair[one])
		}
	}
}

// TestEveryPictureValueChangesTheKey is the anti-test: a change to any
// part that changes the picture or the sound changes the key.
// anti-test of TestTheKeyIgnoresWhatDoesNotChangeThePicture
// anti-test of TestTheKeyIsStableUnderReorderingAndOutputs
// anti-test of TestTheKeyFillsInDefaults
func TestEveryPictureValueChangesTheKey(t *testing.T) {
	key := keyOf(t, whole)
	for name, text := range map[string]string{
		"scene": changed(t, "moonlake.yaml", "melancholia.yaml"),
		"film":  changed(t, "rays: 2", "rays: 3"),
		"lens": changed(t, "exposure: 0.5}",
			"exposure: 0.5, aperture: 0.1, focus: 12}"),
		"renderer": changed(t, "renderer: gpu", "renderer: cpu"),
		"quality":  changed(t, "quality: preview", "quality: exact"),
		"camera":   changed(t, "name: rim", "name: low"),
		"zoom":     changed(t, "value: 20, unit: deg", "value: 21, unit: deg"),
		"time":     changed(t, "fps: 24", "fps: 25"),
		"start":    changed(t, "start: {value: 4", "start: {value: 5"),
		"shutter":  changed(t, "glimpses: 2", "glimpses: 3"),
		"stock":    changed(t, "seed: 7", "seed: 8"),
		"titles":   changed(t, "words: B", "words: C"),
		"title order": changed(t, "first: {kind: text, words: A",
			"third: {kind: text, words: A"),
		"sound": changed(t, "take.wav", "other.wav"),
		"print": changed(t, "stock: {name: dusk, seed: 7, rated: 1}",
			"print: {knee: 0.5, top: 8}"),
	} {
		if got := keyOf(t, text); got == key {
			t.Errorf("changing the %s left the key alone", name)
		}
	}
}

// TestCanonicalIsSortedJSONWithoutOutputs holds the canonical form to
// JSON that says no output, limit or note, and lists the titles in order.
func TestCanonicalIsSortedJSONWithoutOutputs(t *testing.T) {
	job := read(t, whole)
	canonical := job.Canonical()
	var decoded map[string]any
	if err := json.Unmarshal(canonical, &decoded); err != nil {
		t.Fatal(err)
	}
	again, _ := json.Marshal(decoded)
	if string(again) != string(canonical) {
		t.Error("the canonical form is not in sorted order")
	}
	text := string(canonical)
	for _, absent := range []string{"outputs", "limits", "note"} {
		if strings.Contains(text, absent) {
			t.Errorf("the canonical form says %s", absent)
		}
	}
	if strings.Index(text, `"first"`) > strings.Index(text, `"second"`) {
		t.Error("the canonical form reordered the titles")
	}
	canonical[nothing] = ' '
	if job.Canonical()[nothing] == ' ' {
		t.Error("changing the canonical bytes changed the job")
	}
}

// shuffled are the lines a job may be written in, in any order, for
// the key's stability property.
var shuffled = []string{"fps: 24\n", "duration: {value: 2, unit: s}\n",
	"renderer: gpu\n", "quality: preview\n",
	"stock: {name: dusk, seed: 7, rated: 1}\n",
	"start: {value: 4, unit: s}\n"}

// TestTheKeyIsStableUnderReorderingAndOutputs is the property: any order
// of the lines, and any outputs or limits, give the same key.
func TestTheKeyIsStableUnderReorderingAndOutputs(t *testing.T) {
	key := keyOf(t, still+strings.Join(shuffled, ""))
	if keyOf(t, still) == key {
		t.Fatal("the shuffled lines change nothing")
	}
	rapid.Check(t, func(trial *rapid.T) {
		order := rapid.Permutation(shuffled).Draw(trial, "order")
		frames := rapid.StringMatching("[a-z]{1,8}").Draw(trial, "frames")
		memory := rapid.IntRange(one, mostRate).Draw(trial, "memory")
		text := still + strings.Join(order, "") + "outputs: {frames: " +
			frames + "}\nlimits: {memory: {value: " +
			strconv.Itoa(memory) + ", unit: GiB}}\n"
		if keyOf(t, text) != key {
			trial.Fatalf("%q changed the key", text)
		}
	})
}
