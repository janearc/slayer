package titan

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/slayer/titan/mocks"
	"go.uber.org/mock/gomock"
)

// the numbers the sphere map's tests stand on, each a plain test value.
const (
	testRow       = 4641.0589 // defined: Magellan's row spacing in metres
	testColumn    = 4000.0    // picked: a column spacing smaller than the row's
	testHeight    = 1234.5    // picked: a height to pass through
	testLongitude = 303.0     // defined: Venera 13's longitude, degrees east
	testLatitude  = -7.5      // defined: Venera 13's latitude, degrees north
)

// sphereOf is a mock raster that says it is an equirectangular sphere and
// gives the row spacing at the equator, as NewSphereMap asks.
func sphereOf(t *testing.T) *mocks.MockSphere {
	t.Helper()
	raster := mocks.NewMockSphere(gomock.NewController(t))
	raster.EXPECT().EquirectangularSphere().Return(6051000.0, true)
	raster.EXPECT().GroundSpacing(0.0).Return(testRow, testRow, nil)
	return raster
}

// venera is the place the tests ask about.
func venera(t *testing.T) mesh.Point {
	t.Helper()
	at, err := mesh.NewPoint(testLongitude, testLatitude)
	if err != nil {
		t.Fatal(err)
	}
	return *at
}

// A known height passes through, asked for by longitude then latitude.
func TestSphereMapHeightPasses(t *testing.T) {
	raster := sphereOf(t)
	raster.EXPECT().EquirectangularElevation(testLongitude, testLatitude).
		Return(testHeight, true, nil)
	from, err := NewSphereMap(raster, "key")
	if err != nil {
		t.Fatal(err)
	}
	height, known, err := from.Height(venera(t))
	if err != nil || !known || height != testHeight {
		t.Fatalf("got %g %v %v", height, known, err)
	}
}

// Anti: where the raster has no height, the map has none, and says so;
// it does not say nought is known.
func TestSphereMapAbsentStaysAbsent(t *testing.T) {
	raster := sphereOf(t)
	raster.EXPECT().EquirectangularElevation(testLongitude, testLatitude).
		Return(0.0, false, nil)
	from, err := NewSphereMap(raster, "key")
	if err != nil {
		t.Fatal(err)
	}
	if _, known, err := from.Height(venera(t)); known || err != nil {
		t.Fatalf("absent came back known %v, error %v", known, err)
	}
}

// An error from the raster comes out as it went in.
func TestSphereMapErrorPasses(t *testing.T) {
	broken := errors.New("a tile would not decode")
	raster := sphereOf(t)
	raster.EXPECT().EquirectangularElevation(testLongitude, testLatitude).
		Return(0.0, false, broken)
	from, err := NewSphereMap(raster, "key")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := from.Height(venera(t)); !errors.Is(err, broken) {
		t.Fatalf("got %v", err)
	}
}

// Anti: a height that came with an error is not read as measured.
func TestSphereMapErrorIsNotKnown(t *testing.T) {
	raster := sphereOf(t)
	raster.EXPECT().EquirectangularElevation(testLongitude, testLatitude).
		Return(0.0, false, errors.New("closed"))
	from, err := NewSphereMap(raster, "key")
	if err != nil {
		t.Fatal(err)
	}
	if _, known, _ := from.Height(venera(t)); known {
		t.Fatal("a failed read was called known")
	}
}

// Finest is the larger of the two spacings, whichever way round.
func TestSphereMapFinestIsLarger(t *testing.T) {
	for _, spacing := range [][2]float64{{testColumn, testRow},
		{testRow, testColumn}} {
		raster := sphereOf(t)
		raster.EXPECT().GroundSpacing(testLatitude).
			Return(spacing[0], spacing[1], nil)
		from, err := NewSphereMap(raster, "key")
		if err != nil {
			t.Fatal(err)
		}
		if got := from.Finest(venera(t)); got != testRow {
			t.Fatalf("spacings %v gave finest %g", spacing, got)
		}
	}
}

// Anti: Finest is never the smaller spacing.
func TestSphereMapFinestIsNotSmaller(t *testing.T) {
	raster := sphereOf(t)
	raster.EXPECT().GroundSpacing(testLatitude).Return(testColumn, testRow, nil)
	from, err := NewSphereMap(raster, "key")
	if err != nil {
		t.Fatal(err)
	}
	if from.Finest(venera(t)) == testColumn {
		t.Fatal("finest took the smaller spacing")
	}
}

// Where the raster cannot give a spacing, the map has nothing to say.
func TestSphereMapFinestWithoutSpacing(t *testing.T) {
	raster := sphereOf(t)
	raster.EXPECT().GroundSpacing(testLatitude).
		Return(0.0, 0.0, errors.New("no spacing"))
	from, err := NewSphereMap(raster, "key")
	if err != nil {
		t.Fatal(err)
	}
	if got := from.Finest(venera(t)); !math.IsInf(got, 1) {
		t.Fatalf("got %g", got)
	}
}

// A raster that is not an equirectangular sphere is refused when the
// provider is made, naming the source.
func TestSphereMapRefusesOtherRasters(t *testing.T) {
	raster := mocks.NewMockSphere(gomock.NewController(t))
	raster.EXPECT().EquirectangularSphere().Return(0.0, false)
	_, err := NewSphereMap(raster, "some-key")
	if err == nil || !strings.Contains(err.Error(), "some-key") {
		t.Fatalf("got %v", err)
	}
}

// Anti: a raster that is a sphere is accepted, and a missing one refused.
func TestSphereMapAcceptsSpheresAndRefusesNone(t *testing.T) {
	if _, err := NewSphereMap(sphereOf(t), "key"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSphereMap(nil, "key"); err == nil {
		t.Fatal("a map with no raster was made")
	}
}

// Source names the key and the spacing.
func TestSphereMapSourceNamesKeyAndSpacing(t *testing.T) {
	from, err := NewSphereMap(sphereOf(t), "usgs-magellan-venus-topography")
	if err != nil {
		t.Fatal(err)
	}
	said := from.Source()
	if !strings.Contains(said, "usgs-magellan-venus-topography") ||
		!strings.Contains(said, "4641") {
		t.Fatalf("got %q", said)
	}
}

// Anti: a spacing that is not the file's is not in the sentence.
func TestSphereMapSourceNamesNoOtherSpacing(t *testing.T) {
	from, err := NewSphereMap(sphereOf(t), "key")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(from.Source(), "4000") {
		t.Fatalf("got %q", from.Source())
	}
}
