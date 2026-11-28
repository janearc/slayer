package primates

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// AuricleForm shapes the external ears of the reference before stature scaling.
// Height, Width and Projection are in metres. Projection sets the peak lift of
// the rear wall above the skull. BackTilt is in radians. Positive Asymmetry
// makes the left ear larger and the right ear smaller. Zero uses reference
// values, except for Asymmetry, where zero is symmetric.
type AuricleForm struct {
	Height, Width, Projection, BackTilt, Asymmetry float64
}

// WithAuricles replaces the lateral head relief with connected external ears.
func (spec HumanSpec) WithAuricles(form AuricleForm) HumanSpec {
	spec.auricles = &form
	return spec
}

func (form AuricleForm) resolved() (AuricleForm, error) {
	for _, control := range [...]struct {
		set                     *float64
		reference, lower, upper float64
	}{
		{&form.Height, .062, .058, .071},
		{&form.Width, .033, .030, .039},
		{&form.Projection, .018, .013, .024},
		{&form.BackTilt, .235, .14, .32},
		{&form.Asymmetry, 0, -.035, .035},
	} {
		if *control.set == 0 {
			*control.set = control.reference
		}
		if !between(*control.set, control.lower, control.upper) {
			return AuricleForm{}, fmt.Errorf("adult human: auricle dimensions or asymmetry outside range")
		}
	}
	return form, nil
}

func auricleSockets(stations []float64) [2]socket {
	bottom := sort.SearchFloat64s(stations, 1.539)
	top := sort.SearchFloat64s(stations, 1.558)
	return [2]socket{
		{70, 76, bottom, top, humanAround, 1},
		{humanAround - 76, humanAround - 70, bottom, top, humanAround, -1},
	}
}

const (
	auricleY = 1.547
	auricleZ = -.010
)

type auricle struct {
	side                              humanSide
	halfHeight, halfWidth, tilt, lift float64
}

func (side humanSide) auricle(form AuricleForm) auricle {
	scale := 1 + side.sign*form.Asymmetry
	return auricle{side: side, halfHeight: .5 * form.Height * scale,
		halfWidth: .5 * form.Width * scale, tilt: math.Tan(form.BackTilt),
		lift: form.Projection * scale}
}

type auricleRay struct {
	sine, cosine            float64
	root, lip, round, reach float64
}

// The head opening supplies the angles and inner radii. The same rays
// address both sides of the ear and the arc that rolls around the margin.
func (ear auricle) ray(root maths.Vec) auricleRay {
	y := root.Y - auricleY
	sine := y / ear.halfHeight
	cosine := (root.Z - auricleZ + ear.tilt*y) / ear.halfWidth
	radius := math.Hypot(sine, cosine)
	sine, cosine = sine/radius, cosine/radius
	back := maths.Smooth(-.15, .83, -cosine)
	crown := bell((sine - .49) / .48)
	pinch := bell((sine + .51) / .20)
	lobule := maths.Smooth(.70, .96, -sine)
	outline := 1 + .10*sine + .10*back*crown - .14*pinch - .052*lobule
	outline -= .11 * maths.Smooth(.35, .95, cosine) * bell((sine+.25)/.31)
	dy := ear.halfHeight * sine
	dz := ear.halfWidth*cosine - ear.tilt*dy
	round := .00145 / math.Hypot(dy, dz)
	reach := ear.lift * (.47 + .53*maths.Smooth(-.87, .47, sine)) *
		(.20 + .80*maths.Smooth(-.64, .73, -cosine))
	return auricleRay{sine: sine, cosine: cosine, root: radius,
		lip: outline - round, round: round, reach: reach}
}

