package primates

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/bioforms/ocular"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

const (
	// Globe radius follows the adult CT study cited in the model notes.
	globeRadius   = .0121
	cornealBulge  = .00105
	globeHeight   = .979
	domeStart     = .77
	eyeYawLimit   = .40
	eyePitchLimit = .28

	eyeHeight      = 1.561
	eyeDepth       = .0625
	eyeHalfSpacing = .0315

	lidHalfWidth   = .013
	lidMedialShift = .0004
	lidSlant       = .00105
)

type eyeOrbit struct {
	centre maths.Vec
	side   humanSide
}

func (side humanSide) orbit() eyeOrbit {
	return eyeOrbit{side: side, centre: side.at(eyeHalfSpacing, eyeHeight, eyeDepth)}
}

func faceEyeSockets(stations []float64) [2]socket {
	row := func(height float64) int {
		i, _ := slices.BinarySearch(stations, height)
		if stations[i]-height > height-stations[i-1] {
			return i - 1
		}
		return i
	}
	bottom, top := row(1.542), row(1.582)
	return [2]socket{
		{7, 39, bottom, top, humanAround, 1},
		{humanAround - 39, humanAround - 7, bottom, top, humanAround, -1},
	}
}

type lidAperture struct{ upper, lower float64 }

func lidContour(across float64) lidAperture {
	span := math.Max(0, 1-across*across)
	line := eyeHeight + lidSlant*across
	// The upper arch peaks slightly medial to the pupil. The lower
	// margin reaches its lowest point slightly toward the temple.
	return lidAperture{
		upper: line + .00465*math.Sqrt(span)*(1-.45*across),
		lower: line - .00485*math.Pow(span, .72)*(1+.31*across),
	}
}

func (orbit eyeOrbit) lidPosition(angle float64) maths.Vec {
	sine, cosine := math.Sincos(angle)
	across := orbit.side.sign * cosine
	contour := lidContour(across)
	y := contour.upper
	if sine < 0 {
		y = contour.lower
	}
	// The medial corner continues into the lacrimal recess. The temporal
	// corner turns toward the globe a little sooner.
	offset := lidHalfWidth*across - lidMedialShift*across*across
	radius := math.Hypot(offset, (y-orbit.centre.Y)/globeHeight)
	return maths.V(orbit.centre.X+orbit.side.sign*offset, y, lidMarginDepth(radius, offset))
}

func eyelidSkin(x, y float64) maths.Vec {
	ring := headSection(y)
	position := x / ring.w
	cosine := math.Sqrt(math.Max(0, 1-position*position))
	return maths.V(x, y, faceDepth(ring, math.Abs(x), cosine))
}

// Orbital depth belongs to the face chart, including beyond the eyelid
// patch. The lid and cheek share the same slope regardless of tessellation.
func orbitalDepth(wall, x, y float64) float64 {
	across, height := x-eyeHalfSpacing, y-eyeHeight
	radius := math.Hypot(across, height/globeHeight)
	if radius >= .029 {
		return wall
	}
	// The broad support carries tissue into the cheek and brow. The lid
	// mesh seats on the globe inside this continuous facial wall.
	reach := radius / .0146
	cover := eyeDepth + globeRadius + cornealBulge + .00065 - .0074*reach*reach
	span := 1 - maths.Smooth(.008, .0128, math.Abs(across))
	lateral := across / lidHalfWidth
	creaseAt := eyeHeight + .0089 + .0008*lateral - .0030*lateral*lateral
	rollAt := eyeHeight - .0073 + .0010*lateral + .0018*lateral*lateral
	crease := .0009 * bell((y-creaseAt)/.00245)
	roll := .00038 * bell((y-rollAt)/.0025)
	cover += span * (roll - crease)
	// Round the join to the facial wall and taper the orbital work away.
	const transition = .0032
	join := math.Max(0, transition-math.Abs(wall-cover)) / transition
	depth := math.Max(wall, cover) + .25*transition*join*join
	return mix(wall, depth, 1-maths.Smooth(.016, .029, radius))
}

func lidMarginDepth(radius, across float64) float64 {
	// The cornea can turn into the margin near the pupil. Toward the
	// corners the swept cap falls away, leaving the ordinary scleral arc.
	const tangentRadius = .01145
	projected := math.Min(radius, tangentRadius)
	front := globeFront(projected, sweptRadialGlobe)
	tangent := front - (radius-projected)*projected/front
	depth := eyeDepth + .00032 + tangent
	canthus := eyeDepth + mix(.0049, .0032, maths.Smooth(-lidHalfWidth, lidHalfWidth, across))
	const joinWidth = .00105
	join := math.Max(0, joinWidth-math.Abs(depth-canthus)) / joinWidth
	return math.Max(depth, canthus) + .25*joinWidth*join*join
}

