package film_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"pgregory.net/rapid"
)

// the tables the emitter tests are run over
const (
	radarSeenFile    = "testdata/radar-seen.csv"
	radarBeamsFile   = "testdata/radar-beams.csv"
	sunSeenFile      = "testdata/sun-seen.csv"
	floodFacingFile  = "testdata/flood-facing.csv"
	spotEdgeFile     = "testdata/spot-edge.csv"
	spotHardFile     = "testdata/spot-hard.csv"
	spotRoundingFile = "testdata/spot-rounding.csv"
)

// what the emitter tests choose, each picked in picked-numbers.yml
var (
	insideMost   = data.Picked("film-inside-most")
	strengthMost = data.Picked("film-strength-most")
	coneMost     = data.Picked("film-cone-most")
)

// vector is the vector of a row. Its columns are the name with x, y or z
// after an underscore. Rows without them stop the test.
func vector(t *testing.T, row data.Row, name string) maths.Vec {
	t.Helper()
	return maths.V(number(t, row, name+"_x"), number(t, row, name+"_y"),
		number(t, row, name+"_z"))
}

// label is the label of a row.
func label(row data.Row) string {
	text, _ := row.Text("label")
	return text
}

// place draws a vector, each part within eyeMost of zero.
func place(t *rapid.T, name string) maths.Vec {
	part := rapid.Float64Range(-eyeMost, eyeMost)
	return maths.V(part.Draw(t, name+" x"), part.Draw(t, name+" y"),
		part.Draw(t, name+" z"))
}

// strong draws a light of three lamps.
func strong(t *rapid.T, name string) maths.Light {
	part := rapid.Float64Range(none, strengthMost)
	return maths.RGB(part.Draw(t, name+" red"),
		part.Draw(t, name+" green"), part.Draw(t, name+" blue"))
}

// apart draws a place at least a unit away from another.
func apart(t *rapid.T, name string, from maths.Vec) maths.Vec {
	part := rapid.Float64Range(-eyeMost, eyeMost)
	sign := rapid.SampledFrom([]float64{-whole, whole}).Draw(t, name+" sign")
	far := rapid.Float64Range(whole, eyeMost).Draw(t, name+" x")
	up, deep := part.Draw(t, name+" y"), part.Draw(t, name+" z")
	return from.Add(maths.V(sign*far, up, deep))
}

// Radars beam their strength in their range alone. The range holds its
// low end and not its high one. They light nothing in visible light.
func TestARadarBeamsInItsRangeAlone(t *testing.T) {
	for _, row := range table(t, radarBeamsFile) {
		strong := number(t, row, "beams") * eyeMost
		glass := film.NewRadar(maths.Vec{}, radiation.XRay(), strong)
		band := maths.Band{Nanometres: number(t, row, "nanometres")}
		if got := glass.Beams(band); got != strong {
			t.Errorf("%s: at %v nm it beams %v, want %v", label(row),
				band.Nanometres, got, strong)
		}
	}
	rapid.Check(t, radarBeamsInItsRange)
}

func radarBeamsInItsRange(t *rapid.T) {
	profile := rapid.SampledFrom([]radiation.Range{radiation.XRay(),
		radiation.Ultraviolet(), radiation.Infrared(), radiation.Radio(),
		radiation.Microwave()}).Draw(t, "profile")
	strength := rapid.Float64Range(whole, strengthMost).Draw(t, "strength")
	glass := film.NewRadar(maths.Vec{}, profile, strength)
	low, high := profile.Edges()
	share := rapid.Float64Range(none, insideMost).Draw(t, "share")
	inside := maths.Band{Nanometres: low + share*(high-low)}
	if got := glass.Beams(inside); got != strength {
		t.Fatalf("%v beams %v inside its range at %v", profile, got,
			inside.Nanometres)
	}
	for _, out := range []float64{low / pair, high, high * pair} {
		if got := glass.Beams(maths.Band{Nanometres: out}); got != none {
			t.Fatalf("%v beams %v at %v", profile, got, out)
		}
	}
	if got := glass.Beams(maths.Band{Nanometres: low}); got != strength {
		t.Fatalf("%v beams %v at the low end of its range", profile,
			got)
	}
}

