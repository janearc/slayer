package titan

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/grissom/mesh"
)

// vector is a direction from the world's centre. x points to longitude
// and latitude none, y to longitude 90 and z to the north pole, as
// grissom's mesh has them.
type vector struct {
	x, y, z float64
}

// vectorOf is the direction to a place.
func vectorOf(at mesh.Point) vector {
	sinLon, cosLon := math.Sincos(at.Lon() * math.Pi / halfTurn)
	sinLat, cosLat := math.Sincos(at.Lat() * math.Pi / halfTurn)
	return vector{x: cosLat * cosLon, y: cosLat * sinLon, z: sinLat}
}

// point is the place a direction points to. Directions titan makes are
// one long and finite, so the mesh never refuses them.
func (way vector) point() mesh.Point {
	lon := math.Atan2(way.y, way.x) * halfTurn / math.Pi
	lat := math.Atan2(way.z, math.Hypot(way.x, way.y)) * halfTurn / math.Pi
	at, err := mesh.NewPoint(lon, lat)
	if err != nil {
		panic(err)
	}
	return *at
}

// plus is both vectors together.
func (way vector) plus(other vector) vector {
	return vector{way.x + other.x, way.y + other.y, way.z + other.z}
}

// minus is the other vector taken away.
func (way vector) minus(other vector) vector {
	return vector{way.x - other.x, way.y - other.y, way.z - other.z}
}

// scaled is the vector by times as long.
func (way vector) scaled(by float64) vector {
	return vector{way.x * by, way.y * by, way.z * by}
}

// dot is the dot product.
func (way vector) dot(other vector) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// cross is the cross product.
func (way vector) cross(other vector) vector {
	return vector{way.y*other.z - way.z*other.y,
		way.z*other.x - way.x*other.z, way.x*other.y - way.y*other.x}
}

// unit is the same direction at length one.
func (way vector) unit() vector {
	size := math.Sqrt(way.dot(way))
	return vector{way.x / size, way.y / size, way.z / size}
}

// inside is how far a direction lies on the inner side of the great
// circle from one corner to the next: the sine of its angle from it. The
// circle's normal is (from+to) cross (to-from), which keeps its digits
// when the corners are close, as grissom's mesh takes it.
func (way vector) inside(from, to vector) float64 {
	return way.dot(from.plus(to).cross(to.minus(from)).unit())
}

// margin is how far inside a triangle a direction lies: the least of its
// three edges' insides. It is none or more inside and less outside.
func (way vector) margin(shape [corners]corner) float64 {
	return min(way.inside(shape[0].at, shape[1].at),
		way.inside(shape[1].at, shape[2].at),
		way.inside(shape[2].at, shape[0].at))
}

// place is a direction as the bits of its three parts: what a corner is
// known by. Two triangles that share a corner reach it by the same sums,
// so they reach the same bits.
type place struct {
	x, y, z uint64
}

// named is the direction's place. Nought is given one sign, so that it
// has one name.
func (way vector) named() place {
	return place{math.Float64bits(way.x + none),
		math.Float64bits(way.y + none), math.Float64bits(way.z + none)}
}

// The numbers a hash is mixed with. Each is splitmix64's, as Steele, Lea
// and Flood published it in 2014; they are written here from memory.
const (
	golden = 0x9e3779b97f4a7c15
	first  = 0xbf58476d1ce4e5b9
	second = 0x94d049bb133111eb
)

// mix is splitmix64's finaliser: every bit of what goes in reaches every
// bit of what comes out.
func mix(bits uint64) uint64 {
	bits = (bits ^ bits>>30) * first
	bits = (bits ^ bits>>27) * second
	return bits ^ bits>>31
}

// hashed is a number for a direction and a seed, the same for the pair
// for ever.
func (way vector) hashed(seed uint64) uint64 {
	name := way.named()
	bits := mix(seed + golden)
	bits = mix(bits ^ mix(name.x+golden))
	bits = mix(bits ^ mix(name.y+golden))
	return mix(bits ^ mix(name.z+golden))
}

// unit is the top 53 bits of a hash as a number from nought up to one.
func unit(bits uint64) float64 { return float64(bits>>11) / (1 << 53) }

