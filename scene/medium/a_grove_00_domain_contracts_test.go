package medium

import (
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The tables the domain contracts read: weathers, canopies and points, each
// row by its name.
const (
	aGrove00WeatherFile = "testdata/a-grove-00-domain-contracts-weathers.csv"
	aGrove00TreeFile    = "testdata/a-grove-00-domain-contracts-trees.csv"
	aGrove00PointFile   = "testdata/a-grove-00-domain-contracts-points.csv"
	aGrove00TaperFile   = "testdata/a-grove-00-domain-contracts-tapers.csv"
)

// aGrove00Weather is the weather of a row of the table of weathers.
func aGrove00Weather(t *testing.T, name string) GroveWeather {
	t.Helper()
	row := readSheet(t, aGrove00WeatherFile).row(t, aGrove00WeatherFile, name)
	return GroveWeather{Extinction: cell(t, row, "extinction"),
		Depth: cell(t, row, "depth"), Coverage: cell(t, row, "coverage")}
}

// aGrove00Tree is the canopy of a row of the table of canopies.
func aGrove00Tree(t *testing.T, name string) Canopy {
	t.Helper()
	row := readSheet(t, aGrove00TreeFile).row(t, aGrove00TreeFile, name)
	return Canopy{At: maths.V(cell(t, row, "x"), cell(t, row, "y"),
		cell(t, row, "z")),
		Radius: cell(t, row, "radius"), Height: cell(t, row, "height")}
}

// aGrove00Point is the point of a row of the table of points.
func aGrove00Point(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, aGrove00PointFile).row(t, aGrove00PointFile, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// aGrove00Floor is ground at a half of a metre everywhere.
func aGrove00Floor(float64, float64) float64 { return halfway }

// aGrove00Result is what building a grove gave: the grove and the error.
type aGrove00Result struct {
	grove *Grove
	err   error
}

// aGrove00Build builds a grove and sends what came of it down done.
func aGrove00Build(done chan<- aGrove00Result, weather GroveWeather,
	trees []Canopy, floor func(float64, float64) float64) {
	grove, err := NewGrove(weather, trees, floor)
	done <- aGrove00Result{grove, err}
}

// aGrove00Case is a grove asked for under a name, and whether it is to be
// built.
type aGrove00Case struct {
	name    string
	weather GroveWeather
	trees   []Canopy
	floor   func(float64, float64) float64
	want    bool
}

// aGrove00Judge stops the test if what building a case gave is not what the
// case wants: a grove and no error, or neither, and not too many buckets.
func aGrove00Judge(t *testing.T, each aGrove00Case, got aGrove00Result) {
	t.Helper()
	perTree := pickedCount(t, "a-grove-00-domain-contracts-buckets-per-tree")
	if each.want && got.grove != nil &&
		len(got.grove.canopies) > perTree*len(each.trees) {
		t.Fatalf("%s: %d canopy buckets for %d trees", each.name,
			len(got.grove.canopies), len(each.trees))
	}
	if (got.err == nil) != each.want || (got.grove != nil) != each.want {
		t.Fatalf("%s: grove %v, error %v, valid %t", each.name, got.grove,
			got.err, each.want)
	}
}

// aGrove00Check builds a case, and stops the test if it is not built as the
// case wants, or is not built within a second.
func aGrove00Check(t *testing.T, each aGrove00Case) {
	t.Helper()
	done := make(chan aGrove00Result, unity)
	go aGrove00Build(done, each.weather, each.trees, each.floor)
	select {
	case got := <-done:
		aGrove00Judge(t, each, got)
	case <-time.After(time.Second):
		t.Fatalf("%s: canopy indexing did not finish within one second",
			each.name)
	}
}

// aGrove00Accepted checks the groves that are built: an ordinary one, a thin
// layer with a broad canopy, and one with no extinction.
func aGrove00Accepted(t *testing.T, weather GroveWeather, tree Canopy) {
	t.Helper()
	trees := []Canopy{tree}
	aGrove00Check(t, aGrove00Case{"positive", weather, trees, flatGround, true})
	thin := weather
	thin.Depth = picked(t, "a-grove-00-domain-contracts-thin-depth")
	broad := []Canopy{aGrove00Tree(t, "broad")}
	aGrove00Check(t, aGrove00Case{"thin layer with broad canopy", thin, broad,
		flatGround, true})
	clean := weather
	clean.Extinction = nought
	aGrove00Check(t, aGrove00Case{"zero extinction", clean, trees, flatGround,
		true})
	aGrove00Check(t, aGrove00Case{"nil ground", weather, trees, nil, false})
	aGrove00Check(t, aGrove00Case{"no canopies", weather, nil, flatGround,
		false})
}

// aGrove00SpoilWeather is the weather with the field named set to a value.
func aGrove00SpoilWeather(weather GroveWeather, field string,
	value float64) GroveWeather {
	switch field {
	case "extinction":
		weather.Extinction = value
	case "depth":
		weather.Depth = value
	case "coverage":
		weather.Coverage = value
	case "floor":
		weather.GroundFloor = value
	case "wind x":
		weather.Wind.X = value
	case "wind y":
		weather.Wind.Y = value
	case "wind z":
		weather.Wind.Z = value
	}
	return weather
}

// aGrove00SpoilCanopy is the canopy with the field named set to a value.
func aGrove00SpoilCanopy(canopy Canopy, field string, value float64) Canopy {
	switch field {
	case "radius":
		canopy.Radius = value
	case "height":
		canopy.Height = value
	case "x":
		canopy.At.X = value
	case "y":
		canopy.At.Y = value
	case "z":
		canopy.At.Z = value
	}
	return canopy
}

// aGrove00NotNumbers checks that a NaN and an infinity either way, in each
// field of the weather and the canopy, are refused.
func aGrove00NotNumbers(t *testing.T, weather GroveWeather, tree Canopy) {
	t.Helper()
	weatherFields := []string{"extinction", "depth", "coverage", "floor",
		"wind x", "wind y", "wind z"}
	canopyFields := []string{"radius", "height", "x", "y", "z"}
	for _, value := range []float64{math.NaN(), math.Inf(unity),
		math.Inf(-unity)} {
		for _, field := range weatherFields {
			spoiled := aGrove00SpoilWeather(weather, field, value)
			aGrove00Check(t, aGrove00Case{field, spoiled, []Canopy{tree},
				flatGround, false})
		}
		for _, field := range canopyFields {
			spoiled := aGrove00SpoilCanopy(tree, field, value)
			aGrove00Check(t, aGrove00Case{"canopy " + field, weather,
				[]Canopy{spoiled}, flatGround, false})
		}
	}
}

// aGrove00Refusal is a weather and a canopy a hair outside their domain.
type aGrove00Refusal struct {
	name    string
	weather GroveWeather
	tree    Canopy
}

// aGrove00Refusals is the weathers and canopies a hair outside their domain:
// below nought where a field may not be, above one where it may not be, a
// floor above the tree, and a canopy of no size.
func aGrove00Refusals(weather GroveWeather, tree Canopy) []aGrove00Refusal {
	below := -math.SmallestNonzeroFloat64
	above := math.SmallestNonzeroFloat64
	over := math.Nextafter(unity, double)
	spoilW := aGrove00SpoilWeather
	spoilC := aGrove00SpoilCanopy
	return []aGrove00Refusal{
		{"extinction", spoilW(weather, "extinction", below), tree},
		{"depth", spoilW(weather, "depth", nought), tree},
		{"coverage", spoilW(weather, "coverage", below), tree},
		{"coverage above one", spoilW(weather, "coverage", over), tree},
		{"floor above tree", spoilW(weather, "floor", above), tree},
		{"zero radius", weather, spoilC(tree, "radius", nought)},
		{"zero height", weather, spoilC(tree, "height", nought)},
	}
}

// TestGroveRequiresEachWeatherAndCanopyDomainIndependently holds that every
// field of the weather and the canopy is refused outside its domain, one at
// a time, and that a grove inside it is built within a second.
func TestGroveRequiresEachWeatherAndCanopyDomainIndependently(t *testing.T) {
	weather := aGrove00Weather(t, "standard")
	tree := aGrove00Tree(t, "small")
	aGrove00Accepted(t, weather, tree)
	aGrove00NotNumbers(t, weather, tree)
	for _, each := range aGrove00Refusals(weather, tree) {
		aGrove00Check(t, aGrove00Case{each.name, each.weather,
			[]Canopy{each.tree}, flatGround, false})
	}
}

// aGrove00Edges stops the test if there is density at any of the points
// named, which lie on a kind of boundary of the short canopy.
func aGrove00Edges(t *testing.T, grove *Grove, kind string, names []string) {
	t.Helper()
	for _, name := range names {
		point := aGrove00Point(t, name)
		if density := grove.Density(point); density != nought {
			t.Fatalf("%s boundary %v has density %g", kind, point, density)
		}
	}
}

// aGrove00Short stops the test if the short canopy is not where it ends: its
// top is still mist, and the next float above it is not.
func aGrove00Short(t *testing.T, grove *Grove) {
	t.Helper()
	atTop := aGrove00Point(t, "short-top")
	if !(grove.Density(atTop) > nought) {
		t.Fatal("short canopy lost its top boundary")
	}
	past := maths.V(atTop.X, math.Nextafter(atTop.Y, atTop.Y+unity), atTop.Z)
	if got := grove.Density(past); got != nought {
		t.Fatalf("past short canopy: %g", got)
	}
}

// aGrove00Translated stops the test if the density at a point inside the
// translated canopy is not the extinction thinned by the smoothstep there.
func aGrove00Translated(t *testing.T, grove *Grove, weather GroveWeather) {
	t.Helper()
	// At 2 m above the ground, smoothstep(1/4) = 5/32.
	share := picked(t, "a-grove-00-domain-contracts-smooth-numerator") /
		picked(t, "a-grove-00-domain-contracts-smooth-denominator")
	expected := weather.Extinction * (unity - share)
	tolerance := picked(t, "a-grove-00-domain-contracts-tolerance")
	at := aGrove00Point(t, "translated")
	if got := grove.Density(at); !closeTo(got, expected, tolerance) {
		t.Fatalf("translated canopy density %g, want %g", got, expected)
	}
}

// TestGroveBoundsFinestAndTranslatedCanopyEdges holds that a grove's bounds
// and finest cell come from its canopies, and that a canopy's edges are
// where its place and size put them.
func TestGroveBoundsFinestAndTranslatedCanopyEdges(t *testing.T) {
	weather := aGrove00Weather(t, "standard")
	trees := []Canopy{aGrove00Tree(t, "short"), aGrove00Tree(t, "tall")}
	grove, err := NewGrove(weather, trees, aGrove00Floor)
	if err != nil {
		t.Fatal(err)
	}
	want := maths.Box{Min: aGrove00Point(t, "box-min"),
		Max: aGrove00Point(t, "box-max")}
	if got := grove.Bounds(); got != want {
		t.Fatalf("bounds %v, want %v", got, want)
	}
	finest := picked(t, "a-grove-00-domain-contracts-finest")
	if got := grove.Finest(); got != finest {
		t.Fatalf("finest %g, want %g m", got, finest)
	}
	aGrove00Translated(t, grove, weather)
	// The short canopy ends at 3 m, inside the tall canopy's bounding box.
	aGrove00Short(t, grove)
	aGrove00Edges(t, grove, "radial", []string{"radial-east", "radial-south",
		"radial-north", "radial-west"})
	aGrove00Edges(t, grove, "vertical", []string{"vertical-floor",
		"vertical-below", "vertical-ceiling"})
}

// TestGroveTaperUsesTheOuterThirtyPercentOnly holds that a fading grove
// thins only over the outer share of a canopy's radius, and halfway up.
func TestGroveTaperUsesTheOuterThirtyPercentOnly(t *testing.T) {
	weather := aGrove00Weather(t, "standard")
	weather.Fade = true
	tree := aGrove00Tree(t, "wide")
	grove, err := NewGrove(weather, []Canopy{tree}, flatGround)
	if err != nil {
		t.Fatal(err)
	}
	// Halfway up has half the extinction. At radius 85%, the radial
	// smoothstep is halfway through its 70%-100% transition, hence 1/2.
	height := picked(t, "a-grove-00-domain-contracts-taper-height")
	tolerance := picked(t, "a-grove-00-domain-contracts-tolerance")
	for _, row := range readSheet(t, aGrove00TaperFile) {
		radius, want := cell(t, row, "radius"), cell(t, row, "want")
		at := maths.V(tree.At.X+tree.Radius*radius, height, tree.At.Z)
		if got := grove.Density(at); !closeTo(got, want, tolerance) {
			t.Fatalf("radius %g: density %g, want %g", radius, got, want)
		}
	}
}

// aGrove00Counter counts how many times the terrain is asked its height.
type aGrove00Counter struct{ calls atomic.Int64 }

// floor is flat ground at nought that counts each question.
func (counter *aGrove00Counter) floor(float64, float64) float64 {
	counter.calls.Add(unity)
	return nought
}

// TestGroveDoesNotSampleTerrainOutsideItsBounds holds that the terrain is
// never asked its height at a point outside the grove's bounds.
func TestGroveDoesNotSampleTerrainOutsideItsBounds(t *testing.T) {
	var counter aGrove00Counter
	weather := aGrove00Weather(t, "dry")
	grove, err := NewGrove(weather, []Canopy{aGrove00Tree(t, "field")},
		counter.floor)
	if err != nil {
		t.Fatal(err)
	}
	// Bounds are an evaluation boundary too: a sampled terrain callback
	// may only cover the prescribed footprint.
	for _, name := range []string{"outside-west", "outside-east",
		"outside-below", "outside-above", "outside-south", "outside-north"} {
		point := aGrove00Point(t, name)
		if got := grove.Density(point); got != nought {
			t.Fatalf("outside field %v: %g", point, got)
		}
	}
	if counter.calls.Load() != nought {
		t.Fatalf("sampled terrain %d times outside field",
			counter.calls.Load())
	}
}
