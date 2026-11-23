package maths_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/slayer/maths"
)

// What the draws count in, each defined by what is drawn.
const (
	coinSides  = 2 // defined: a coin has two sides, so a draw of two is heads or tails
	bothWays   = 2 // defined: a reach goes out from nought either way, minus and plus
	limitKinds = 4 // defined: the three ways near and far are drawn, and the way they are left open
)

// the kinds of number the wild draw makes, by the draw's number; a draw past
// these gives an ordinary number
const (
	onTheGrid = iota
	onTheQuarters
	exactlyZero
	nearlyNothing
	farOut
	anExtreme
)

// the ways near and far are drawn, by the draw's number; a draw past these
// leaves them open, from nought to infinity
const (
	anyNearAndFar = iota
	nearNoughtFarAny
	farAtTheReach
)

// shape is what the draws are made of, read once from numbers.yaml and
// the csv files: how the kinds are weighed, the scales they are drawn at and
// the extremes of what a float holds.
type shape struct {
	wildKinds, gridReach, tameReach     int
	tameOdds, boxOdds, zeroOdds         int
	gridDivisor, tameDivisor            float64
	tinyScale, hugeScale, ordinaryScale float64
	tameScale, farReach                 float64
	extremes                            []float64
}

// shapeOf reads the shape of the draws.
func shapeOf(test reporter) shape {
	test.Helper()
	return shape{
		wildKinds:     count(test, "draw-kinds"),
		gridReach:     count(test, "grid-reach"),
		tameReach:     count(test, "tame-reach"),
		tameOdds:      count(test, "tame-odds"),
		boxOdds:       count(test, "box-odds"),
		zeroOdds:      count(test, "zero-odds"),
		gridDivisor:   picked(test, "grid-divisor"),
		tameDivisor:   picked(test, "tame-divisor"),
		tinyScale:     picked(test, "tiny-scale"),
		hugeScale:     picked(test, "huge-scale"),
		ordinaryScale: picked(test, "ordinary-scale"),
		tameScale:     picked(test, "tame-scale"),
		farReach:      picked(test, "far-reach"),
		extremes:      valuesOf(test, extremesFile),
	}
}

// valuesOf is the value column of a csv file, in the order of its rows.
func valuesOf(test reporter, file string) []float64 {
	test.Helper()
	var values []float64
	for _, row := range read(test, file) {
		values = append(values, float(test, row, "value"))
	}
	return values
}

// drawing is a seeded stream of boxes, rays and limits that touch, graze,
// run along a face, or sit at the edge of what a float holds. Each draw is
// tame or wild: most keep the numbers sane, so the interesting answers, a hit
// and a graze, are many.
type drawing struct {
	random *rand.Rand
	shape  shape
	tame   bool
}

// randomFrom is a stream of random numbers seeded by a picked number.
func randomFrom(test reporter, seedName string) *rand.Rand {
	test.Helper()
	seed := int64(picked(test, seedName))
	return rand.New(rand.NewSource(seed))
}

// drawingOf is the stream seeded by a picked number.
func drawingOf(test reporter, seedName string) *drawing {
	test.Helper()
	return &drawing{random: randomFrom(test, seedName), shape: shapeOf(test)}
}

// next starts a draw, tame or wild.
func (drawn *drawing) next() {
	drawn.tame = drawn.random.Intn(drawn.shape.tameOdds) != nought
}

// grid is a whole number from minus the grid's reach to plus it, so faces
// and origins meet exactly.
func (drawn *drawing) grid() int {
	reach := drawn.shape.gridReach
	return drawn.random.Intn(bothWays*reach+unity) - reach
}

// wild is a number of a kind chosen to make boxes and rays that touch,
// graze, run along a face, or sit at the edge of what a float holds.
func (drawn *drawing) wild() float64 {
	switch drawn.random.Intn(drawn.shape.wildKinds) {
	case onTheGrid:
		return float64(drawn.grid())
	case onTheQuarters:
		return float64(drawn.grid()) / drawn.shape.gridDivisor
	case exactlyZero:
		return nought
	case nearlyNothing:
		return drawn.random.NormFloat64() * drawn.shape.tinyScale
	case farOut:
		return drawn.random.NormFloat64() * drawn.shape.hugeScale
	case anExtreme:
		extremes := drawn.shape.extremes
		return extremes[drawn.random.Intn(len(extremes))]
	}
	return drawn.random.NormFloat64() * drawn.shape.ordinaryScale
}

// tamed is a number that keeps the draw sane: a grid of halves, or a
// normal number.
func (drawn *drawing) tamed() float64 {
	reach := drawn.shape.tameReach
	if drawn.random.Intn(coinSides) == nought {
		grid := drawn.random.Intn(bothWays*reach+unity) - reach
		return float64(grid) / drawn.shape.tameDivisor
	}
	return drawn.random.NormFloat64() * drawn.shape.tameScale
}

// number is the draw's next number.
func (drawn *drawing) number() float64 {
	if drawn.tame {
		return drawn.tamed()
	}
	return drawn.wild()
}

// vec is three of the draw's numbers.
func (drawn *drawing) vec() maths.Vec {
	return maths.V(drawn.number(), drawn.number(), drawn.number())
}

// boxBetween is the box with two corners, the lesser of each axis for Min
// and the greater for Max.
func boxBetween(first, second maths.Vec) maths.Box {
	return maths.Box{Min: maths.V(min(first.X, second.X), min(first.Y, second.Y),
		min(first.Z, second.Z)),
		Max: maths.V(max(first.X, second.X), max(first.Y, second.Y),
			max(first.Z, second.Z))}
}

