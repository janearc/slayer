package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/searching"
)

// harness runs one verb of a throwaway tool that declares the same flags
// as the real command, so the code under test receives a real call.
type harness struct {
	out, errs bytes.Buffer
	failed    error
	ran       bool
}

// run invokes fn as the verb named name with args and records its error.
func (h *harness) run(name string, args []string, fn func(pzm.Call) error) {
	tool := pzm.New("searching", "test tool", pzm.Build(""), pzm.Built(""), pzm.Who(""), pzm.Sessions(""))
	writes := verb.NewFacts(verb.WritesTo("the named new output"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb(verb.Name(name), verb.NoShort, "test verb", "Runs the code under test.", verb.Example("Example_"+name), writes,
		func(call pzm.Call) error {
			h.ran = true
			h.failed = fn(call)
			return h.failed
		},
		pzm.Value("at", verb.NoShort, "seconds"),
		pzm.Switch("defer-movie", verb.NoShort, "encode after drawing"),
		pzm.Switch("serial", verb.NoShort, "serial comparison"),
		pzm.Value("site", verb.NoShort, "site"),
		pzm.Value("from", verb.NoShort, "first frame"),
		pzm.Value("frames", verb.NoShort, "frame count"),
		pzm.Value("limit-seconds", verb.NoShort, "wall limit"),
		pzm.Value("out", verb.NoShort, "output"))
	tool.RunWith(append([]string{name}, args...), strings.NewReader(""), &h.out, &h.errs)
}

// wavBytes returns a mono 16-bit wav of the given length in seconds.
func wavBytes(rate int, seconds int) []byte {
	data := make([]byte, 0, rate*seconds*2)
	for i := 0; i < rate*seconds; i++ {
		sample := int16((i % 100) * 100)
		data = binary.LittleEndian.AppendUint16(data, uint16(sample))
	}
	var out []byte
	out = append(out, "RIFF"...)
	chunkSize := uint32(36 + len(data))
	out = binary.LittleEndian.AppendUint32(out, chunkSize)
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint32(out, uint32(rate))
	out = binary.LittleEndian.AppendUint32(out, uint32(rate*2))
	out = binary.LittleEndian.AppendUint16(out, 2)
	out = binary.LittleEndian.AppendUint16(out, 16)
	out = append(out, "data"...)
	dataSize := uint32(len(data))
	out = binary.LittleEndian.AppendUint32(out, dataSize)
	return append(out, data...)
}

// shoot writes a small valid shoot, score and song into a new directory
// and returns the path of the shoot file. The song bytes are the given ones.
func shoot(t *testing.T, song []byte) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "song.wav"), song, 0o644); err != nil {
		t.Fatal(err)
	}
	var times, marks []string
	for i := 0; i < 40; i++ {
		beat := float64(i) * 0.5
		times = append(times, fmt.Sprintf("%g", beat))
		marks = append(marks, "1")
	}
	score := fmt.Sprintf("score:\n  audio: song.wav\n  sha256: %x\n  duration: 20\n  meter: 4\n  status: estimated\n  method: test\n  beats:\n    times: [%s]\n    measured: [%s]\n",
		sha256.Sum256(song), strings.Join(times, ", "), strings.Join(marks, ", "))
	if err := os.WriteFile(filepath.Join(dir, "score.yaml"), []byte(score), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := `name: test shoot
music: song.wav
score: score.yaml
width: 160
height: 90
fps: 6
rays: 1
lock: "2.1"
radio-one: "3.1"
radio-two: "4.1"
reveal: "5.1"
coda-bars: 1
search-cut-bars: 1
beat-motion: [1, 0.45, 0.18, 0.03]
beat-two-accents: []
search-days: 6
shared-azimuth-deg: 25
shared-altitude-deg: 55
optical-diameter-metres: 38
optical-aperture-metres: 8
array-count: 3
exposure: 1.5
bloom: 0.65
`
	path := filepath.Join(dir, "shoot.yaml")
	if err := os.WriteFile(path, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// needRoom skips a test that must pass the real disk guard when the
// machine has too little free space for it.
func needRoom(t *testing.T) {
	t.Helper()
	if err := resources(t.TempDir()); err != nil {
		t.Skip("resource guard refuses this machine: " + err.Error())
	}
}

// TestPlanDescribesTheEdit checks the plan verb's JSON against the cues
// worked out by hand from a half-second beat grid.
func TestPlanDescribesTheEdit(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	var h harness
	h.run("plan", []string{path}, func(call pzm.Call) error {
		edit, err := read(call)
		if err != nil {
			return err
		}
		return describe(call, edit)
	})
	if h.failed != nil {
		t.Fatal(h.failed)
	}
	var got map[string]any
	if err := json.Unmarshal(h.out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	// bar 2 beat 1 is beat index 4, at 0.5 seconds a beat: 2 seconds.
	want := map[string]float64{"lock": 2, "radio-one": 4, "radio-two": 6, "reveal": 8,
		"coda": 18, "duration": 20, "frames": 120, "fps": 6, "width": 160, "height": 90}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v, want %v", key, got[key], value)
		}
	}
	if got["name"] != "test shoot" {
		t.Errorf("name = %v", got["name"])
	}
}

// TestReadRefusesBadArguments checks the argument count and a missing file.
func TestReadRefusesBadArguments(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	cases := map[string][]string{
		"none":    {},
		"two":     {path, path},
		"missing": {filepath.Join(t.TempDir(), "nope.yaml")},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var h harness
			h.run("plan", args, func(call pzm.Call) error {
				_, err := read(call)
				return err
			})
			if !h.ran || h.failed == nil {
				t.Fatalf("read accepted %v", args)
			}
		})
	}
}

