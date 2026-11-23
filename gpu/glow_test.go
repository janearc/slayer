package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// The test air: the night haze of the neon street's sets, an extinction
// of 0.0039 per metre, 300 m deep, even, scattering nine tenths.
const (
	testHaze       = 0.0039
	testHazeDepth  = 300.0
	testHazeAlbedo = 0.9
)

// testAirBox is a block of air a few hundred metres across from the
// ground to the top of the haze.
var testAirBox = maths.Box{Min: maths.V(-200, 0, -150),
	Max: maths.V(200, testHazeDepth, 150)}

// flatAir is a flat layer of the test haze over a box, the volume
// the session's flatAir makes: an aerosol over a level deck.
func flatAir(t *testing.T, box maths.Box) thing.Volume {
	t.Helper()
	weather := medium.NewWeather(testHaze, testHazeDepth, 1, 1, maths.Vec{})
	floor := func(float64, float64) float64 { return 0 }
	deck, err := medium.NewDeck(weather, box, floor, 0)
	if err != nil {
		t.Fatal(err)
	}
	aerosol, err := medium.NewAerosol(deck, medium.DefaultExponent)
	if err != nil {
		t.Fatal(err)
	}
	albedo := maths.Grey(testHazeAlbedo)
	return thing.NewVolume(aerosol, thing.Diffuse("night haze",
		albedo))
}

// meanOnly is a glow that says its mean and which air it lights and no
// more: one the kernel has no grid of to read.
type meanOnly struct{ lit medium.Medium }

// Mean is a dim even glow.
func (glow meanOnly) Mean(maths.Vec) maths.Light { return maths.Grey(1e-3) }

// Lights is the one air.
func (glow meanOnly) Lights(made medium.Medium) bool { return made == glow.lit }

// counted is a glow on a grid whose every cell says where it is: red
// is its place across, green up and blue along, so a cell read from the
// wrong place shows.
type counted struct {
	meanOnly
	low, high maths.Vec
	count     [3]int
}

// Cells is the test's box and counts.
func (glow counted) Cells() (low, high maths.Vec, count [3]int) {
	return glow.low, glow.high, glow.count
}

// Cell is the cell's place as a light.
func (glow counted) Cell(x, y, z int) maths.Light {
	return maths.RGB(float64(x), float64(y), float64(z))
}

// countedOver is a counted glow over the test box with cells two, three
// and four across, lighting one air.
func countedOver(lit medium.Medium) counted {
	return counted{meanOnly: meanOnly{lit: lit}, low: testAirBox.Min,
		high: testAirBox.Max, count: [3]int{2, 3, 4}}
}

// glowReasons is the reasons given against the world as a whole.
func glowReasons(world *thing.World) []string {
	var said []string
	for _, refused := range Refusals(world, 0) {
		if refused.Part == partWorld {
			said = append(said, refused.Reason)
		}
	}
	return said
}

// A glow the kernel cannot read on a grid is refused by name, by the
// drawing path and in the list, and is not drawn as no glow.
func TestAGlowWithNoGridIsRefusedByName(t *testing.T) {
	world := thing.NewWorld()
	air := world.AddVolume(flatAir(t, testAirBox))
	world.Glow = meanOnly{lit: air.Medium}
	const want = "the world's glow field of kind gpu.meanOnly"
	if said := glowReasons(world); len(said) != 1 || said[0] != want {
		t.Errorf("the world is refused for %q, want %q", said, want)
	}
	_, err := encode(world, maths.Subspace{}, nil, 0, 8, 8)
	var refused unsupported
	if !errors.As(err, &refused) || refused.what != want {
		t.Errorf("the encoder says %v, want %q", err, want)
	}
}

// A glow on a grid of no cells, or in a box a float cannot hold apart,
// is refused by name: the kernel would read nothing, or read the wrong
// cells.
func TestAGlowOnAGridItCannotHoldIsRefused(t *testing.T) {
	empty := countedOver(nil)
	empty.count = [3]int{4, 0, 4}
	infinite := countedOver(nil)
	infinite.low.X = math.Inf(-1)
	flat := countedOver(nil)
	flat.high.Y = flat.low.Y
	far := countedOver(nil)
	far.low.Z, far.high.Z = 1e12, 1e12+1
	for _, glow := range []counted{empty, infinite, flat, far} {
		refused, is := glowRefused(glow)
		if !is || !strings.HasPrefix(refused.what, "the world's glow field") {
			t.Errorf("a glow over %v to %v with %v cells is not refused: %q",
				glow.low, glow.high, glow.count, refused.what)
		}
	}
}

