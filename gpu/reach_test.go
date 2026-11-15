//go:build draw && darwin

package gpu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// the fixture's picture and its tolerances.
const (
	// reachWidth, reachHeight and reachSide are the fixture's picture:
	// small, two by two samples a pixel. Picked 2026-11-12.
	reachWidth, reachHeight, reachSide = 96, 64, 2
	// reachBudget is the most the fast picture's light may differ from
	// the exact one's, summed over the picture, as a share of the exact
	// picture's light. Picked 2026-11-12 as the cached smoke grid's own
	// two percent (doc/smoke-light-grid-study.md), looser than exact.
	reachBudget = 0.02
	// reachWorst is the most any one channel of any pixel may differ,
	// as a share of the exact picture's brightest. Picked 2026-11-12
	// from the measured fixture, with room: see the test's log.
	reachWorst = 0.05
)

// smokeUnderLamps is the fire's smoke column over its floor, lit by a
// lamp of each shape the store keeps: a point, a distant one and a
// narrow beam through the column.
func smokeUnderLamps(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	world, eye := fire(t)
	world.Volumes = world.Volumes[1:]
	world.Lamps = nil
	world.Lamp(emit.Lit{At: maths.V(4, 8, 4), Strength: maths.RGB(40, 10, 2)})
	far, err := emit.NewDistant(maths.V(-1, 2, 1), maths.RGB(0.2, 0.3, 0.5))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(far)
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(-6, 5, 0.2),
		Toward: maths.V(1, 0.1, 0), Strength: maths.RGB(1, 6, 1)},
		measure.Metres(0.15), measure.Metres(14), measure.Degrees(0))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(beam)
	return world, eye
}

// apart is how far two pictures' light is apart: summed over the first's
// light, and the worst channel over the first's brightest.
func apart(t *testing.T, exact, fast *frame.Frame) (summed, worst float64) {
	t.Helper()
	energy, sum, peak, most := 0.0, 0.0, 0.0, 0.0
	for index, want := range exact.Pix {
		got := fast.Pix[index]
		wants := []float32{want.R, want.G, want.B}
		for channel, value := range []float32{got.R, got.G, got.B} {
			if !(value >= 0) || math.IsInf(float64(value), 0) {
				t.Fatal("the fast picture holds a number that is no light")
			}
			gap := math.Abs(float64(wants[channel] - value))
			energy += math.Abs(float64(wants[channel]))
			sum += gap
			peak = max(peak, float64(wants[channel]))
			most = max(most, gap)
		}
	}
	if !(energy > 0 && peak > 0) {
		t.Fatal("the exact picture is dark; the fixture shows nothing")
	}
	return sum / energy, most / peak
}

// drawnFastAt is the fixture drawn with a lamp store of so many cells.
func drawnFastAt(t *testing.T, world *thing.World, eye maths.Subspace,
	cells int) *frame.Frame {
	t.Helper()
	picture, _, err := sampledLightingGrid(world, nil, eye,
		film.Pinhole{Wide: 50 * maths.Degree}, 0, reachWidth, reachHeight,
		reachSide, fastTraversal, cells)
	mustDevice(t, err)
	return picture
}