// TestIntegerReadsGivenValues checks the fallback, a parsed value and a
// refused word.
func TestIntegerReadsGivenValues(t *testing.T) {
	cases := []struct {
		args []string
		want int
		bad  bool
	}{
		{nil, 7, false},
		{[]string{"--from", "12"}, 12, false},
		{[]string{"--from", "twelve"}, 0, true},
	}
	for _, c := range cases {
		var h harness
		var got int
		h.run("render", c.args, func(call pzm.Call) error {
			var err error
			got, err = integer(call, "from", 7)
			return err
		})
		if (h.failed != nil) != c.bad || (!c.bad && got != c.want) {
			t.Errorf("args %v: got %d, err %v", c.args, got, h.failed)
		}
	}
}

// TestStillRefusesBadRequests checks every refusal that comes before any
// renderer is built.
func TestStillRefusesBadRequests(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "x.png")
	cases := map[string][]string{
		"no shoot":      {},
		"unparsed at":   {path, "--at", "soon", "--out", out},
		"negative at":   {path, "--at", "-1", "--out", out},
		"past the end":  {path, "--at", "20", "--out", out},
		"not a number":  {path, "--at", "NaN", "--out", out},
		"infinite":      {path, "--at", "Inf", "--out", out},
		"no out":        {path, "--at", "3"},
		"bad limit":     {path, "--at", "3", "--out", out, "--limit-seconds", "0"},
		"limit too big": {path, "--at", "3", "--out", out, "--limit-seconds", "28801"},
		"limit a word":  {path, "--at", "3", "--out", out, "--limit-seconds", "x"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var h harness
			h.run("still", args, func(call pzm.Call) error { return still(context.Background(), call) })
			if !h.ran || h.failed == nil {
				t.Fatal("still accepted the request")
			}
			if _, err := os.Stat(out); err == nil {
				t.Fatal("a picture was written")
			}
		})
	}
}

// TestStillDryRunDescribesAndDrawsNothing checks that a dry run prints the
// plan and leaves the output path alone.
func TestStillDryRunDescribesAndDrawsNothing(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "x.png")
	var h harness
	h.run("still", []string{path, "--out", out, "--dry-run"}, func(call pzm.Call) error { return still(context.Background(), call) })
	if h.failed != nil {
		t.Fatal(h.failed)
	}
	if !strings.Contains(h.out.String(), `"frames":120`) {
		t.Errorf("no plan in output: %q", h.out.String())
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("dry run wrote a picture")
	}
}

// TestStillStopsWhenCancelled checks that a cancelled context reaches the
// renderer and that no picture is left behind.
func TestStillStopsWhenCancelled(t *testing.T) {
	needRoom(t)
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "x.png")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var h harness
	h.run("still", []string{path, "--at", "3", "--out", out}, func(call pzm.Call) error { return still(ctx, call) })
	if h.failed == nil || !strings.Contains(h.failed.Error(), "canceled") {
		t.Fatalf("err = %v, want a cancellation", h.failed)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("a picture was written")
	}
}

// TestStillRefusesUnknownSite checks the site override is validated by
// the renderer before any drawing.
func TestStillRefusesUnknownSite(t *testing.T) {
	needRoom(t)
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "x.png")
	var h harness
	h.run("still", []string{path, "--at", "3", "--site", "mars", "--out", out},
		func(call pzm.Call) error { return still(context.Background(), call) })
	if h.failed == nil || !strings.Contains(h.failed.Error(), "unknown shoot site") {
		t.Fatalf("err = %v", h.failed)
	}
}

// TestStillRefusesGuardFailure checks that an output directory that does
// not exist stops the run at the resource guard.
func TestStillRefusesGuardFailure(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "missing", "x.png")
	var h harness
	h.run("still", []string{path, "--out", out}, func(call pzm.Call) error { return still(context.Background(), call) })
	if h.failed == nil {
		t.Fatal("still ran against a missing directory")
	}
}

