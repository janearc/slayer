package trace

import (
	"errors"
	"math"
	"sync/atomic"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Wire is the picture of what the light picture's first rays meet,
// drawn as edges: each pixel's rays are the rays Sampled makes, through
// the same staging, camera, time and tree of things, and each takes the
// first thing it meets, as Sampled's do.
//
// On a face of a mesh the pixel is bright within half a line's width of
// the face's nearest edge, measured on the picture, and faint between.
// A surface with no polygons, a sphere, a plane or a marched shape, is
// one flat tint, and no edge is made up for it. A height field is that
// tint with its sampled cells drawn as edges: the cells are real, the
// edges are not polygons. The sky and every volume are black.
//
// Nothing is lit, shadowed or mirrored. A ray that would enter a
// gravitational field before it meets a surface bends in the light
// picture, and Wire refuses the picture rather than draw it straight.
func Wire(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int) (*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, rays)
	if err != nil {
		return nil, err
	}
	if err := fieldFault(world); err != nil {
		return nil, err
	}
	wired := &wiring{seen: seen.fine(), rays: float64(rays),
		crowd:  gathered(place(world, moment)),
		fields: placeFields(world, moment)}
	drawn := paint(wired.seen.width, wired.seen.height, wired.pixel)
	if wired.bent.Load() {
		return nil, errWireBent
	}
	if rays == 1 {
		return drawn, nil
	}
	return averaged(drawn, width, height, rays), nil
}

// errWireBent is the refusal of a wire picture whose rays a field bends.
var errWireBent = errors.New("trace: a ray of this wire picture enters " +
	"a gravitational field before it meets a surface; the light picture " +
	"bends it there, and the wire draws straight rays only")

// the wire picture's numbers
const (
	// wireWidth is how wide an edge is drawn, in pixels of the picture
	// asked for, whatever its rays. Picked: one pixel, so a dense
	// mesh far off stays lines and does not fill in.
	wireWidth = 1.0
	// wireLine is the light of an edge, white through a plain tone.
	// Picked.
	wireLine = 1.0
	// wireFace is the faint light of a face between its edges, so a
	// mesh's silhouette reads against the black. Picked.
	wireFace = 0.03
	// wireRamp is how many fine pixels an edge's side fades over, so
	// a picture of one ray a pixel is not stepped. Picked: one.
	wireRamp = 1.0
)

// The wire picture's tints, each picked.
var (
	// wireAnalytic is a surface with no polygons: analytic, a dim blue.
	wireAnalytic = maths.Light{R: 0.01, G: 0.03, B: 0.09}
	// wireCell is a height field's cell edge, a pale blue: the cells
	// are the field's real samples, and not polygons.
	wireCell = maths.Light{R: 0.35, G: 0.6, B: 1}
	// wireUnknown is a shape that answers a ray exactly and is neither
	// a mesh this file knows nor an analytic shape it knows, a dim
	// magenta, so it is seen and is taken for neither.
	wireUnknown = maths.Light{R: 0.3, B: 0.3}
)

// wiring is one wire picture in progress: the fine view its rays are
// made for, its rays a side, the things and fields placed at its time,
// and whether a field bent any of its rays.
type wiring struct {
	seen   view
	rays   float64
	crowd  crowd
	fields []placedField
	bent   atomic.Bool
}

// look is the ray the fine view makes for a pixel, as through makes it
// for the light picture.
func (wired *wiring) look(x, y int) maths.Ray {
	return film.Look(wired.seen.eye, wired.seen.lens, x, y,
		wired.seen.width, wired.seen.height, maths.Band{})
}

// pixel is one fine pixel of the wire: black and uncovered for the
// sky, and otherwise the first thing met, drawn by what it is.
func (wired *wiring) pixel(x, y int) frame.Pixel {
	ray := wired.look(x, y)
	met, found := wired.crowd.first(ray, 0)
	into, enters := entering(wired.fields, ray)
	if enters && (!found || met.dist > into.dist) {
		wired.bent.Store(true)
		return frame.Pixel{}
	}
	if !found {
		return pixelOf(maths.Light{}, 0, float32(math.Inf(1)))
	}
	sides := [3]maths.Ray{ray, wired.look(x+1, y), wired.look(x, y+1)}
	return pixelOf(wired.drawn(met, sides), 1, float32(met.dist))
}

// drawn is the light of a thing met, by what kind of surface it is.
// sides is the pixel's ray and the rays of the pixels to its right and
// below it, which measure a length on the surface in pixels.
func (wired *wiring) drawn(met hit, sides [3]maths.Ray) maths.Light {
	placed := met.thing
	local := placed.at.RayIn(sides[0])
	switch solid := placed.thing.Shape.(type) {
	case *mesh.Mesh:
		one, two, three := solid.Corners(met.on.met.Face())
		return wired.faced(placed, [3]maths.Vec{one, two, three}, true,
			sides)
	case *shape.Mesh:
		_, corners, found := solid.HitCorners(local, 0, nowhere)
		return wired.faced(placed, corners, found, sides)
	case *shape.MeshGroup:
		_, corners, found := solid.HitCorners(local, 0, nowhere)
		return wired.faced(placed, corners, found, sides)
	case *shape.HeightField:
		return wired.celled(solid, met, sides)
	}
	return analytic(placed.thing.Shape)
}