// TestFastLightTracksExactLighting holds the lamp store to its budget
// against the exact march, on a medium lit by a lamp of each shape, and
// says what the store cost.
func TestFastLightTracksExactLighting(t *testing.T) {
	world, eye := smokeUnderLamps(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	exact, direct, err := SampledAccelerated(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	fast, spent, err := SampledFast(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	summed, worst := apart(t, exact, fast)
	t.Logf("summed-relative %.9g, worst/peak %.9g; exact trace %v, "+
		"fast trace %v, store %v, %d bytes", summed, worst,
		direct.TraceGPU, spent.TraceGPU, spent.LightingGPU,
		spent.LampStoreBytes)
	if summed > reachBudget || worst > reachWorst {
		t.Fatalf("fast light is %.6f of the picture's light from exact "+
			"(budget %v), worst channel %.6f of its peak (budget %v)",
			summed, reachBudget, worst, reachWorst)
	}
	if spent.LampStoreBytes <= 0 || spent.LightingGPU <= 0 ||
		spent.LightingWall < spent.LightingGPU {
		t.Fatalf("the store was not accounted for: %+v", spent)
	}
	if direct.LampStoreBytes != 0 || direct.LightingGPU != 0 {
		t.Fatalf("the exact picture says it kept a store: %+v", direct)
	}
}

// TestFastLightIsNotTheExactPictureAndMovesWithItsLamps is the near
// miss: the fast picture differs from the exact one somewhere, so the
// budget is of a real approximation, and a lamp moved moves the smoke's
// light, so the store is the frame's own and not a kept one.
func TestFastLightIsNotTheExactPictureAndMovesWithItsLamps(t *testing.T) {
	world, eye := smokeUnderLamps(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	exact, _, err := SampledAccelerated(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	fast, _, err := SampledFast(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	if summed, _ := apart(t, exact, fast); summed == 0 {
		t.Fatal("fast light drew the exact picture bit for bit: the " +
			"store was not read")
	}
	world.Lamps[0] = emit.Lit{At: maths.V(-5, 3, 4),
		Strength: maths.RGB(40, 10, 2)}
	moved, _, err := SampledFast(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	if summed, _ := apart(t, fast, moved); summed == 0 {
		t.Fatal("a moved lamp left the fast picture as it was")
	}
	after, _, err := SampledAccelerated(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	if summed, _ := apart(t, after, moved); summed > reachBudget {
		t.Fatalf("after the lamp moved fast light is %.6f from exact",
			summed)
	}
}

// TestFastLightComesTogetherAsItsCellsGrow draws the fixture with a
// store of 32, 64 and 128 cells: the production 128 is within the
// budget and closer than 32, as a store that tracks the exact march is
// and a mistake would not be.
func TestFastLightComesTogetherAsItsCellsGrow(t *testing.T) {
	world, eye := smokeUnderLamps(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	exact, _, err := SampledAccelerated(world, eye, lens, 0, reachWidth,
		reachHeight, reachSide)
	mustDevice(t, err)
	gaps := map[int]float64{}
	for _, cells := range []int{32, 64, reachCells} {
		summed, worst := apart(t, exact, drawnFastAt(t, world, eye, cells))
		t.Logf("%d cells: summed-relative %.9g, worst/peak %.9g", cells,
			summed, worst)
		gaps[cells] = summed
	}
	if gaps[reachCells] > reachBudget || gaps[reachCells] >= gaps[32] {
		t.Fatalf("the store does not come together: %v", gaps)
	}
}

// lightingRow is a volume's lighting field as the kernel reads it.
func lightingRow(encoded *scene, volume int) [4]int32 {
	at := volume*volumeStruct.size() + fieldOffset(volumeStruct, "lighting")
	var row [4]int32
	for part := range row {
		row[part] = int32(binary.LittleEndian.Uint32(
			encoded.volumes.bytes[at+part*wordBytes:]))
	}
	return row
}

// headerLighting is the header's lighting field as the kernel reads it.
func headerLighting(encoded *scene) [4]int32 {
	at := fieldOffset(headerStruct, "lighting")
	var row [4]int32
	for part := range row {
		row[part] = int32(binary.LittleEndian.Uint32(
			encoded.header.bytes[at+part*wordBytes:]))
	}
	return row
}

// TestReserveReachKeepsAGridForBoxedMediaAndNoOther reserves a store
// over a puff, the smoke column and a boxed gas: only the column's row
// gets a grid; the boxes grow by exactly the store; nothing before it
// changes; and the header says nothing until a bake has filled it.
func TestReserveReachKeepsAGridForBoxedMediaAndNoOther(t *testing.T) {
	world, eye := smokeUnderLamps(t)
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 0.1}})
	world.AddVolume(thing.Volume{Medium: gas.Of{
		Shape: world.Volumes[0].Medium, Unit: 1,
		In: gas.Air()}})
	encoded, err := encode(world, eye, film.Pinhole{}, 0, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.reached) != 1 || encoded.reach != nil {
		t.Fatalf("encoding noted %d media and made a store unasked: %v",
			len(encoded.reached), encoded.reach)
	}
	volumes := bytes.Clone(encoded.volumes.bytes)
	boxes, header := bytes.Clone(encoded.boxes.bytes),
		bytes.Clone(encoded.header.bytes)
	if err = encoded.reserveReachAt(4); err != nil {
		t.Fatal(err)
	}
	store := encoded.reach
	if len(store.wide) != 2 || len(store.narrow) != 1 ||
		store.mask() != 0b011 {
		t.Fatalf("lamps read as wide %v, narrow %v", store.wide,
			store.narrow)
	}
	cells := store.grids[0].grid.count()
	grown := (cells*2 + reachBeamCells*3) * reachChannels * wordBytes
	if len(encoded.boxes.bytes) != len(boxes)+grown ||
		store.bytes(3) != int64(grown) ||
		!bytes.Equal(encoded.boxes.bytes[:len(boxes)], boxes) {
		t.Fatalf("the boxes grew by %d, want %d, or changed before it",
			len(encoded.boxes.bytes)-len(boxes), grown)
	}
	if !bytes.Equal(encoded.header.bytes, header) {
		t.Fatal("reserving wrote the header before any bake")
	}
	want := [4]int32{int32(store.grids[0].grid.cells[0]),
		int32(store.grids[0].grid.cells[1]),
		int32(store.grids[0].grid.cells[2]), int32(len(boxes) / wordBytes)}
	if got := lightingRow(encoded, 0); got != want {
		t.Fatalf("the column's lighting row is %v, want %v", got, want)
	}
	for _, other := range []int{1, 2} {
		if got := lightingRow(encoded, other); got[3] != -1 {
			t.Fatalf("volume %d, not a boxed medium that scatters "+
				"evenly, was given a grid: %v", other, got)
		}
	}
	stride := volumeStruct.size()
	if !bytes.Equal(encoded.volumes.bytes[stride:], volumes[stride:]) {
		t.Fatal("reserving changed another volume's row")
	}
}

// TestReachHeaderSaysTheStoreOnlyWhenItIsToBeRead writes the header's
// lighting field for a bake and for the trace, and reads it back.
func TestReachHeaderSaysTheStoreOnlyWhenItIsToBeRead(t *testing.T) {
	world, eye := smokeUnderLamps(t)
	encoded, err := encode(world, eye, film.Pinhole{}, 0, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got := headerLighting(encoded); got != [4]int32{} {
		t.Fatalf("a header with no store says %v", got)
	}
	if err = encoded.reserveReachAt(4); err != nil {
		t.Fatal(err)
	}
	encoded.writeReachHeader(5, 2, false)
	if got := headerLighting(encoded); got != [4]int32{5, 2, 0, 0} {
		t.Fatalf("a bake's header says %v", got)
	}
	encoded.shareSkyFor(fastTraversal, 3)
	encoded.writeReachHeader(5, 2, true)
	want := [4]int32{0, 0, 0b011, int32(encoded.reach.tables + 1)}
	if got := headerLighting(encoded); got != want {
		t.Fatalf("the trace's header says %v, want %v", got, want)
	}
	encoded.shareSkyFor(fasterTraversal, 4)
	encoded.writeReachHeader(5, 2, true)
	want[0] = 4
	if got := headerLighting(encoded); got != want {
		t.Fatalf("the header with the sky shared says %v, want %v", got,
			want)
	}
	// anti: at three along a side the looks are not dealt evenly, and the
	// header says nothing is shared
	encoded.reach.shared = 0
	encoded.shareSkyFor(fasterTraversal, 3)
	encoded.writeReachHeader(5, 2, true)
	want[0] = 0
	if got := headerLighting(encoded); got != want {
		t.Fatalf("the header at three a side says %v, want %v", got, want)
	}
}

// TestFastLightRefusesMoreLampsThanAPointAsks is the refusal, by name,
// and its near miss: exactly as many lamps as a point asks is taken.
func TestFastLightRefusesMoreLampsThanAPointAsks(t *testing.T) {
	for _, lamps := range []int{lampsAsked, lampsAsked + 1} {
		world, eye := smokeUnderLamps(t)
		world.Lamps = nil
		for index := range lamps {
			world.Lamp(emit.Lit{At: maths.V(float64(index), 8, 4),
				Strength: maths.RGB(1, 1, 1)})
		}
		encoded, err := encode(world, eye, film.Pinhole{}, 0, 4, 4)
		if err != nil {
			t.Fatal(err)
		}
		err = encoded.reserveReachAt(4)
		var refused errTooManyLamps
		if lamps <= lampsAsked {
			if err != nil || encoded.reach == nil {
				t.Fatalf("%d lamps were refused: %v", lamps, err)
			}
			continue
		}
		if !errors.As(err, &refused) || encoded.reach != nil ||
			!strings.Contains(err.Error(), "exact lighting") {
			t.Fatalf("%d lamps were not refused by name: %v", lamps, err)
		}
	}
}

// TestLampShapesReadEachLampAsTheKernelDoes reads a flood, a beam and a
// point lamp's rows: nothing, a table and a grid.
func TestLampShapesReadEachLampAsTheKernelDoes(t *testing.T) {
	beam, err := emit.NewBeam(emit.Lit{Toward: maths.V(0, 1, 0),
		Strength: maths.RGB(1, 1, 1)}, measure.Metres(1),
		measure.Metres(10), measure.Degrees(0))
	if err != nil {
		t.Fatal(err)
	}
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: []thing.Lamp{
		film.Flood{Strength: maths.RGB(1, 1, 1)}, beam,
		emit.Lit{At: maths.V(1, 2, 3), Strength: maths.RGB(1, 1, 1)}}})
	wide, narrow := encoded.lampShapes()
	if len(wide) != 1 || wide[0] != 2 || len(narrow) != 1 || narrow[0] != 1 {
		t.Fatalf("lamps read as wide %v, narrow %v", wide, narrow)
	}
}

// writeLampStore writes fast light's words for the layout check, over a
// header and volumes already written: the second volume's grid, and the
// header's lighting as the trace reads it. It says what the kernel is
// to read of each.
func writeLampStore(encoded *scene) []reading {
	encoded.writeReachRow(reachGrid{index: 1,
		grid: boxed{cells: [3]int{5, 6, 7}}, first: 91})
	encoded.reach = &reachStore{wide: []int{0, 2}, tables: 76, shared: 4}
	encoded.writeReachHeader(3, 1, true)
	table := fields("volume.lighting", xyzw, 5, 6, 7, 91)
	return append(table, fields("header->lighting", xyzw, 4, 0, 0b101,
		77)...)
}
