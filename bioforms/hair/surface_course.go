package hair

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// SurfaceCourseSpec sets the length sampled along a direction field
// and then through an authored fall. Length, Contact and Lift are
// metres. Fall is a rest-model direction carried by the release face.
type SurfaceCourseSpec struct {
	Segments              int
	Length, Contact, Lift float64
	Fall                  maths.Vec
	Envelope              CourseEnvelope
}

// CourseEnvelope shapes the free segment in rest-model metres.
// Shoulder offsets its middle and Tip offsets its end.
type CourseEnvelope struct {
	Shoulder, Tip maths.Vec
}

// SurfaceCourse preserves the posed skin attachment of each contact
// sample. Free samples follow the frame of the last contact triangle.
// The zero value uses the strand's ordinary guide.
type SurfaceCourse struct {
	rings []courseRing
}

type courseRing struct {
	pin   GuidePin
	local maths.Vec
}

const courseStep = .003

// Course traces a bounded midpoint path in the authored chart.
// At a weak direction or domain boundary it releases from the last
// attached position. The outgoing path bends from the same tangent
// toward Fall. It describes a static style.
func (field *TangentField) Course(root Site, spec SurfaceCourseSpec) (SurfaceCourse, error) {
	if spec.Segments < 2 || spec.Segments > 64 || !(spec.Length > 0 && spec.Length <= 4) ||
		!(spec.Contact >= 0 && spec.Contact <= spec.Length) || !(spec.Lift >= 0 && finite(spec.Lift)) ||
		!finite(spec.Envelope.Shoulder.Len()) || !finite(spec.Envelope.Tip.Len()) ||
		!(spec.Fall.Len() > leastVector && finite(spec.Fall.Len())) ||
		root.Face < 0 || root.Face >= len(field.faces) {
		return SurfaceCourse{}, fmt.Errorf("hair: invalid surface course")
	}

	course := SurfaceCourse{rings: make([]courseRing, spec.Segments)}
	domain := field.domain(root.Chart)
	at := root
	heading := field.direction(root, domain)
	lastDirection := heading
	travelled := 0.
	released := false
	var fall fallingCourse
	for i := range course.rings {
		target := float64(i+1) * (spec.Length / float64(spec.Segments))
		contact := min(spec.Contact, target)
		for !released && contact-travelled > leastVector {
			step := min(courseStep, contact-travelled)
			next, ok := field.advance(at, heading, domain, step)
			if !ok {
				released = true
				break
			}
			lastDirection = next.Chart.Sub(at.Chart)
			at = next
			heading = field.direction(at, domain)
			travelled += step
		}
		lift := spec.Lift * maths.Smooth(0, min(.045, max(courseStep, spec.Contact)), travelled)
		pin := GuidePin{face: at.Face, corners: field.faces[at.Face], weights: at.Weights, height: lift, share: 1}
		if target-travelled <= leastVector {
			course.rings[i] = courseRing{pin: pin}
			continue
		}
		if fall == (fallingCourse{}) {
			entry := tangentTo(lastDirection, at.ChartNormal).Unit()
			fall = field.release(at, entry, spec, spec.Length-travelled)
		}
		released = true
		course.rings[i] = courseRing{pin: pin, local: fall.at(target - travelled)}
	}
	return course, nil
}

func (field *TangentField) advance(from Site, heading maths.Vec, domain int, step float64) (Site, bool) {
	if domain < 0 || heading.Len() < weakTangent {
		return Site{}, false
	}
	direction := heading.Unit()
	middle, ok := field.project(from.Chart.Add(direction.Scale(.5*step)), domain, step)
	if !ok || field.blocked(from.Chart, middle.Chart) {
		return Site{}, false
	}
	midHeading := field.direction(middle, domain)
	if midHeading.Len() < weakTangent || midHeading.Dot(direction) <= 0 {
		return Site{}, false
	}
	next, ok := field.project(from.Chart.Add(midHeading.Unit().Scale(step)), domain, step)
	return next, ok && !field.blocked(middle.Chart, next.Chart) &&
		!field.blocked(from.Chart, next.Chart) && next.ChartNormal.Dot(from.ChartNormal) > .55 &&
		next.Chart.Sub(from.Chart).Dot(direction) > .20*step
}

type fallingCourse struct {
	entry, down, shift, shoulder, tip maths.Vec
	turned, travel, length            float64
}

func (field *TangentField) release(site Site, heading maths.Vec, spec SurfaceCourseSpec, length float64) fallingCourse {
	face := field.faces[site.Face]
	chart, real := edges(field.points, face), edges(field.rest, face)
	frame := courseFrame(real)
	entry := real.flow(chart.coordinates(heading)).Unit()
	if heading.Len() < weakTangent {
		entry = tangentTo(spec.Fall, frame.up).Unit()
	}
	if entry == (maths.Vec{}) {
		entry = frame.along
	}
	local := func(direction maths.Vec) maths.Vec {
		return maths.V(direction.Dot(frame.along), direction.Dot(frame.up), direction.Dot(frame.side))
	}
	return fallingCourse{
		entry: local(entry), down: local(spec.Fall.Unit().Add(frame.up.Scale(.12)).Unit()),
		shoulder: local(spec.Envelope.Shoulder), tip: local(spec.Envelope.Tip),
		turned: max(courseStep, min(.055, .57*length)), length: length,
	}
}

func (fall *fallingCourse) at(distance float64) maths.Vec {
	for distance-fall.travel > leastVector {
		step := min(courseStep, distance-fall.travel)
		share := maths.Smooth(0, fall.turned, fall.travel+.5*step)
		up := maths.V(0, .22*math.Sin(math.Pi*share), 0)
		tangent := fall.entry.Scale(1 - share).Add(fall.down.Scale(share)).Add(up).Unit()
		fall.shift = fall.shift.Add(tangent.Scale(step))
		fall.travel += step
	}
	if fall.shoulder == (maths.Vec{}) && fall.tip == (maths.Vec{}) {
		return fall.shift
	}
	t := min(1, distance/fall.length)
	bow := 4 * t * (1 - t)
	return fall.shift.Add(fall.shoulder.Scale(bow * bow)).
		Add(fall.tip.Scale(maths.Smooth(.24, 1, t)))
}

func courseFrame(basis triangle) shaftFrame {
	along, up := basis.first.Unit(), basis.area().Unit()
	return shaftFrame{along, up, up.Cross(along)}
}

func (course SurfaceCourse) valid(faces [][3]int, segments int) bool {
	if len(course.rings) == 0 {
		return true
	}
	if len(course.rings) != segments {
		return false
	}
	for _, ring := range course.rings {
		if !ring.pin.valid(faces) {
			return false
		}
	}
	return true
}

func (ring courseRing) posed(points, normals []maths.Vec) maths.Vec {
	pin := ring.pin
	basis := edges(points, pin.corners)
	site := locate(points, normals, pin.face, pin.corners, pin.weights, basis)
	return site.Point.Add(site.Normal.Scale(pin.height)).Add(courseFrame(basis).out(ring.local))
}
