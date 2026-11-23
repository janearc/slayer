package session

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

// The glint sea's frame: its size in pixels, the sun's altitude in
// degrees and strength, and the eye's height and distance south of the
// sea's middle in metres.
const (
	glintFrameWidth, glintFrameHeight = 48, 32
	glintSunAltitude, glintSunLight   = 17.5, 10
	glintEyeHeight, glintEyeSouth     = 6, 20
)

// glintOceanText is a windy sea 64 m across at 1 m cells, its water a
// dark colour, with a glint block said under the ocean, or none for an
// empty glint.
func glintOceanText(t *testing.T, glint string) string {
	t.Helper()
	text := authoredOceanText("1")
	for _, change := range []struct{ was, now string }{
		{"extent: {value: 8, unit: m}", "extent: {value: 64, unit: m}"},
		{"samples: 9", "samples: 65"},
		{"wind: {value: 0, unit: m/s}", "wind: {value: 7, unit: m/s}"},
		{"fetch: {value: 0, unit: m}", "fetch: {value: 500000, unit: m}"},
		{"toward: {value: 0, unit: rad}", "toward: {value: 1.5708, unit: rad}"},
		{"clear: {through: 1", "colour: {srgb_linear: [0.01, 0.04, 0.06]}\n" +
			"        clear: {through: 1"},
	} {
		if !strings.Contains(text, change.was) {
			t.Fatalf("fixture lacks %q", change.was)
		}
		text = strings.Replace(text, change.was, change.now, 1)
	}
	if glint == "" {
		return text
	}
	// the ocean's source comes before the material's
	const said = "      source: eye\n"
	return strings.Replace(text, said, said+"      glint: "+glint+"\n", 1)
}