// TestRenderRefusesBadRequests checks the frame range and output checks.
func TestRenderRefusesBadRequests(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "film")
	cases := map[string][]string{
		"no shoot":      {},
		"from a word":   {path, "--from", "a", "--out", out},
		"frames a word": {path, "--frames", "a", "--out", out},
		"negative from": {path, "--from", "-1", "--out", out},
		"zero frames":   {path, "--frames", "0", "--out", out},
		"from the end":  {path, "--from", "120", "--out", out},
		"too many":      {path, "--from", "100", "--frames", "21", "--out", out},
		"no out":        {path},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var h harness
			h.run("render", args, func(call pzm.Call) error { return render(context.Background(), call) })
			if !h.ran || h.failed == nil {
				t.Fatal("render accepted the request")
			}
			if _, err := os.Stat(out); err == nil {
				t.Fatal("output directory was made")
			}
		})
	}
}

// TestRenderDryRunMakesNothing checks a dry run describes and creates no
// directory.
func TestRenderDryRunMakesNothing(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "film")
	var h harness
	h.run("render", []string{path, "--out", out, "--dry-run"}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed != nil {
		t.Fatal(h.failed)
	}
	if !strings.Contains(h.out.String(), `"fps":6`) {
		t.Errorf("no plan in output: %q", h.out.String())
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("dry run made the output directory")
	}
}

// TestRenderRefusesUndecodableSong checks that a song AVFoundation cannot
// read stops the run after the recipes are copied.
func TestRenderRefusesUndecodableSong(t *testing.T) {
	needRoom(t)
	path := shoot(t, []byte("not a song"))
	out := filepath.Join(t.TempDir(), "film")
	var h harness
	h.run("render", []string{path, "--out", out, "--frames", "2"}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed == nil {
		t.Fatal("render accepted an undecodable song")
	}
	for _, name := range []string{"shoot.yaml", "score.yaml"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("recipe %s not kept: %v", name, err)
		}
	}
}

// TestRenderRefusesTrimPastTheSong checks the audio trim bound: a one
// second song cannot cover a start at frame 100.
func TestRenderRefusesTrimPastTheSong(t *testing.T) {
	needRoom(t)
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "film")
	var h harness
	h.run("render", []string{path, "--out", out, "--from", "100", "--frames", "2"}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed == nil || !strings.Contains(h.failed.Error(), "audio trim past end") {
		t.Fatalf("err = %v", h.failed)
	}
}

// TestRenderStopsWhenCancelled checks that a cancelled context ends the
// run with an error and publishes no movie.
func TestRenderStopsWhenCancelled(t *testing.T) {
	needRoom(t)
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "film")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var h harness
	h.run("render", []string{path, "--out", out, "--frames", "2"}, func(call pzm.Call) error { return render(ctx, call) })
	if h.failed == nil {
		t.Fatal("render ignored cancellation")
	}
	if _, err := os.Stat(filepath.Join(out, "animation.mp4")); err == nil {
		t.Fatal("a movie was published")
	}
	if _, err := os.Stat(filepath.Join(out, "render.json")); err == nil {
		t.Fatal("a receipt was written")
	}
}