// corner is one corner of a triangle of the ground.
type corner struct {
	at     vector
	kind   Kind
	height float64 // metres above the map's zero
	made   float64 // how many of those metres are made
	// differs is how much the map's heights differ, on average, between
	// places one spacing apart near here. hurst is how fast that falls
	// as the places close in: differs goes as distance to the hurst.
	// Both are the map's, read at measured corners and carried between
	// them.
	differs, hurst, spacing float64
	// slope is which way the ground rises here and how fast, in metres
	// a metre: a direction along the ground, as long as the ground is
	// steep. The map's at a measured corner; carried along an edge to a
	// made one.
	slope vector
}

// read is what the map says at a measured corner.
type read struct {
	height, differs, hurst, spacing float64
	slope                           vector
}

// read is the corner at a direction, read from the map: its height, and
// how rough the map is round it. Corners are kept, so the map is asked
// about each once.
//
// The roughness is read at the map's own spacing and at twice it, at six
// places round the corner each time. The first is how much heights
// differ a sample apart. The ratio of the two is two to the Hurst
// exponent: one where a slope carries on, a half for a random walk,
// nought where each step forgets the last. Twelve places are few, and a
// reading outside nought to one is held to the nearer end.
func (ground *Ground) read(at vector) (corner, error) {
	name := at.named()
	ground.guard.Lock()
	kept, found := ground.kept[name]
	ground.guard.Unlock()
	if !found {
		var err error
		kept, err = ground.ask(at)
		if err != nil {
			return corner{}, err
		}
		ground.guard.Lock()
		if len(ground.kept) >= most {
			clear(ground.kept)
		}
		ground.kept[name] = kept
		ground.guard.Unlock()
	}
	return corner{at: at, kind: Measured, height: kept.height,
		differs: kept.differs, hurst: kept.hurst,
		spacing: kept.spacing, slope: kept.slope}, nil
}

// ask is what the map says at a direction and round it.
func (ground *Ground) ask(at vector) (asked read, err error) {
	here := at.point()
	height, known, err := ground.from.Height(here)
	if err != nil {
		return asked, err
	}
	if !known {
		return asked, fmt.Errorf("%w: the map has no height at %.6f, "+
			"%.6f", errRefused, here.Lon(), here.Lat())
	}
	asked = read{height: height, hurst: smooth,
		spacing: ground.from.Finest(here)}
	if !(asked.spacing > none) {
		return asked, nil
	}
	near, slope, err := ground.differs(at, height, asked.spacing)
	if err != nil {
		return asked, err
	}
	far, _, err := ground.differs(at, height, twice*asked.spacing)
	if err != nil {
		return asked, err
	}
	asked.differs, asked.slope = near, slope
	if near > none && far > none {
		asked.hurst = min(max(math.Log2(far/near), rough), smooth)
	}
	return asked, nil
}

// differs is how much the map's heights differ from a corner's, at a
// distance in metres from it: the root of the mean square over the places
// round it that the map knows. Round a corner the map knows nothing
// about, it is none.
//
// The same readings give the ground's slope at the corner: the one slope
// that fits the rises to the places round it best. With fewer than all
// of them known the fit is not trusted, and the slope is none.
func (ground *Ground) differs(at vector, height, metres float64) (float64,
	vector, error) {
	// two directions square to the corner's, to step round it by
	other := vector{x: 1}
	if math.Abs(at.x) > half {
		other = vector{y: 1}
	}
	east := other.cross(at).unit()
	north := at.cross(east)
	angle := metres / ground.radius
	var squares float64
	var slope vector
	var counted int
	for step := range around {
		sin, cos := math.Sincos(turn * float64(step) / around)
		way := east.scaled(cos).plus(north.scaled(sin))
		there, known, err := ground.from.Height(
			at.plus(way.scaled(angle)).unit().point())
		if err != nil {
			return none, vector{}, err
		}
		if known {
			rise := there - height
			squares += rise * rise
			slope = slope.plus(way.scaled(rise))
			counted++
		}
	}
	if counted == none {
		return none, vector{}, nil
	}
	if counted < around {
		slope = vector{}
	}
	// evenly round a corner, the ways' squares sum to half their count
	slope = slope.scaled(split / (around * metres))
	return math.Sqrt(squares / float64(counted)), slope, nil
}