// lidMargin names the outer skin strip immediately beside the lid rim.
type lidMargin struct{ first, end int }

func (margin lidMargin) contains(face int) bool {
	return face >= margin.first && face < margin.end
}

func (orbit eyeOrbit) lids(model *humanMesh, rim []int) lidMargin {
	const rows = 20
	edges := make([]maths.Vec, len(rim))
	depths := make([]float64, len(rim))
	for i, index := range rim {
		delta := model.vertices[index].chart.Sub(orbit.centre)
		angle := math.Atan2(delta.Y/.020, delta.X/.025)
		edge := orbit.lidPosition(angle)
		edges[i] = edge
		depths[i] = edge.Z - eyelidSkin(edge.X, edge.Y).Z
	}
	previous := rim
	var margin lidMargin
	for row := 1; row <= rows; row++ {
		share := float64(row) / rows
		seat := maths.Smooth(.22, .96, share)
		next := make([]int, len(rim))
		for i, index := range rim {
			chart := edges[i]
			if row < rows {
				point := mixPoint(model.vertices[index].chart, chart, share)
				chart = eyelidSkin(point.X, point.Y)
				chart.Z += seat * depths[i]
			}
			next[i] = model.headVertex(chart, fixedWeight("head"))
		}
		margin.first = len(model.faces)
		model.connect(previous, next)
		previous = next
	}
	margin.end = len(model.faces)
	orbit.lining(model, previous, edges)
	return margin
}

func (orbit eyeOrbit) lining(model *humanMesh, rim []int, edges []maths.Vec) {
	rolled := make([]int, len(rim))
	contact := make([]int, len(rim))
	for i, edge := range edges {
		delta := edge.Sub(orbit.centre)
		radius := math.Hypot(delta.X, delta.Y/globeHeight)
		// The lid margin turns in to meet the opaque globe. At the canthi
		// it spans the short distance from the corner to the sclera.
		const contactLimit = .01155
		const buried = .00045
		const cornerBlend = .0013
		wanted := .981 * radius
		join := math.Max(0, cornerBlend-math.Abs(wanted-contactLimit)) / cornerBlend
		seated := math.Min(wanted, contactLimit) - .25*cornerBlend*join*join
		inset := seated / radius
		under := orbit.centre.Add(maths.V(delta.X*inset, delta.Y*inset,
			globeFront(seated, radialGlobe)-buried))
		lip := mixPoint(edge, under, .30)
		lip.Z += .00009
		rolled[i] = model.headVertex(lip, fixedWeight("head"))
		contact[i] = model.headVertex(under, fixedWeight("head"))
	}
	model.connect(rim, rolled)
	model.connect(rolled, contact)
	model.cap(contact, model.headVertex(orbit.centre.Add(
		maths.ZAxis.Scale(-.014)), fixedWeight("head")))
}

func radialGlobe(front float64) float64 {
	approach := math.Max(0, (front-domeStart)/(1-domeStart))
	return globeRadius + cornealBulge*approach*approach
}

// The farthest permitted boresight is reached at the combined yaw and
// pitch limits. This cone bounds the turning cap around the neutral axis.
var gazeConeCosine = math.Cos(eyeYawLimit) * math.Cos(eyePitchLimit)
var gazeConeSine = math.Sqrt(1 - gazeConeCosine*gazeConeCosine)

func sweptRadialGlobe(front float64) float64 {
	if front >= gazeConeCosine {
		return radialGlobe(1)
	}
	across := math.Sqrt(1 - front*front)
	return radialGlobe(front*gazeConeCosine + across*gazeConeSine)
}

// globeFront inverts the front of the globe at a radius measured before
// vertical flattening. Radial supplies its neutral or swept profile.
func globeFront(radius float64, radial func(float64) float64) float64 {
	lower, upper := 0.0, 1.0
	for range 16 {
		front := .5 * (lower + upper)
		width := radial(front) * math.Sqrt(1-front*front)
		if width > radius {
			lower = front
		} else {
			upper = front
		}
	}
	front := .5 * (lower + upper)
	return radial(front) * front
}

// EyeAim gives a turn in radians relative to the head. Positive yaw looks
// toward +X, positive pitch toward +Y. Appearance supplies a complete
// ocular texture. Without it, a nonzero Iris gives the base linear RGB
// reflectance, and the zero Iris picks the native brown.
type EyeAim struct {
	Yaw, Pitch float64
	Iris       maths.Light
	Appearance *ocular.Surface
}