// box is a box the right way round, as a tree holds, or sometimes turned
// inside out, or with a corner that is no number.
func (drawn *drawing) box() maths.Box {
	first, second := drawn.vec(), drawn.vec()
	if drawn.random.Intn(drawn.shape.boxOdds) == nought {
		return maths.Box{Min: first, Max: second}
	}
	if math.IsNaN(first.X + second.X + first.Y + second.Y + first.Z + second.Z) {
		return maths.Box{Min: first, Max: second}
	}
	return boxBetween(first, second)
}

// ray is a ray whose way has an axis of nought now and then.
func (drawn *drawing) ray() maths.Ray {
	way := drawn.vec()
	for _, axis := range []*float64{&way.X, &way.Y, &way.Z} {
		if drawn.random.Intn(drawn.shape.zeroOdds) == nought {
			*axis = nought
		}
	}
	return maths.Ray{At: drawn.vec(), Dir: way}
}

// limits are the near and far a ray is asked between: drawn, or from nought
// to a drawn far, or from nought to the far reach, or left open.
func (drawn *drawing) limits() (near, far float64) {
	near, far = nought, math.Inf(infinity)
	switch drawn.random.Intn(limitKinds) {
	case anyNearAndFar:
		near, far = drawn.number(), drawn.number()
	case nearNoughtFarAny:
		far = math.Abs(drawn.number())
	case farAtTheReach:
		far = drawn.shape.farReach
	}
	return near, far
}

// agree stops the test unless Meets says what Hit says for the box, ray, near
// and far.
func agree(test reporter, box maths.Box, ray maths.Ray, near, far float64) bool {
	test.Helper()
	_, _, want := box.Hit(ray, near, far)
	if got := box.Meets(ray, ray.Dir.Inverse(), near, far); got != want {
		test.Fatalf("box %v, ray %v, near %v, far %v: Meets says %v and Hit "+
			"says %v", box, ray, near, far, got, want)
	}
	return want
}

// For every box, ray, near and far a seeded draw makes, Meets answers as Hit
// does: the grid draws make rays that graze faces, edges and corners, and
// the others make zero, tiny, huge, infinite and NaN numbers.
//
// The draws must also test both answers: each of a hit and a miss is at least
// a share of them. A draw that was all misses would pass the agreement alone.
func TestMeetsIsHitsAnswerForEveryRay(test *testing.T) {
	drawn := drawingOf(test, "meets-seed")
	cases, hit := count(test, "meets-cases"), nought
	for range cases {
		drawn.next()
		box, ray := drawn.box(), drawn.ray()
		near, far := drawn.limits()
		if agree(test, box, ray, near, far) {
			hit++
		}
	}
	least := picked(test, "answer-share-least") * float64(cases)
	if float64(hit) < least || float64(hit) > float64(cases)-least {
		test.Errorf("%d of %d draws hit: the draws do not test both answers",
			hit, cases)
	}
}

// A ray that runs along a face of a box, or touches an edge or a corner,
// is met or missed as Hit says, whichever way the numbers round.
func TestMeetsAgreesAtTheFacesOfAUnitBox(test *testing.T) {
	box := maths.Box{Min: vector(test, "origin"),
		Max: vector(test, "corner of the unit box")}
	steps, fars := valuesOf(test, stepsFile), valuesOf(test, farsFile)
	depth := picked(test, "face-ray-start")
	for _, startX := range steps {
		for _, startY := range steps {
			for _, wayX := range steps {
				for _, wayY := range steps {
					ray := maths.Ray{At: maths.V(startX, startY, depth),
						Dir: maths.V(wayX, wayY, unity)}
					for _, far := range fars {
						agree(test, box, ray, nought, far)
					}
				}
			}
		}
	}
}

// stepping is how many places from a number, in which direction, a distance
// is asked at.
type stepping struct {
	places int
	toward float64
}

// neighboursOf is the steppings of neighbours.csv: each a number of places up
// or down from a distance.
func neighboursOf(test reporter) []stepping {
	test.Helper()
	var neighbours []stepping
	for _, row := range read(test, neighboursFile) {
		toward := math.Inf(infinity)
		switch row["direction"] {
		case "up":
		case "down":
			toward = math.Inf(-infinity)
		default:
			test.Fatalf("%s: direction is %q", row["name"], row["direction"])
		}
		places := int(float(test, row, "places"))
		neighbours = append(neighbours, stepping{places: places, toward: toward})
	}
	return neighbours
}

// from is the distance moved by the stepping's places, one unit in the last
// place each.
func (step stepping) from(along float64) float64 {
	for range step.places {
		along = math.Nextafter(along, step.toward)
	}
	return along
}

// normals is three normal numbers from a stream.
func normals(random *rand.Rand) maths.Vec {
	return maths.V(random.NormFloat64(), random.NormFloat64(), random.NormFloat64())
}

// A ray that reaches a corner of a box at a distance, asked up to that
// distance and the ones a few places either side of it, is where the
// quotients Hit takes and the products Meets takes differ in the last
// place: Meets answers as Hit does.
func TestMeetsAgreesAtACornerReachedAtTheFar(test *testing.T) {
	random := randomFrom(test, "corner-seed")
	neighbours := neighboursOf(test)
	span, least := picked(test, "corner-along-span"), picked(test, "corner-along-least")
	for range count(test, "corner-cases") {
		ray := maths.Ray{At: normals(random), Dir: normals(random)}
		along := random.Float64()*span + least
		corner := ray.At.Add(ray.Dir.Scale(along))
		other := corner.Add(normals(random))
		box := boxBetween(corner, other)
		for _, step := range neighbours {
			agree(test, box, ray, nought, step.from(along))
		}
	}
}