// glintOcean is the glint sea's set, read.
func glintOcean(t *testing.T, glint string) set.Set {
	t.Helper()
	source := glintOceanText(t, glint)
	read, err := set.Read("glint-ocean.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// glintOceanFrame draws the glint sea from south of its middle toward
// the sun's mirror point on a flat sea, under a sun low in the north,
// and returns the sha256 of every pixel's values and their summed
// luminance.
func glintOceanFrame(t *testing.T, glint string) (string, float64) {
	t.Helper()
	drawn := stagedWorld(glintOcean(t, glint), asked{set: "glint-ocean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	world := drawn.world
	up := glintSunAltitude * math.Pi / 180
	toward := maths.V(0, math.Sin(up), -math.Cos(up))
	sun, err := emit.NewDistant(toward,
		maths.Grey(glintSunLight))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(sun)
	from := maths.V(0, glintEyeHeight, glintEyeSouth)
	mirror := maths.V(0, 0, glintEyeSouth-glintEyeHeight/math.Tan(up))
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, mirror)}
	frame, err := trace.Render(world, eye, film.Pinhole{}, 0,
		glintFrameWidth, glintFrameHeight)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	var total float64
	for _, pixel := range frame.Pix {
		for _, part := range []float32{pixel.R, pixel.G, pixel.B,
			pixel.Cover, pixel.Depth} {
			var word [4]byte
			binary.LittleEndian.PutUint32(word[:], math.Float32bits(part))
			digest.Write(word[:])
		}
		total += pixel.Light().Luminance()
	}
	return hex.EncodeToString(digest.Sum(nil)), total
}

// A sea that says no glint wears none: its water has no satin and no
// slope variance, the two things a glint is drawn from, so nothing a
// glint adds reaches its picture. Nought is defined as no glint for both.
// The tests after this one are its anti-tests: a glint that is said gives
// one of the two a value and changes the frame.
func TestOceanWithoutGlintWearsNone(t *testing.T) {
	plain := stagedWorld(glintOcean(t, ""), asked{set: "plain.yaml"})
	if plain.err != nil {
		t.Fatal(plain.err)
	}
	water := plain.world.Things[0].Material
	if water.Satin != 0 || water.SlopeVariance != 0 {
		t.Fatalf("a sea that says no glint wears satin %g and slope "+
			"variance %g", water.Satin, water.SlopeVariance)
	}
}

func TestOceanArtisticGlintChangesThePicture(t *testing.T) {
	plain, plainLight := glintOceanFrame(t, "")
	glinted, glintLight := glintOceanFrame(t,
		"{model: artistic, roughness: 0.3}")
	if glinted == plain {
		t.Fatal("an artistic glint drew the same frame as no glint")
	}
	if !(glintLight > plainLight) {
		t.Fatalf("an artistic glint drew %g light, no more than %g without",
			glintLight, plainLight)
	}
}

func TestOceanArtisticGlintWearsItsOwnSatin(t *testing.T) {
	plain := stagedWorld(glintOcean(t, ""), asked{set: "plain.yaml"})
	glinted := stagedWorld(glintOcean(t, "{model: artistic, roughness: 0.2}"),
		asked{set: "glint.yaml"})
	if plain.err != nil || glinted.err != nil {
		t.Fatal(plain.err, glinted.err)
	}
	was, now := plain.world.Things[0].Material, glinted.world.Things[0].Material
	if was.Satin != 0 || now.Satin != 0.2 {
		t.Fatalf("satin %g without a glint and %g with one", was.Satin,
			now.Satin)
	}
	if now.Through != was.Through || now.Index != was.Index ||
		now.Gloss != was.Gloss || now.Colour != was.Colour ||
		now.Rank != was.Rank || now.ClearMedium != was.ClearMedium {
		t.Fatalf("the glint changed more than satin: %+v from %+v", now, was)
	}
}

// Anti-test for the sea that wears none: a cox-munk glint changes the frame, adds
// light, and draws a frame of its own, not the artistic one's.
func TestOceanCoxMunkGlintChangesThePicture(t *testing.T) {
	plain, plainLight := glintOceanFrame(t, "")
	derived, derivedLight := glintOceanFrame(t, "{model: cox-munk}")
	artistic, _ := glintOceanFrame(t, "{model: artistic, roughness: 0.3}")
	if derived == plain || derived == artistic {
		t.Fatal("a cox-munk glint drew the frame of no glint or of 0.3")
	}
	if !(derivedLight > plainLight) {
		t.Fatalf("a cox-munk glint drew %g light, no more than %g without",
			derivedLight, plainLight)
	}
}

func TestOceanCoxMunkGlintWearsTheUnresolvedSlope(t *testing.T) {
	read := glintOcean(t, "{model: cox-munk}")
	drawn := stagedWorld(read, asked{set: "cox-munk.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	said, _ := read.World()
	surface, _ := said.OceanSurface()
	wind, fetch, depth, toward := surface.Sea()
	budget, err := arid.GlintSlopes(oceantex.Sea{Wind: wind, Fetch: fetch,
		Depth: depth, Toward: toward}, surface.Extent(), surface.Samples())
	if err != nil {
		t.Fatal(err)
	}
	made := drawn.world.Things[0].Material
	if made.SlopeVariance != budget.Unresolved || made.Satin != 0 ||
		!(budget.Unresolved > 0) {
		t.Fatalf("slope variance %g, satin %g; budget %+v",
			made.SlopeVariance, made.Satin, budget)
	}
}

func TestOceanReceiptLabelsAnArtisticGlint(t *testing.T) {
	for _, each := range []struct {
		glint, want string
		has         bool
	}{
		{"", "glint=", false},
		{"{model: artistic, roughness: 0.2}",
			"glint=ARTISTIC roughness=0.2 (GGX alpha 0.04", true},
		{"{model: cox-munk}", "glint=cox-munk, Beckmann lobe: " +
			"mean-square slope Cox-Munk 1954 total=0.03884", true},
		{"{model: cox-munk}", "ARTISTIC", false},
	} {
		said, _ := glintOcean(t, each.glint).World()
		surface, _ := said.OceanSurface()
		report, err := oceanSpectrumEstimate(surface)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(report.line(), each.want); got != each.has {
			t.Errorf("glint %q: line %q says %q %t, want %t", each.glint,
				report.line(), each.want, got, each.has)
		}
	}
}