// Gaze gives independent turns for the anatomical left and right eyes.
type Gaze struct{ Left, Right EyeAim }

func (aim EyeAim) check() error {
	if !between(aim.Yaw, -eyeYawLimit, eyeYawLimit) ||
		!between(aim.Pitch, -eyePitchLimit, eyePitchLimit) {
		return fmt.Errorf("eye aim outside authored yaw or pitch range")
	}
	if aim.Appearance != nil {
		return nil
	}
	iris := aim.Iris
	if !between(iris.R, 0, 1) || !between(iris.G, 0, 1) || !between(iris.B, 0, 1) {
		return fmt.Errorf("eye iris needs linear reflectance in [0, 1]")
	}
	return nil
}

// Eyes returns the left and right ocular surfaces in the same posed space as
// Surface. Aim turns each globe's corneal relief and optical markings.
func (pose *HumanPose) Eyes(gaze Gaze) ([2]thing.Thing, error) {
	var eyes [2]thing.Thing
	head, _ := pose.Frame("head")
	for i, aim := range [...]EyeAim{gaze.Left, gaze.Right} {
		side := humanSides[i]
		if err := aim.check(); err != nil {
			return [2]thing.Thing{}, fmt.Errorf("adult human %s eye: %w", side.prefix, err)
		}
		eye, err := side.orbit().globe(eyeGlobe{spec: pose.human.spec, head: head, aim: aim})
		if err != nil {
			return [2]thing.Thing{}, fmt.Errorf("adult human %s eye: %w", side.prefix, err)
		}
		eyes[i] = eye
	}
	return eyes, nil
}

type eyeGlobe struct {
	spec HumanSpec
	head skeleton.Frame
	aim  EyeAim
}

func (orbit eyeOrbit) globe(globe eyeGlobe) (thing.Thing, error) {
	const around, bands = 96, 64
	var model humanMesh
	yaw := maths.Rotation(maths.YAxis, globe.aim.Yaw)
	pitch := maths.Rotation(maths.XAxis, -globe.aim.Pitch)
	add := func(normal maths.Vec) int {
		chart := pitch.In(yaw.In(normal))
		// The dome is steeper than the globe at its centre and joins it
		// with a shared tangent. Both turn with the eye's rest chart.
		point := orbit.centre.Add(maths.V(normal.X, globeHeight*normal.Y, normal.Z).
			Scale(radialGlobe(chart.Z)))
		return model.add(humanVertex{
			point: globe.head.Apply(globe.spec.point(headPosition(point))),
			chart: chart.Scale(globeRadius),
		})
	}
	var previous []int
	for row := 1; row < bands; row++ {
		rim := make([]int, around)
		circle, height := math.Sincos(math.Pi * float64(row) / bands)
		for column := range rim {
			sine, cosine := math.Sincos(2 * math.Pi * float64(column) / around)
			rim[column] = add(maths.V(circle*sine, -height, circle*cosine))
		}
		if row == 1 {
			lower := slices.Clone(rim)
			slices.Reverse(lower)
			model.cap(lower, add(maths.YAxis.Neg()))
		} else {
			model.connect(previous, rim)
		}
		previous = rim
	}
	model.cap(previous, add(maths.YAxis))
	appearance := globe.aim.Appearance
	if appearance == nil {
		seed := uint64(1)
		if orbit.side.sign < 0 {
			seed = 2
		}
		spec := ocular.Human(ocular.HumanIris{Seed: seed})
		if tint := globe.aim.Iris; tint != (maths.Light{}) {
			spec.Palette.Stroma, spec.Palette.Inner = tint, tint
		}
		var err error
		appearance, err = ocular.NewSurface(spec)
		if err != nil {
			return thing.Thing{}, err
		}
	}
	return model.ocularSurface(appearance)
}

func (model *humanMesh) ocularSurface(appearance *ocular.Surface) (thing.Thing, error) {
	points, chart := make([]maths.Vec, len(model.vertices)), make([]maths.Vec, len(model.vertices))
	for i, vertex := range model.vertices {
		points[i], chart[i] = vertex.point, vertex.chart
	}
	shape, err := mesh.NewWithRestPoints(points, chart, model.faces,
		geometry.AreaNormals(points, model.faces), true)
	if err != nil {
		return thing.Thing{}, err
	}
	return thing.Thing{Shape: shape, Material: &thing.Material{
		Name: "adult human eye", Texture: appearance,
		Gloss: .86, Index: 1.337, Satin: .25,
	}}, nil
}