// The rear annulus starts on the head rim and acquires its clearance
// gradually. Its shared outer ring turns onto the front of the pinna.
func (ear auricle) grow(model *humanMesh, boundary []int) {
	const rearRows, rollRows, frontRows = 12, 10, 42
	rays := make([]auricleRay, 2*len(boundary))
	for i, index := range boundary {
		here := model.vertices[index].chart
		next := model.vertices[boundary[(i+1)%len(boundary)]].chart
		rays[2*i] = ear.ray(here)
		rays[2*i+1] = ear.ray(mixPoint(here, next, .5))
	}
	weight := fixedWeight("head")
	addRing := func(profile func(auricleRay) (radius, depth float64)) []int {
		ring := make([]int, len(rays))
		for i, ray := range rays {
			radius, depth := profile(ray)
			ring[i] = model.headVertex(ear.point(ray, radius, depth), weight)
		}
		return ring
	}
	rear := func(share float64) []int {
		return addRing(func(ray auricleRay) (float64, float64) {
			return mix(ray.root, ray.lip, share), 0
		})
	}
	rim := rear(1.0 / rearRows)
	model.joinAuricleRoot(boundary, rim)
	for row := 2; row <= rearRows; row++ {
		next := rear(float64(row) / rearRows)
		model.connect(rim, next)
		rim = next
	}
	ring := func(profile func(auricleRay) (radius, depth float64)) {
		next := addRing(profile)
		model.connect(rim, next)
		rim = next
	}
	for row := 1; row <= rollRows; row++ {
		sine, cosine := math.Sincos(math.Pi * float64(row) / rollRows)
		ring(func(ray auricleRay) (float64, float64) {
			return ray.lip + ray.round*sine, .5 * (1 - cosine) * ray.thickness(ray.lip)
		})
	}
	for row := 1; row < frontRows; row++ {
		share := 1 - float64(row)/frontRows
		ring(func(ray auricleRay) (float64, float64) {
			radius := ray.lip * share
			return radius, ray.thickness(radius)
		})
	}
	centre := auricleRay{lip: 1}
	model.cap(rim, model.headVertex(ear.point(centre, 0, centre.thickness(0)), weight))
}

// joinAuricleRoot doubles the angular sampling past the skull seam. The
// three triangles keep the unsplit edge shared with the surrounding skin.
func (model *humanMesh) joinAuricleRoot(root, ring []int) {
	for i, here := range root {
		next := (i + 1) % len(root)
		middle := ring[2*i+1]
		model.faces = append(model.faces, [3]int{here, root[next], middle},
			[3]int{here, middle, ring[2*i]}, [3]int{root[next], ring[2*next], middle})
	}
}

func (ear auricle) point(ray auricleRay, radius, depth float64) maths.Vec {
	y := radius * ray.sine * ear.halfHeight
	z := radius*ray.cosine*ear.halfWidth - ear.tilt*y
	y, z = auricleY+y, auricleZ+z
	gap := ray.reach * maths.Smooth(ray.root, ray.lip, radius)
	return ear.side.at(auricleWall(y, z)+gap+depth, y, z)
}

func auricleWall(y, z float64) float64 {
	pointAt := func(height float64) maths.Vec {
		skull := headSection(height)
		cosine := (2*z - skull.front + skull.back) / (skull.front + skull.back)
		sine := math.Sqrt(1 - cosine*cosine)
		return mandibularShape(maths.V(skull.w*sine, height, z), sine, cosine)
	}
	wall := pointAt(y)
	if wall.Y == y {
		return wall.X
	}
	// The ramus raises the wall below the attachment. Recover the
	// original station at this chart height before adding ear depth.
	lower, upper := y-.015, y
	for range 13 {
		middle := (lower + upper) / 2
		wall = pointAt(middle)
		if wall.Y < y {
			lower = middle
		} else {
			upper = middle
		}
	}
	return wall.X
}

func (ray auricleRay) thickness(radius float64) float64 {
	u, v := radius*ray.cosine, radius*ray.sine
	margin := radius / ray.lip
	helix := .00205 * bell((margin-.91)/.085) * maths.Smooth(-.58, -.33, v) *
		(1 - .87*maths.Smooth(.30, .76, u))
	concha := .00195 * bell((u-.12+.16*v)/.32) * bell((v+.10)/.39)
	crest := -.27 - .14*bell((v-.24)/.43)
	antihelix := .0037 * bell((u-crest)/.13) * bell((v-.16)/.50)
	crus := .0026 * bell((u-(-.52+.94*v))/.11) * bell((v-.39)/.22) *
		maths.Smooth(.08, .25, v)
	tragus := .0036 * bell((u-.48)/.14) * bell((v+.16)/.17)
	antitragus := .0022 * bell((u+.12)/.17) * bell((v+.44)/.14)
	intertragic := .0010 * bell((u-.23)/.14) * bell((v+.42)/.13)
	lobule := .00145 * bell((u-.02)/.47) * bell((v+.72)/.22)
	meatus := .0011 * bell((u-.23)/.12) * bell((v+.23)/.12)
	anterior := .00185 * maths.Smooth(.54, .91, u) * maths.Smooth(.68, .96, margin)
	return .00375 + helix + antihelix + crus + tragus + antitragus + lobule -
		concha - intertragic - meatus - anterior
}
