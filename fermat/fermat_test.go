package fermat

import (
	"crypto/sha256"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// pale is a sky, brighter overhead than at the horizon.
type pale struct{}

// Light is the sky's light along a way.
func (pale) Light(toward maths.Vec) maths.Light {
	return maths.Grey(0.2 + 0.5*math.Max(0, toward.Y))
}

// scene is a glossy ball on a grey floor under a sun and a pale sky, and
// how it is seen.
type scene struct {
	world *thing.World
	eye   maths.Subspace
	lens  film.Sensor
}

// lit is the scene the tests prove.
func lit() scene {
	world := thing.NewWorld()
	world.Sky = pale{}
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "floor", Colour: maths.Grey(0.5)}})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "ball", Colour: maths.Grey(0.6),
			Gloss: 1},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)})})
	world.Lamp(thing.Sun(maths.V(-0.4, 1, 0.5).Unit(), maths.Grey(1)))
	from := maths.V(0, 2, 6)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 1, 0))}
	return scene{world: world, eye: eye, lens: film.Pinhole{}}
}

// kept renders the scene at width by height and rays, keeps its raw in
// a fresh directory, and says where, with the replay that answers for it.
func kept(t *testing.T, seen scene, width, height,
	rays int) (string, Replay) {
	t.Helper()
	drawn, err := trace.Sampled(seen.world, seen.eye, seen.lens, 0, width,
		height, rays)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "render.pfm")
	if err := raw.Write(path, drawn); err != nil {
		t.Fatal(err)
	}
	return path, func(x, y int) (frame.Pixel, []trace.Step, error) {
		return trace.Replay(seen.world, seen.eye, seen.lens, 0, width,
			height, rays, x, y)
	}
}

// the picks are the raw's to make: the same bytes pick the same pixels,
// each once and inside the picture; other bytes pick others; a picture
// no bigger than the picks gives every pixel, and none gives none.
func TestPicksAreTheRawsOwn(t *testing.T) {
	sum := sha256.Sum256([]byte("a raw"))
	first, again := Picks(sum, 40, 30, Many), Picks(sum, 40, 30, Many)
	seen := map[[2]int]bool{}
	for i, at := range first {
		if at != again[i] || seen[at] || at[0] < 0 || at[0] >= 40 ||
			at[1] < 0 || at[1] >= 30 {
			t.Fatalf("pick %d: %v, again %v, seen %v", i, at, again[i],
				seen[at])
		}
		seen[at] = true
	}
	other := Picks(sha256.Sum256([]byte("another raw")), 40, 30, Many)
	if len(first) != Many || first[0] == other[0] && first[1] == other[1] {
		t.Errorf("%d picks; other bytes picked %v, %v first", len(first),
			other[0], other[1])
	}
	if every := Picks(sum, 5, 4, Many); len(every) != 20 {
		t.Errorf("a picture of 20 gave %d picks", len(every))
	}
	if Picks(sum, 0, 4, Many) != nil || Picks(sum, 5, 4, 0) != nil {
		t.Error("nothing to pick picked something")
	}
}

// a raw of a render holds: each of its 64 picked pixels made again is its light
// bit for bit, with the steps that made it.
func TestARenderHoldsToItsRaw(t *testing.T) {
	path, replay := kept(t, lit(), 32, 24, 2)
	proof, err := Prove(path, "the lit test scene", nil, Many, replay)
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Holds() || len(proof.Pixels) != Many {
		t.Fatalf("%s", proof)
	}
	for _, pixel := range proof.Pixels {
		if len(pixel.Steps) < 2*2*2 {
			t.Errorf("pixel %d, %d told %d steps", pixel.X, pixel.Y,
				len(pixel.Steps))
		}
	}
	if !strings.Contains(proof.String(), "holds: 64 of 64") {
		t.Errorf("said %q", proof)
	}
}

// a pixel of the raw changed by the least a float can change fails the
// proof at that pixel and nowhere else, and says what it kept and made.
func TestAChangedPixelFailsTheProofThere(t *testing.T) {
	path, replay := kept(t, lit(), 8, 8, 1)
	frame, err := raw.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	was := frame.At(2, 3)
	changed := was
	changed.G = math.Nextafter32(was.G, float32(math.Inf(1)))
	frame.Set(2, 3, changed)
	if err := raw.Write(path, frame); err != nil {
		t.Fatal(err)
	}
	proof, err := Prove(path, "the lit test scene, changed", nil, Many, replay)
	if err != nil {
		t.Fatal(err)
	}
	var failed []Pixel
	for _, pixel := range proof.Pixels {
		if !pixel.Holds() {
			failed = append(failed, pixel)
		}
	}
	if proof.Holds() || len(failed) != 1 || failed[0].X != 2 ||
		failed[0].Y != 3 {
		t.Fatalf("%s; failed %v", proof, failed)
	}
	if why := failed[0].why(); !strings.Contains(why, "the raw kept") ||
		!strings.Contains(why, "and it was made") {
		t.Errorf("why: %q", why)
	}
}

// a raw that is not there, or not a raw, is an error, not a proof.
func TestAProofNeedsARaw(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone.pfm")
	if _, err := Prove(missing, "", nil, Many, nil); err == nil {
		t.Error("a missing raw was proved")
	}
	junk := filepath.Join(t.TempDir(), "junk.pfm")
	os.WriteFile(junk, []byte("not a raw\n"), 0o644)
	if _, err := Prove(junk, "", nil, Many, nil); err == nil {
		t.Error("a file that is not a raw was proved")
	}
}

// the sheet is proof.md, naming the verdict, the raw's sum and how it was asked
// for, and a page of work for every pixel, step by step.
func TestTheSheetHasAPageForEveryPixel(t *testing.T) {
	path, replay := kept(t, lit(), 16, 12, 1)
	proof, err := Prove(path, "hillside -seed 3", nil, Many, replay)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "proof")
	if err := proof.Write(dir); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(dir, "proof.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, said := range []string{"holds: 64 of 64", proof.Sum,
		"hillside -seed 3", "1. pixel"} {
		if !strings.Contains(string(index), said) {
			t.Errorf("proof.md lacks %q", said)
		}
	}
	pages, _ := filepath.Glob(filepath.Join(dir, "pixel-*.md"))
	if len(pages) != Many {
		t.Fatalf("%d pages", len(pages))
	}
	first := proof.Pixels[0]
	work, _ := os.ReadFile(filepath.Join(dir, first.page()))
	for _, said := range []string{"holds: made again", "a ray from the lens",
		"the pixel:"} {
		if !strings.Contains(string(work), said) {
			t.Errorf("%s lacks %q:\n%s", first.page(), said, work)
		}
	}
}

// a picture is marked about each picked pixel, green where it holds and
// red where it fails, and the pixel itself is left as it was.
func TestTheMarksSayWhichHeld(t *testing.T) {
	proof := Proof{Pixels: []Pixel{{X: 4, Y: 4}, {X: 0, Y: 0,
		Err: os.ErrNotExist}}}
	picture := image.NewRGBA(image.Rect(0, 0, 10, 10))
	proof.Mark(picture)
	if green := picture.RGBAAt(7, 4); green.G != 220 || green.R != 0 {
		t.Errorf("a held pixel's mark is %v", green)
	}
	if red := picture.RGBAAt(3, 0); red.R != 255 || red.G != 0 {
		t.Errorf("a failed pixel's mark is %v", red)
	}
	if centre := picture.RGBAAt(4, 4); centre.A != 0 {
		t.Errorf("the held pixel itself was drawn on: %v", centre)
	}
}