// Radars are where they are from the point asked. They are at their
// distance and give no visible light.
func TestARadarIsWhereItIs(t *testing.T) {
	for _, row := range table(t, radarSeenFile) {
		glass := film.NewRadar(vector(t, row, "at"), radiation.Radio(), whole)
		toward, dist, light := glass.Light(vector(t, row, "from"),
			maths.Up)
		if !toward.Near(vector(t, row, "toward"), tight) ||
			math.Abs(dist-number(t, row, "dist")) > tight ||
			light != (maths.Light{}) {
			t.Errorf("%s: %v, %v, %v", label(row), toward, dist, light)
		}
	}
	rapid.Check(t, radarLooksBackAtItself)
}

func radarLooksBackAtItself(t *rapid.T) {
	at := place(t, "at")
	from := apart(t, "from", at)
	glass := film.NewRadar(at, radiation.Radio(), whole)
	toward, dist, light := glass.Light(from, maths.Up)
	back := from.Add(toward.Scale(dist))
	if !back.Near(at, loose*dist) || light != (maths.Light{}) ||
		math.Abs(toward.Len()-whole) > tight {
		t.Fatalf("from %v the radar at %v is %v away along %v", from, at,
			dist, toward)
	}
}

// The sun is its direction, one unit long. It is infinitely far at its
// full strength wherever it is looked from.
func TestTheSunIsTheSameEverywhere(t *testing.T) {
	for _, row := range table(t, sunSeenFile) {
		glass := film.NewSun(vector(t, row, "toward"), maths.Grey(whole))
		toward, dist, _ := glass.Light(maths.Vec{}, maths.Up)
		if !toward.Near(vector(t, row, "unit"), tight) ||
			!math.IsInf(dist, whole) {
			t.Errorf("%s: %v, %v", label(row), toward, dist)
		}
	}
	rapid.Check(t, sunIsTheSameEverywhere)
}

func sunIsTheSameEverywhere(t *rapid.T) {
	aim := apart(t, "toward", maths.Vec{})
	strength := strong(t, "strength")
	glass := film.NewSun(aim, strength)
	toward, dist, light := glass.Light(place(t, "p"), place(t, "n"))
	if !toward.Near(aim.Unit(), tight) || !math.IsInf(dist, whole) ||
		light != strength {
		t.Fatalf("the sun: %v, %v, %v", toward, dist, light)
	}
}

// Floods are no direction and infinitely far. All their strength faces
// up. Half is on their side. None faces down. They use the world's up
// when they are given none.
func TestAFloodFillsFromAbove(t *testing.T) {
	for _, row := range table(t, floodFacingFile) {
		for _, up := range []maths.Vec{{}, maths.Up} {
			glass := film.NewFlood(up, maths.Grey(whole))
			toward, dist, light := glass.Light(maths.Vec{},
				vector(t, row, "normal"))
			want := maths.Grey(number(t, row, "share"))
			if toward != (maths.Vec{}) || !math.IsInf(dist, whole) ||
				light != want {
				t.Errorf("%s with up %v: %v, %v, %v", label(row), up,
					toward, dist, light)
			}
		}
	}
	rapid.Check(t, floodTradesFacesForShares)
}

func floodTradesFacesForShares(t *rapid.T) {
	up := apart(t, "up", maths.Vec{})
	strength := strong(t, "strength")
	glass := film.NewFlood(up, strength)
	normal := apart(t, "normal", maths.Vec{})
	_, _, facing := glass.Light(maths.Vec{}, normal)
	_, _, away := glass.Light(maths.Vec{}, normal.Neg())
	total := facing.Add(away)
	if math.Abs(total.R-strength.R) > loose*strength.R ||
		math.Abs(total.G-strength.G) > loose*strength.G ||
		math.Abs(total.B-strength.B) > loose*strength.B {
		t.Fatalf("a face and its back get %v and %v of %v", facing, away,
			strength)
	}
}

// aimed draws a spot at the origin and the way it is aimed. It has a
// drawn strength and cone. Its soft edge is no wider than the cone.
func aimed(t *rapid.T) (film.Spot, maths.Vec, maths.Light) {
	aim := apart(t, "aim", maths.Vec{}).Unit()
	strength := strong(t, "strength")
	cone := rapid.Float64Range(wideLeast, coneMost).Draw(t, "cone")
	soft := rapid.Float64Range(none, cone).Draw(t, "soft")
	return film.NewSpot(maths.Vec{}, aim, strength, cone, soft), aim, strength
}

// Spots fall off with the square of the distance. Down the aim the light
// times the distance squared is the strength. That holds at any distance.
func TestASpotFallsOffWithTheSquare(t *testing.T) {
	rapid.Check(t, spotFallsOffWithTheSquare)
}

