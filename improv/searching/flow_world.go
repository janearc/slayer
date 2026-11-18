package searching

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// flowSegment retains one mesh between adjacent path samples. Only its
// placement and material change; frame order cannot change its next pose.
type flowSegment struct {
	object     *thing.Thing
	from, to   maths.Vec
	start, end float64
	path       int
}

// flowSet keeps the authored filaments and fixed reference points apart.
type flowSet struct {
	world    *thing.World
	segments []flowSegment
	opening  *thing.Thing
	home     *thing.Thing
}

// newFlow builds actual GPU-supported geometry, with no projected paint or
// invented catalogue provenance. All display units belong to the recipe.
func newFlow(plan *FlowPlan) (*flowSet, error) {
	set := &flowSet{world: thing.NewWorld()}
	for _, cloud := range plan.Clouds {
		set.world.AddVolume(thing.Volume{
			Medium: medium.Wispy{Of: medium.Puff{Radius: cloud.Radius,
				Thick: cloud.Density}, Scale: 0.8, Amount: 1, Seed: cloud.Seed},
			Material: &thing.Material{Name: "authored diffuse field",
				Glow: maths.RGB(cloud.RGB[0], cloud.RGB[1], cloud.RGB[2])},
			Motion: thing.Still(maths.Subspace{Origin: vector(cloud.At)})})
	}
	for index, path := range plan.Paths {
		along := fractions(path.Points)
		for point := 1; point < len(path.Points); point++ {
			from, to := vector(path.Points[point-1]), vector(path.Points[point])
			mesh, err := flowTube(path.Width / to.Sub(from).Len())
			if err != nil {
				return nil, err
			}
			object := set.world.Add(thing.Thing{Shape: mesh,
				Material: &thing.Material{Name: "authored flow filament"}})
			set.segments = append(set.segments, flowSegment{object: object,
				from: from, to: to, start: along[point-1], end: along[point], path: index})
		}
	}
	for _, point := range plan.Landmarks {
		set.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Name: "authored fixed landmark",
				Glow: maths.RGB(point.RGB[0], point.RGB[1], point.RGB[2])},
			Motion: thing.Still(maths.Subspace{Origin: vector(point.At), Scale: point.Radius})})
	}
	mesh, err := flowOpening()
	if err != nil {
		return nil, err
	}
	set.opening = set.world.Add(thing.Thing{Shape: mesh,
		Material: &thing.Material{Name: "authored dark opening"}})
	set.home = set.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "authored return destination"},
		Motion: thing.Still(maths.Subspace{
			Origin: vector(plan.Paths[plan.ReturnPath].Points[0]), Scale: 0.13})})
	return set, nil
}

// flowTube is an open four-sided sleeve from zero to minus z. Its radius
// is divided by the rest length so uniform instance scale restores width.
// It is a display filament, not a physical cable or a plasma simulation.
func flowTube(radius float64) (*shape.Mesh, error) {
	vertices := make([]maths.Vec, 0, 8)
	for _, depth := range []float64{0, -1} {
		for corner := range 4 {
			angle := float64(corner) * math.Pi / 2
			vertices = append(vertices, maths.V(radius*math.Cos(angle),
				radius*math.Sin(angle), depth))
		}
	}
	faces := make([][3]int, 0, 8)
	for corner := range 4 {
		next := (corner + 1) % 4
		faces = append(faces, [3]int{corner, next, next + 4},
			[3]int{corner, next + 4, corner + 4})
	}
	return shape.NewMesh(vertices, faces, nil, false)
}

// flowOpening makes an asymmetric silhouette rather than a circular eye.
// These dimensions and harmonics are authored compositional choices.
func flowOpening() (*shape.Mesh, error) {
	vertices := []maths.Vec{{}}
	const corners = 48
	for index := range corners {
		angle := float64(index) * 2 * math.Pi / corners
		radius := 1 + 0.12*math.Sin(3*angle) + 0.07*math.Cos(7*angle)
		vertices = append(vertices, maths.V(4.2*radius*math.Cos(angle),
			1.6*radius*math.Sin(angle), 0))
	}
	faces := make([][3]int, corners)
	for index := range corners {
		faces[index] = [3]int{0, index + 1, (index+1)%corners + 1}
	}
	return shape.NewMesh(vertices, faces, nil, false)
}

// deform gathers only the inner path into an authored fold. Endpoints
// shared by adjacent segments receive the same transform, avoiding cracks.
func deform(point maths.Vec, along, fold float64) maths.Vec {
	weight := smooth((along - 0.45) / 0.55)
	turn := maths.Rotation(maths.ZAxis, fold*weight*0.5)
	point = turn.Out(point)
	point.Z += fold * weight * (1.5 + 0.7*math.Sin(point.X))
	return point
}

// update computes geometry and travelling emission from absolute beat time.
// The fixed landmark field is never moved by the fold or the return pulse.
func (set *flowSet) update(edit *Edit, beat float64) {
	plan := edit.Plan.FlowStudy
	start := edit.Grid.BeatAt(edit.FlowStart)
	foldBeat := edit.Grid.BeatAt(edit.FlowFold)
	answer := edit.Grid.BeatAt(edit.FlowAnswer)
	set.updateClock(plan, beat, start, foldBeat, answer)
}

// updateClock lets a new score preserve the selected reference geometry
// while explicitly retiming its travelling light and twisting opening.
func (set *flowSet) updateClock(plan *FlowPlan, beat, start, foldBeat, answer float64) {
	fold := smooth((beat - foldBeat) / 4)
	established := smooth((beat - start) / 4)
	for _, segment := range set.segments {
		path := plan.Paths[segment.path]
		from := deform(segment.from, segment.start, fold)
		to := deform(segment.to, segment.end, fold)
		segment.object.Motion = thing.Still(maths.Subspace{Origin: from,
			Orientation: maths.LookAt(from, to), Scale: to.Sub(from).Len()})
		position := (segment.start + segment.end) / 2
		// Each four-beat phrase sends one packet inward over four beats.
		// A delay staggers paths; pulse tails vanish before the next launch.
		light := 0.0
		for launch := start + 4 + path.Delay; launch < answer; launch += 4 {
			light += packet(position, (beat-launch)/4, 0.1)
		}
		back := 0.0
		if segment.path == plan.ReturnPath && beat >= answer {
			back = packet(position, 1-(beat-answer)/6, 0.12)
		}
		base := established * (0.035 + 0.08*position)
		segment.object.Material.Glow = maths.RGB(
			base*0.9+light*2.8+back*5,
			base*0.4+light*0.9+back*3,
			base*1.7+light*0.16+back*1.2)
	}
	set.opening.Motion = thing.Still(maths.Subspace{
		Origin: maths.V(0, 0, 7), Scale: 0.02 + fold*0.98,
		Orientation: maths.Rotation(maths.ZAxis, 0.3+fold*0.5)})
	arrived := smooth((beat - answer - 5.5) / 0.5)
	set.home.Material.Glow = maths.RGB(0.1+arrived*4, 0.8+arrived*2, 1.6)
}
