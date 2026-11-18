package titan

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
)

// TestKindsSayWhatTheyAre checks the word for each kind of ground.
func TestKindsSayWhatTheyAre(t *testing.T) {
	if Measured.String() != "measured" || Generated.String() != "generated" {
		t.Errorf("kinds are %q and %q", Measured.String(), Generated.String())
	}
}

// TestADirectionThatIsNotAPlaceStopsTheProgram checks that a direction with
// no number in it, which titan never makes, is a fault and not a refusal.
func TestADirectionThatIsNotAPlaceStopsTheProgram(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a direction of NaN gave a place")
		}
	}()
	vector{x: math.NaN(), y: math.NaN(), z: math.NaN()}.point()
}

// failing is a map that gives the same flat height everywhere until its
// n-th reading, which it refuses.
type failing struct {
	inner *chart
	at    int
	calls int
	err   error
}

// Source is the inner map's.
func (f *failing) Source() string { return f.inner.Source() }

// Finest is the inner map's.
func (f *failing) Finest(at mesh.Point) float64 { return f.inner.Finest(at) }

// Height is the inner map's until the chosen reading.
func (f *failing) Height(at mesh.Point) (float64, bool, error) {
	f.calls++
	if f.calls == f.at {
		return 0, false, f.err
	}
	return f.inner.Height(at)
}

// TestAMapThatFailsAtAnyReadingFailsTheGround checks that the error of every
// reading, whether of a corner, of the rings round it or of a deeper corner,
// reaches the caller unchanged.
func TestAMapThatFailsAtAnyReadingFailsTheGround(t *testing.T) {
	sentinel := errors.New("the map is gone")
	probe := &failing{inner: &chart{finest: 100, relief: 300, share: 0.6}}
	world := ground(t, probe, 1)
	at := spot(t, 12.3, 4.5)
	if _, err := world.At(at, 20); err != nil {
		t.Fatal(err)
	}
	total := probe.calls
	if total < 3*(1+2*around) {
		t.Fatalf("a clean run read the map %d times", total)
	}
	for n := 1; n <= total; n++ {
		flaky := &failing{inner: &chart{finest: 100, relief: 300, share: 0.6}, at: n, err: sentinel}
		if _, err := ground(t, flaky, 1).At(at, 20); !errors.Is(err, sentinel) {
			t.Fatalf("a map failing at reading %d of %d gave %v", n, total, err)
		}
	}
}

// islet is a flat map that is known only at a few places: every other place
// within a tenth of a degree of one of them is not.
type islet struct {
	places []mesh.Point
	finest float64
	// eastOnly makes only the places east of the known one unknown, so a
	// ring round it is known on one side.
	eastOnly bool
}

// Source names the map.
func (islet) Source() string { return "an islet" }

// Finest is the same everywhere.
func (m islet) Finest(mesh.Point) float64 { return m.finest }

// Height is zero at the known places and where no known place is near.
func (m islet) Height(at mesh.Point) (float64, bool, error) {
	for _, place := range m.places {
		near := math.Hypot(at.Lon()-place.Lon(), at.Lat()-place.Lat())
		if near > 0 && near < 0.1 && (!m.eastOnly || at.Lon() > place.Lon()) {
			return 0, false, nil
		}
	}
	return 0, true, nil
}

// cellCorners is the places at the corners of the face a point falls in.
func cellCorners(t *testing.T, at *mesh.Point) []mesh.Point {
	t.Helper()
	cell, err := mesh.Locate(at, 0)
	if err != nil {
		t.Fatal(err)
	}
	a, b, c := cell.Corners()
	return []mesh.Point{a, b, c}
}

// TestRoughnessIsNoneWhereTheMapKnowsNothingRound checks the roughness and
// slope read round a corner whose ring is wholly unknown, half unknown and
// wholly known.
func TestRoughnessIsNoneWhereTheMapKnowsNothingRound(t *testing.T) {
	at := spot(t, 12.3, 4.5)
	corners := cellCorners(t, at)
	for name, want := range map[string]struct {
		eastOnly bool
		unknown  bool
	}{"wholly unknown": {false, true}, "half unknown": {true, true}, "wholly known": {false, false}} {
		places := corners[1:2]
		if !want.unknown {
			places = nil
		}
		world := ground(t, islet{places: places, finest: 100, eastOnly: want.eastOnly}, 1)
		here := vectorOf(corners[1])
		differs, slope, err := world.differs(here, 0, 100)
		if err != nil || differs != 0 || slope != (vector{}) {
			t.Errorf("%s: roughness %v, slope %v, %v; want none of either on flat ground", name, differs, slope, err)
		}
	}
}