func spotFallsOffWithTheSquare(t *rapid.T) {
	glass, aim, strength := aimed(t)
	for _, name := range []string{"near", "far"} {
		dist := rapid.Float64Range(wideLeast, eyeMost).Draw(t, name)
		toward, got, light := glass.Light(aim.Scale(dist), maths.Up)
		if math.Abs(got-dist) > loose*dist ||
			!toward.Near(aim.Neg(), tight) {
			t.Fatalf("a spot is %v away along %v, not %v along %v", got,
				toward, dist, aim.Neg())
		}
		scaled := light.Scale(dist * dist)
		if math.Abs(scaled.R-strength.R) > loose*strengthMost ||
			math.Abs(scaled.G-strength.G) > loose*strengthMost ||
			math.Abs(scaled.B-strength.B) > loose*strengthMost {
			t.Fatalf("%v away the light %v scales to %v, not %v", dist,
				light, scaled, strength)
		}
	}
}

// At the lamp itself there is no direction and no light.
func TestASpotAtItsOwnPlaceGivesNothing(t *testing.T) {
	rapid.Check(t, spotGivesNothingAtItsPlace)
}

func spotGivesNothingAtItsPlace(t *rapid.T) {
	glass, _, _ := aimed(t)
	toward, dist, light := glass.Light(maths.Vec{}, maths.Up)
	if toward != (maths.Vec{}) || dist != none || light != (maths.Light{}) {
		t.Fatalf("at the lamp: %v, %v, %v", toward, dist, light)
	}
}

// offAim is a point a unit away from the origin at an angle off an aim
// of down minus y, in the plane of x and y.
func offAim(off float64) maths.Vec {
	return maths.V(math.Sin(off), -math.Cos(off), none)
}

// Spots light a point by how far it is off their aim. Cones of half a
// radian softened over a fifth light all inside and none past the cone.
// The smooth step lies between. Hard edges light all short of the cone.
func TestASpotsEdgeIsASmoothStep(t *testing.T) {
	soft := film.NewSpot(maths.Vec{}, maths.V(none, -whole, none),
		maths.Grey(whole), half, data.Picked("film-soft-edge"))
	hard := film.NewSpot(maths.Vec{}, maths.V(none, -whole, none),
		maths.Grey(whole), half, none)
	for _, row := range table(t, spotEdgeFile) {
		off := offAim(number(t, row, "off"))
		_, _, light := soft.Light(off, maths.Up)
		if math.Abs(light.R-number(t, row, "share")) > tight {
			t.Errorf("%s: %v", label(row), light)
		}
	}
	for _, row := range table(t, spotHardFile) {
		off := offAim(number(t, row, "off"))
		_, _, light := hard.Light(off, maths.Up)
		if math.Abs(light.R-number(t, row, "share")) > tight {
			t.Errorf("%s: %v", label(row), light)
		}
	}
}

// The further off the aim, the less light.
func TestASpotLightsLessTheFurtherOffItsAim(t *testing.T) {
	rapid.Check(t, spotLightsLessFurtherOff)
}

func spotLightsLessFurtherOff(t *rapid.T) {
	cone := rapid.Float64Range(wideLeast, coneMost).Draw(t, "cone")
	soft := rapid.Float64Range(none, cone).Draw(t, "soft")
	glass := film.NewSpot(maths.Vec{}, maths.V(none, -whole, none),
		maths.Grey(whole), cone, soft)
	near := rapid.Float64Range(none, math.Pi).Draw(t, "near")
	far := rapid.Float64Range(near, math.Pi).Draw(t, "far")
	_, _, lit := glass.Light(offAim(near), maths.Up)
	_, _, dim := glass.Light(offAim(far), maths.Up)
	if lit.R < dim.R-tight || lit.R > whole+tight || dim.R < none {
		t.Fatalf("%v off the aim is lit %v and %v off is lit %v", near,
			lit.R, far, dim.R)
	}
}

// The dot of the way to a point and the aim may round past one. It may
// round below minus one. Straight down the aim is then all the light.
// Straight behind is none. Neither is not a number.
func TestASpotAimedWhereTheDotRoundsPastOne(t *testing.T) {
	for _, row := range table(t, spotRoundingFile) {
		strength := maths.Grey(number(t, row, "strength"))
		glass := film.NewSpot(maths.Vec{}, vector(t, row, "aim"), strength,
			half, none)
		_, _, light := glass.Light(vector(t, row, "from"), maths.Up)
		if math.IsNaN(light.R) ||
			math.Abs(light.R-number(t, row, "share")) > tight {
			t.Errorf("%s: %v", label(row), light)
		}
	}
}