// TestRenderRefusesExistingRecipe checks that copying the recipes never
// replaces a file already in the output directory.
func TestRenderRefusesExistingRecipe(t *testing.T) {
	needRoom(t)
	path := shoot(t, wavBytes(8000, 1))
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "shoot.yaml"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	var h harness
	h.run("render", []string{path, "--out", out, "--frames", "2"}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed == nil {
		t.Fatal("render replaced an existing recipe")
	}
	if got, _ := os.ReadFile(filepath.Join(out, "shoot.yaml")); string(got) != "keep" {
		t.Fatalf("recipe was changed to %q", got)
	}
}

// TestRenderRefusesUncreatableOutput checks the output directory error.
func TestRenderRefusesUncreatableOutput(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var h harness
	h.run("render", []string{path, "--out", filepath.Join(blocker, "film")}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed == nil {
		t.Fatal("render made a directory under a file")
	}
}

// TestCopyInputsRefusesMissingScore checks a recipe file that cannot be
// read stops the copy.
func TestCopyInputsRefusesMissingScore(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	edit, err := searching.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(edit.Plan.Score); err != nil {
		t.Fatal(err)
	}
	var h harness
	h.run("render", []string{path}, func(call pzm.Call) error { return copyInputs(call, edit, t.TempDir()) })
	if h.failed == nil {
		t.Fatal("copy accepted a missing score")
	}
}

// TestWritePNGWritesOnceAndRefusesReplacement checks the written picture
// reads back with the same pixel and that a second write is refused.
func TestWritePNGWritesOnceAndRefusesReplacement(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(1, 1, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	path := filepath.Join(t.TempDir(), "p.png")
	if err := writePNG(path, picture); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	back, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := back.At(1, 1).RGBA()
	if r>>8 != 200 || g>>8 != 100 || b>>8 != 50 {
		t.Errorf("pixel = %d %d %d", r>>8, g>>8, b>>8)
	}
	if err := writePNG(path, picture); err == nil {
		t.Fatal("second write replaced the file")
	}
}

// failingImage is an image whose pixel reads panic-free but whose bounds
// are empty, which the PNG encoder refuses.
type failingImage struct{ image.RGBA }

// TestWritePNGRemovesAFailedFile checks that an image the encoder refuses
// leaves no file behind.
func TestWritePNGRemovesAFailedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.png")
	if err := writePNG(path, &failingImage{}); err == nil {
		t.Fatal("empty image was written")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("failed picture was left behind")
	}
}

// TestResourcesRefusesMissingDisk checks that a path that does not exist
// is an error from the disk check.
func TestResourcesRefusesMissingDisk(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	if err := resources(gone); err == nil {
		t.Fatal("missing path was accepted")
	}
}

// TestGuardStopsAtTheWallLimit checks the deadline cancels the context
// with its own cause.
func TestGuardStopsAtTheWallLimit(t *testing.T) {
	needRoom(t)
	var h harness
	h.run("still", []string{"--limit-seconds", "1"}, func(call pzm.Call) error {
		ctx, stop, err := guard(context.Background(), call, t.TempDir())
		if err != nil {
			return err
		}
		defer stop()
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Second):
			return fmt.Errorf("guard did not stop")
		}
		if cause := context.Cause(ctx); cause == nil || !strings.Contains(cause.Error(), "wall-time limit") {
			return fmt.Errorf("cause = %v", cause)
		}
		return nil
	})
	if h.failed != nil {
		t.Fatal(h.failed)
	}
}

// TestGuardStopsWhenTheDiskGoesAway checks the once-a-second sample
// cancels the context with the sampling error.
func TestGuardStopsWhenTheDiskGoesAway(t *testing.T) {
	needRoom(t)
	dir := filepath.Join(t.TempDir(), "disk")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var h harness
	h.run("still", nil, func(call pzm.Call) error {
		ctx, stop, err := guard(context.Background(), call, dir)
		if err != nil {
			return err
		}
		defer stop()
		if err := os.Remove(dir); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Second):
			return fmt.Errorf("guard did not notice")
		}
		if cause := context.Cause(ctx); cause == nil || strings.Contains(cause.Error(), "wall-time") {
			return fmt.Errorf("cause = %v", cause)
		}
		return nil
	})
	if h.failed != nil {
		t.Fatal(h.failed)
	}
}

// TestGuardStopFreesTheWatcher checks that the returned stop function
// ends the context without a cause from the guard.
func TestGuardStopFreesTheWatcher(t *testing.T) {
	needRoom(t)
	var h harness
	h.run("still", nil, func(call pzm.Call) error {
		ctx, stop, err := guard(context.Background(), call, t.TempDir())
		if err != nil {
			return err
		}
		stop()
		if ctx.Err() == nil {
			return fmt.Errorf("context still live after stop")
		}
		return nil
	})
	if h.failed != nil {
		t.Fatal(h.failed)
	}
}

// TestRenderRefusesBadWallLimit checks that the guard's refusal of a
// zero wall limit stops the run before any recipe is copied.
func TestRenderRefusesBadWallLimit(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	out := filepath.Join(t.TempDir(), "film")
	var h harness
	h.run("render", []string{path, "--out", out, "--limit-seconds", "0"}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed == nil || !strings.Contains(h.failed.Error(), "wall limit") {
		t.Fatalf("err = %v", h.failed)
	}
	if _, err := os.Stat(filepath.Join(out, "shoot.yaml")); err == nil {
		t.Fatal("a recipe was copied")
	}
}

// TestRenderRefusesExistingMovie checks that a movie already at the
// output path is never replaced.
func TestRenderRefusesExistingMovie(t *testing.T) {
	needRoom(t)
	path := shoot(t, wavBytes(8000, 1))
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "animation.mp4"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	var h harness
	h.run("render", []string{path, "--out", out, "--frames", "2"}, func(call pzm.Call) error { return render(context.Background(), call) })
	if h.failed == nil {
		t.Fatal("render accepted an existing movie")
	}
	if got, _ := os.ReadFile(filepath.Join(out, "animation.mp4")); string(got) != "keep" {
		t.Fatalf("movie was changed to %q", got)
	}
}
