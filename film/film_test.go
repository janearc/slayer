package film

import (
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

var up = maths.V(0, 1, 0)

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-12 }

// Film stands on slayer's maths, on celestial for its radiation and on
// libtheme for its colour, and on nothing else, as it once stood on a
// maths of its own, so it could leave. This is the rule, checked.
func TestFilmImportsOnlyMathsCelestialAndLibtheme(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), f, nil,
			parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range parsed.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			switch {
			case !strings.Contains(path, "."): // standard library
			case path == "github.com/janearc/slayer/maths":
			case strings.HasPrefix(path, "github.com/janearc/celestial/"):
			case strings.HasPrefix(path,
				"github.com/janearc/libtheme-css/"):
			default:
				t.Errorf("%s imports %s", f, path)
			}
		}
	}
	if _, err := os.Stat("film.go"); err != nil {
		t.Fatal("the rule looked at nothing: ", err)
	}
}

// A sun's light comes from its direction, from infinitely far, the same
// everywhere.
func TestTheSunIsTheSameEverywhere(t *testing.T) {
	s := Sun{Toward: maths.V(0, 3, 0), Strength: maths.Grey(1.1)}
	for _, p := range []maths.Vec{{}, maths.V(100, 0, -40)} {
		toward, far, l := s.Light(p, up)
		if toward != up || !math.IsInf(far, 1) ||
			l != maths.Grey(1.1) {
			t.Errorf("at %v: %v, %v, %v", p, toward, far, l)
		}
	}
}

// A flood fills from every direction: a surface facing up gets all of it,
// on its side half, facing down none, and it names no direction, so it
// casts no shadow.
func TestAFloodFills(t *testing.T) {
	f := Flood{Strength: maths.Grey(0.4)}
	for _, c := range []struct {
		n    maths.Vec
		want float64
	}{{up, 0.4}, {maths.V(1, 0, 0), 0.2}, {maths.V(0, -1, 0), 0}} {
		toward, _, l := f.Light(maths.Vec{}, c.n)
		if toward != (maths.Vec{}) || !near(l.R, c.want) {
			t.Errorf("facing %v: toward %v, light %v", c.n,
				toward, l)
		}
	}
}

// A spot is bright down its aim, falls off with the square of distance,
// fades at the edge of its cone and gives nothing past it.
func TestASpotMakesAPool(t *testing.T) {
	s := Spot{At: maths.V(0, 4, 0), Aim: maths.V(0, -1, 0),
		Strength: maths.Grey(16), Cone: 30 * maths.Degree,
		Soft: 10 * maths.Degree}
	light := func(p maths.Vec) float64 {
		_, _, l := s.Light(p, up)
		return l.R
	}
	toward, far, _ := s.Light(maths.V(0, 0, 0), up)
	if !toward.Near(up, 1e-12) || !near(far, 4) {
		t.Errorf("under the spot: toward %v, %v away", toward, far)
	}
	if !near(light(maths.V(0, 0, 0)), 1) { // 16 over 4 squared
		t.Errorf("under the spot, %v", light(maths.V(0, 0, 0)))
	}
	if !near(light(maths.V(0, 2, 0)), 4) { // half as far, four times
		t.Errorf("halfway up, %v", light(maths.V(0, 2, 0)))
	}
	edge := 4 * math.Tan(25*maths.Degree) // halfway through the soft
	if l := light(maths.V(edge, 0, 0)); !(l > 0 && l < 1) {
		t.Errorf("in the soft edge, %v", l)
	}
	if l := light(maths.V(4*math.Tan(31*maths.Degree), 0, 0)); l != 0 {
		t.Errorf("past the cone, %v", l)
	}
}

// A hard spot has no soft edge: all its light to the cone and none past.
func TestAHardSpot(t *testing.T) {
	s := Spot{At: maths.V(0, 1, 0), Aim: maths.V(0, -1, 0),
		Strength: maths.Grey(1), Cone: 20 * maths.Degree}
	for _, c := range []struct {
		deg  float64
		want float64
	}{{19, 1}, {21, 0}} {
		x := math.Tan(c.deg * maths.Degree)
		_, d, l := s.Light(maths.V(x, 0, 0), up)
		if !near(l.R*d*d, c.want) {
			t.Errorf("%v degrees off: %v", c.deg, l.R*d*d)
		}
	}
}