// slanted is a map whose height is three metres a degree of latitude.
type slanted struct{}

// Source names the map.
func (slanted) Source() string { return "a slanted map" }

// Finest is a hundred metres everywhere.
func (slanted) Finest(mesh.Point) float64 { return 100 }

// Height is 3 metres for each degree north of the equator.
func (slanted) Height(at mesh.Point) (float64, bool, error) { return 3 * at.Lat(), true, nil }

// TestSlopeIsTrustedOnlyWhereTheWholeRingIsKnown checks that a plane's
// slope is read round a corner with the whole ring known, and dropped with
// part of it missing.
func TestSlopeIsTrustedOnlyWhereTheWholeRingIsKnown(t *testing.T) {
	at := spot(t, 12.3, 4.5)
	corners := cellCorners(t, at)
	whole := ground(t, slanted{}, 1)
	here := vectorOf(corners[1])
	height, _, _ := slanted{}.Height(corners[1])
	_, slope, err := whole.differs(here, height, 100)
	if err != nil || math.Sqrt(slope.dot(slope)) == 0 {
		t.Fatalf("a plane has slope %v, %v", slope, err)
	}
	// 3 metres a degree is 3 / (earth * pi / 180) metres a metre.
	want := 3 / (earth * math.Pi / 180)
	if got := math.Sqrt(slope.dot(slope)); math.Abs(got-want) > want*1e-3 {
		t.Errorf("the slope is %v a metre, want about %v", got, want)
	}
	cut := ground(t, partial{slanted{}, corners[1]}, 1)
	_, none, err := cut.differs(here, height, 100)
	if err != nil || none != (vector{}) {
		t.Errorf("a ring half known gave slope %v, %v", none, err)
	}
}

// partial is a slanted map with no height east of one place, within a tenth
// of a degree of it.
type partial struct {
	slanted
	place mesh.Point
}

// Height is unknown just east of the place.
func (m partial) Height(at mesh.Point) (float64, bool, error) {
	if near := math.Hypot(at.Lon()-m.place.Lon(), at.Lat()-m.place.Lat()); near > 0 && near < 0.1 && at.Lon() > m.place.Lon() {
		return 0, false, nil
	}
	return m.slanted.Height(at)
}

// patchy is a flat map whose samples are no distance apart at the corners of
// one face and a million kilometres apart anywhere else.
type patchy struct{ corners []mesh.Point }

// Source names the map.
func (patchy) Source() string { return "a patchy map" }

// Height is zero everywhere.
func (patchy) Height(mesh.Point) (float64, bool, error) { return 0, true, nil }

// Finest is none at the corners, and enormous elsewhere.
func (m patchy) Finest(at mesh.Point) float64 {
	for _, corner := range m.corners {
		if math.Hypot(at.Lon()-corner.Lon(), at.Lat()-corner.Lat()) < 1e-6 {
			return 0
		}
	}
	return 1e9
}

// TestGroundMadeFromCornersWithNoSpacingAddsNothing checks that made ground
// between corners the map has no spacing for is made, flat, and is not
// given noise or spread that has nothing to scale by.
func TestGroundMadeFromCornersWithNoSpacingAddsNothing(t *testing.T) {
	at := spot(t, 12.3, 4.5)
	world := ground(t, patchy{cellCorners(t, at)}, 1)
	got, err := world.At(at, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind() != Generated || got.Height() != 0 || got.Made() != 0 {
		t.Errorf("ground made from no spacing: %v, %v m high, %v m made", got.Kind(), got.Height(), got.Made())
	}
}

// TestTheKeptCornersAreClearedWhenTheyReachTheirLimit checks that the cache
// of measured corners is emptied when it is full, and then keeps the corner
// just read.
func TestTheKeptCornersAreClearedWhenTheyReachTheirLimit(t *testing.T) {
	world := ground(t, &chart{finest: 100, relief: 300, share: 0.6}, 1)
	for name := range uint64(most) {
		world.kept[name+1<<40] = read{}
	}
	if world.Kept() != most {
		t.Fatalf("the cache holds %d corners", world.Kept())
	}
	if _, err := world.read(vectorOf(*spot(t, 12.3, 4.5)), 7); err != nil {
		t.Fatal(err)
	}
	if world.Kept() != 1 {
		t.Errorf("the cache holds %d corners after a full cache was read", world.Kept())
	}
}
