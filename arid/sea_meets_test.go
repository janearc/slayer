package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// meetHeights are how high, in the sea's reach, the table's rays begin:
// well over the slab, in it above the mean, in it below, and well under.
var meetHeights = []float64{4, 0.5, -0.5, -4}

// meetSlants are how steeply, up for more than nought, the table's rays
// go for each length they go level: steep, shallow, skimming and level.
var meetSlants = []float64{-3, -0.3, -0.02, 0, 0.02, 0.3, 3}

// meetSpans are the nears and fars the table's rays are asked between: far
// enough for any ray of it to cross the slab, and short stretches, one that
// starts after the ray has begun. Ten kilometres is far enough and is not
// so many steps over a coast that a skimming ray takes long.
var meetSpans = [][2]float64{{0, 1e4}, {2, 1e4}, {0, 10}, {2, 40}}

// meetBegins are where, east and south, the table's rays begin: on the
// middle of a shore's ground and far off it.
var meetBegins = [][2]float64{{100, 100}, {-3000, 2500}, {30, 30}}

// meetCase is one ray of the table and the span asked of it.
type meetCase struct {
	ray       maths.Ray
	near, far float64
}

// meetTable is every ray of the table over a sea: each height, slant,
// way round and span, from each place.
func meetTable(sea *Sea) []meetCase {
	var table []meetCase
	for _, begin := range meetBegins {
		for _, height := range meetHeights {
			for _, slant := range meetSlants {
				for way := range 5 {
					turn := float64(way) * 1.3
					ray := maths.Ray{At: maths.V(begin[0],
						height*sea.reach, begin[1]),
						Dir: maths.V(math.Cos(turn), slant, math.Sin(turn))}
					for _, span := range meetSpans {
						table = append(table,
							meetCase{ray: ray, near: span[0], far: span[1]})
					}
				}
			}
		}
	}
	return table
}

// agree fails the test for each ray of the table for which Meets is not
// whether Hit finds the surface, and says how many of the rays Meets
// answered outright.
func agree(t *testing.T, sea *Sea) (outright int) {
	t.Helper()
	for index, each := range meetTable(sea) {
		_, hit := sea.Hit(each.ray, each.near, each.far)
		if met := sea.Meets(each.ray, each.near, each.far); met != hit {
			t.Fatalf("ray %d %+v between %g and %g: Meets %t, Hit %t",
				index, each.ray, each.near, each.far, met, hit)
		}
		if sea.crossed(each.ray, each.near, each.far) {
			outright++
		}
	}
	return outright
}

// A ray through the whole slab of a sea with no coast, which needs no
// search, and every other ray of the table, are met as Hit finds them.
func TestSeaMeetsIsWhetherHitFindsIt(t *testing.T) {
	sea := fixtureSea(t)
	outright := agree(t, sea)
	if outright == 0 || outright == len(meetTable(sea)) {
		rays := len(meetTable(sea))
		t.Fatalf("%d of %d rays were answered outright: the table should "+
			"have some and leave some", outright, rays)
	}
}

// Over a coast a ray is the same, the part of the table over its grid
// searched, and the rest, off the grid, answered outright where it can be.
func TestShoredSeaMeetsIsWhetherHitFindsIt(t *testing.T) {
	shored, _ := ashore(t, fixtureSea(t), shaped(41, 41, 5, rampHeight))
	if outright := agree(t, shored); outright == 0 {
		t.Fatal("no ray off the grid was answered outright")
	}
}

// A ray down through a hollow the open sea does not reach is over no
// water: it is not answered outright, and Meets says what Hit does, that
// it meets nothing.
func TestAnswerIsNotOutrightOverADryHollow(t *testing.T) {
	shored, _ := ashore(t, fixtureSea(t), shaped(7, 7, 10, basinHeight))
	ray := maths.Ray{At: maths.V(30, 10*shored.reach, 30),
		Dir: maths.V(0, -1, 0)}
	if shored.crossed(ray, 0, 1e6) {
		t.Fatal("a ray over a dry hollow was answered outright")
	}
	if shored.Meets(ray, 0, 1e6) {
		t.Fatal("a ray down a dry hollow met water")
	}
}

// A ray that begins inside the slab does not cross it, and one that is
// cut short of the second face does not: neither is answered outright.
func TestAnswerIsNotOutrightForARayThatDoesNotCrossTheSlab(t *testing.T) {
	sea := fixtureSea(t)
	inside := maths.Ray{At: maths.V(0, 0.5*sea.reach, 0), Dir: maths.V(0, -1, 0)}
	if sea.crossed(inside, 0, 1e6) {
		t.Fatal("a ray that began in the slab was answered outright")
	}
	// The ray starts four reaches over the mean and falls: it meets the
	// slab's top face three reaches along and its bottom face five. A far
	// of five would reach the second face, so the cut is half a reach short.
	const shortOfSecondFace = 4.5
	over := maths.Ray{At: maths.V(0, 4*sea.reach, 0), Dir: maths.V(0, -1, 0)}
	if sea.crossed(over, 0, shortOfSecondFace*sea.reach) {
		t.Fatal("a ray cut short of the second face was answered outright")
	}
	if !sea.crossed(over, 0, 1e6) {
		t.Fatal("a ray through the whole slab was not answered outright")
	}
}