// analytic is the tint of a surface that is not a mesh or a height
// field: a sphere, a plane, a globe, or a shape marched by its
// distance, is analytic; any other shape that answers a ray exactly is
// one this file does not know.
func analytic(solid shape.Shape) maths.Light {
	switch solid.(type) {
	case shape.Sphere, *shape.Sphere, shape.Plane, *shape.Plane,
		globe.Globe, *globe.Globe:
		return wireAnalytic
	}
	if _, exact := solid.(shape.Hitter); exact {
		return wireUnknown
	}
	return wireAnalytic
}

// faced is the light of a mesh's face met by the pixel's ray, given its
// corners in the thing's own space: an edge's light near an edge, the
// face's faint light between. A face the second asking did not find is
// drawn as a shape this file does not know.
func (wired *wiring) faced(placed placedThing, corners [3]maths.Vec,
	found bool, sides [3]maths.Ray) maths.Light {
	if !found {
		return wireUnknown
	}
	for i, corner := range corners {
		corners[i] = placed.at.Out(corner)
	}
	share := wired.inked(edgeGap(corners, sides))
	return maths.Grey(wireFace + (wireLine-wireFace)*share)
}

// inked is how much of a fine pixel an edge covers, from its gap to the
// edge in fine pixels: whole within half the line's width, nothing a
// ramp beyond it.
func (wired *wiring) inked(gap float64) float64 {
	half := wireWidth / 2 * wired.rays
	return min(max((half+wireRamp/2-gap)/wireRamp, 0), 1)
}

// edgeGap is how far, in fine pixels, the pixel's ray meets a face from
// the face's nearest edge. Each corner's weight is nought on the edge
// across from it, and changes from pixel to pixel at a rate the rays
// beside measure on the face's plane; the weight over that rate is the
// gap. A face so edge-on that a ray beside misses its plane is all edge.
func edgeGap(corners [3]maths.Vec, sides [3]maths.Ray) float64 {
	normal := corners[1].Sub(corners[0]).Cross(corners[2].Sub(corners[0]))
	var weighed [3][3]float64
	for i, side := range sides {
		point, met := onPlane(corners[0], normal, side)
		if !met {
			return 0
		}
		weighed[i] = weightsOf(corners, normal, point)
	}
	gap := math.Inf(1)
	for k := range weighed[0] {
		gap = min(gap, lineGap(weighed[0][k], weighed[1][k]-weighed[0][k],
			weighed[2][k]-weighed[0][k]))
	}
	return gap
}

// lineGap is how many pixels a quantity at some value is from nought,
// given how much it changes a pixel across and a pixel down. A quantity
// that does not change is no nearer an edge for it.
func lineGap(value, across, down float64) float64 {
	rate := math.Hypot(across, down)
	if rate > 0 {
		return math.Abs(value) / rate
	}
	return math.Inf(1)
}

// onPlane is where a ray meets the plane through a point square to a
// normal, ahead of the ray, and whether it meets it there.
func onPlane(through, normal maths.Vec, ray maths.Ray) (maths.Vec, bool) {
	along := normal.Dot(through.Sub(ray.At)) / normal.Dot(ray.Dir)
	if !(along > 0 && along < nowhere) {
		return maths.Vec{}, false
	}
	return ray.Point(along), true
}

// weightsOf is a point's weight on each corner of a face it lies in the
// plane of: the area it makes with the edge across from that corner,
// over the face's, signed so a point outside has a negative weight.
func weightsOf(corners [3]maths.Vec, normal, point maths.Vec) [3]float64 {
	whole := normal.Dot(normal)
	var weights [3]float64
	for k := range corners {
		one, two := corners[(k+1)%3], corners[(k+2)%3]
		weights[k] = normal.Dot(one.Sub(point).Cross(two.Sub(point))) /
			whole
	}
	return weights
}

// celled is the light of a height field met by the pixel's ray: the
// analytic tint, with the field's cell edges drawn where its samples
// are, measured in pixels on the plane the field touches there.
func (wired *wiring) celled(field *shape.HeightField, met hit,
	sides [3]maths.Ray) maths.Light {
	placed := met.thing
	point := sides[0].Point(met.dist)
	normal := placed.facing(point, sides[0].Dir)
	var cells [3][2]float64
	for i, side := range sides {
		on, ahead := onPlane(point, normal, side)
		if !ahead {
			return wireCell
		}
		cells[i] = cellOf(field, placed.at.In(on))
	}
	gap := math.Inf(1)
	for axis := range cells[0] {
		value := cells[0][axis] - math.Round(cells[0][axis])
		gap = min(gap, lineGap(value, cells[1][axis]-cells[0][axis],
			cells[2][axis]-cells[0][axis]))
	}
	share := wired.inked(gap)
	return wireAnalytic.Scale(1 - share).Add(wireCell.Scale(share))
}

// cellOf is where a point in a height field's own space is on its grid,
// counted in cells across and down from its corner: whole numbers are
// its sample lines.
func cellOf(field *shape.HeightField, local maths.Vec) [2]float64 {
	_, samples, extent, _, _, _ := field.Pack()
	step := extent / float64(samples-1)
	return [2]float64{(local.X + extent/2) / step,
		(local.Z + extent/2) / step}
}
