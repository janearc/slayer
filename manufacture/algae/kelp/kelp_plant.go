package kelp

import (
	"math"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// kelpPlant is one plant being grown: the shared meshes, its seed, where
// its holdfast is, the water it stands in and the time in seconds.
type kelpPlant struct {
	book  *kelpBook
	kit   *kelpKit
	from  branching.Seed
	at    maths.Vec
	water kelpCondition
	clock float64
}

// depth is how far the sea's surface is over the holdfast.
func (plant kelpPlant) depth() float64 {
	return plant.book.pick("depth", plant.from, 0)
}

// top is how high the plant's stipes reach, short of the surface by the
// plant's drop.
func (plant kelpPlant) top() float64 {
	return plant.depth() - plant.book.pick("drop", plant.from, 0)
}

// lean is the share of its greatest lean the current gives: half at the
// file's reference current, nothing in still water.
func (plant kelpPlant) lean() float64 {
	reference := plant.book.constantOf("lean-reference")
	return plant.water.Current / (plant.water.Current + reference)
}

// sway is how far the swell's orbit has taken the water, and the
// plant with it, along x at a height. The orbit of a wave of height H
// and wavelength L is H over two at the surface and dies as exp(-k z)
// with the depth z, k being two pi over L; L is g T squared over two pi.
func (plant kelpPlant) sway(height float64) float64 {
	water := plant.water
	if water.SwellHeight <= 0 {
		return 0
	}
	gravity := plant.book.constantOf("gravity")
	wavelength := gravity * water.SwellPeriod * water.SwellPeriod / (2 * math.Pi)
	wave := (2 * math.Pi) / wavelength
	orbit := water.SwellHeight * kelpHalf *
		math.Exp(-wave*(plant.depth()-height))
	phase := (2*math.Pi)*plant.clock/water.SwellPeriod - wave*plant.at.X
	return plant.book.constantOf("follow") * orbit * math.Cos(phase)
}

// drift is how far the current and the swell have taken a point of a
// stipe at a height from where it would hang in still water. The foot
// stays where it is.
func (plant kelpPlant) drift(height float64) maths.Vec {
	share := height / plant.top()
	lean := plant.lean() * plant.book.constantOf("lean-share") *
		plant.top() * share * share
	return maths.V(lean+plant.sway(height)-plant.sway(0), 0, 0)
}

// grow adds the plant's placements to the members: a holdfast, and each
// stipe with its blades and bladders.
func (plant kelpPlant) grow(into *kelpMembers) {
	plant.holdfast(into)
	for stipe := range plant.book.count("stipes", plant.from, 0) {
		path, canopy := plant.path(stipe)
		plant.stipe(into, path)
		plant.leaves(into, stipe, path, canopy)
	}
}

// holdfast adds the fingers that grip the floor, outward and a little up
// from the plant's foot.
func (plant kelpPlant) holdfast(into *kelpMembers) {
	book := plant.book
	lobes := book.count("holdfast-lobes", plant.from, 0)
	radius := book.pick("holdfast-radius", plant.from, 0)
	foot := plant.at.Add(maths.V(0, kelpGround, 0))
	for lobe := range lobes {
		turn := (2 * math.Pi) * (float64(lobe) +
			plant.from.Draw(1, lobe, 2)) / float64(lobes)
		rise := kelpRise * plant.from.Draw(1, lobe, 3)
		way := maths.V(math.Cos(turn)*math.Cos(rise), math.Sin(rise),
			math.Sin(turn)*math.Cos(rise))
		size := radius * (kelpFingerLeast +
			(1-kelpFingerLeast)*plant.from.Draw(1, lobe, 4))
		into.stipes = append(into.stipes, shape.MeshMember{
			Mesh:  plant.kit.finger,
			Space: maths.NewSubspace(foot, maths.Upright(way), size)})
	}
}

// path is the corners of a stipe, from its foot to the end of its run
// along the surface, and the place in the path where that run starts.
// The stipe rises by the file's segment, fanned outward from the
// plant's middle and drifting with the water, and at its top it turns
// and runs level for the canopy's reach.
func (plant kelpPlant) path(stipe int) ([]maths.Vec, int) {
	book := plant.book
	turn := kelpGolden*float64(stipe) +
		(2*math.Pi)*plant.from.Draw(1, stipe, 0)
	out := maths.V(math.Cos(turn), 0, math.Sin(turn))
	start := plant.at.Add(out.Scale(book.pick("spread", plant.from, stipe)))
	fan := book.pick("fan", plant.from, stipe)
	top, segment := plant.top(), book.middle("segment")
	var path []maths.Vec
	for height := 0.0; ; height += segment {
		height = math.Min(height, top)
		spread := out.Scale(fan * height / top)
		path = append(path, start.Add(maths.V(0, height, 0)).Add(spread).
			Add(plant.drift(height)))
		if height >= top {
			break
		}
	}
	canopy := len(path) - 1
	way := out.Add(maths.V(plant.lean(), 0, 0)).Unit()
	reach := book.pick("canopy-reach", plant.from, stipe)
	for run := segment; run <= reach; run += segment {
		path = append(path, path[canopy].Add(way.Scale(run)))
	}
	return path, canopy
}

// stipe adds the stipe's straight runs, each a shared cone of the girth
// its place along the path gives, stretched to the run's length.
func (plant kelpPlant) stipe(into *kelpMembers, path []maths.Vec) {
	segment := plant.book.middle("segment")
	for index := 0; index+1 < len(path); index++ {
		run := path[index+1].Sub(path[index])
		class := min(kelpClasses-1, index*kelpClasses/len(path))
		into.stipes = append(into.stipes, shape.MeshMember{
			Mesh: plant.kit.stipes[class],
			Space: maths.NewSubspace(path[index], maths.Upright(run),
				run.Len()/segment)})
	}
}