// A world with no glow, or a glow on a grid, is not refused for it.
func TestAGlowOnAGridIsNotRefused(t *testing.T) {
	if _, is := glowRefused(nil); is {
		t.Error("a world with no glow is refused for its glow")
	}
	world := thing.NewWorld()
	air := world.AddVolume(flatAir(t, testAirBox))
	world.Glow = countedOver(air.Medium)
	if said := glowReasons(world); len(said) != 0 {
		t.Errorf("a world with a glow on a grid is refused: %q", said)
	}
}

// The glow's cells go into the boxes x fastest, then y, then z, the order
// boxed_cell reads, and the header names its box, counts and first.
func TestTheGlowIsPackedInTheOrderTheKernelReads(t *testing.T) {
	world := thing.NewWorld()
	air := world.AddVolume(flatAir(t, testAirBox))
	glow := countedOver(air.Medium)
	world.Glow = glow
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.glow == nil {
		t.Fatal("the glow was not packed")
	}
	if encoded.glow.cells != glow.count || encoded.glow.low != glow.low ||
		encoded.glow.high != glow.high {
		t.Errorf("packed %+v, the glow is %v to %v of %v", encoded.glow,
			glow.low, glow.high, glow.count)
	}
	for z := range glow.count[2] {
		for y := range glow.count[1] {
			for x := range glow.count[0] {
				cell := (z*glow.count[1]+y)*glow.count[0] + x
				at := encoded.glow.first + cell*lightNumbers
				got := boxFloats(encoded, at, lightNumbers)
				if got[0] != float32(x) || got[1] != float32(y) ||
					got[2] != float32(z) {
					t.Fatalf("cell %d %d %d holds %v", x, y, z, got)
				}
			}
		}
	}
	grid := headerInts(encoded, "glow_grid")
	if grid != [4]int32{2, 3, 4, int32(encoded.glow.first)} {
		t.Errorf("the header's glow grid is %v", grid)
	}
}

// A world with no glow says so in the header: no cells packed, and a
// first number of minus one, which the kernel reads as no glow.
func TestAWorldWithNoGlowHasNoneInTheHeader(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(flatAir(t, testAirBox))
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.glow != nil {
		t.Errorf("a glow was packed for a world with none: %+v", encoded.glow)
	}
	if grid := headerInts(encoded, "glow_grid"); grid[3] != -1 {
		t.Errorf("the header's glow grid is %v, want no first number", grid)
	}
	if word := airWord(encoded, 0, glowWord); word != 0 {
		t.Errorf("the air is marked as lit by a glow there is not: %d", word)
	}
}

// Only the air the glow is of is marked as lit by it: another layer of
// the same haze beside it takes nothing from it, as in trace.
func TestOnlyTheAirTheGlowIsOfIsMarked(t *testing.T) {
	world := thing.NewWorld()
	other := world.AddVolume(flatAir(t, testAirBox))
	air := world.AddVolume(flatAir(t, testAirBox))
	world.Glow = countedOver(air.Medium)
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if word := airWord(encoded, 1, glowWord); word != glowOn {
		t.Errorf("the air the glow is of is not marked: %d", word)
	}
	if word := airWord(encoded, 0, glowWord); word != 0 {
		t.Errorf("the other air %p is marked: %d", other.Medium, word)
	}
}

// boxFloats is so many floats of the boxes from the first.
func boxFloats(encoded *scene, first, many int) []float32 {
	got := make([]float32, many)
	for index := range got {
		at := (first + index) * wordBytes
		got[index] = math.Float32frombits(
			binary.LittleEndian.Uint32(encoded.boxes.bytes[at:]))
	}
	return got
}

// headerInts is the four ints of a field of the header.
func headerInts(encoded *scene, name string) [4]int32 {
	var got [4]int32
	start := fieldOffset(headerStruct, name)
	for part := range got {
		got[part] = int32(binary.LittleEndian.Uint32(
			encoded.header.bytes[start+part*wordBytes:]))
	}
	return got
}

// airWord is one word of the air field of a volume, by its place.
func airWord(encoded *scene, volume, word int) uint32 {
	start := volume*volumeStruct.size() + fieldOffset(volumeStruct, "air") +
		word*wordBytes
	return binary.LittleEndian.Uint32(encoded.volumes.bytes[start:])
}
