package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// keckTube replaces the generic round cage with an angular, triangulated
// space frame. Its proportions are photographic scenery choices, not CAD.
func (row groundObservatory) keckTube(assembly *instrumentAssembly,
	pose maths.Subspace, radius, length float64,
) {
	defer assembly.detailSince(len(assembly.object.parts))
	assembly.object.dressed[leaf] = groundPaint("Keck pale space frame", "keck-frame-reflectance")
	middle := length * groundValue("keck-middle-frame-height")
	upper := row.aperture * groundValue("keck-upper-frame-radius")
	girth := row.aperture * groundValue("keck-frame-member-radius")
	levels := []float64{0, middle, length}
	radii := []float64{radius, radius, upper}
	for level, height := range levels {
		for corner := range octagonSides {
			from := keckCorner(corner, radii[level], height)
			to := keckCorner(corner+1, radii[level], height)
			keckMember(assembly, pose, from, to, girth)
			if level == 0 {
				continue
			}
			back := keckCorner(corner, radii[level-1], levels[level-1])
			diagonal := keckCorner(corner+1, radii[level-1], levels[level-1])
			keckMember(assembly, pose, back, from, girth)
			keckMember(assembly, pose, diagonal, from, girth)
		}
	}
	row.keckSecondaryCage(assembly, pose, upper, length)
}

// keckCorner places a vertex of the authored octagonal frame in optical axes.
func keckCorner(corner int, radius, height float64) maths.Vec {
	angle := fullTurn * (float64(corner) + spaceHalf) / octagonSides
	return maths.V(radius*math.Cos(angle), height, radius*math.Sin(angle))
}

// keckMember leaves a small joint gap between separately closed members.
func keckMember(assembly *instrumentAssembly, pose maths.Subspace,
	from, to maths.Vec, radius float64,
) {
	gap := to.Sub(from).Unit().Scale(groundValue("rib-joint-metres"))
	assembly.rod(pose.Out(from.Add(gap)), pose.Out(to.Sub(gap)), radius, leaf)
}

// keckSecondaryCage gives the top end depth, diagonal bracing and a spider.
// The existing secondary mirror remains at the same optical station.
func (row groundObservatory) keckSecondaryCage(assembly *instrumentAssembly,
	pose maths.Subspace, outer, length float64,
) {
	radius := row.aperture * groundValue("keck-secondary-cage-radius")
	depth := row.aperture * groundValue("keck-secondary-cage-depth")
	girth := row.aperture * groundValue("keck-secondary-member-radius")
	for corner := range octagonSides {
		front := keckCorner(corner, radius, length)
		next := keckCorner(corner+1, radius, length)
		back := front.Sub(maths.YAxis.Scale(depth))
		backNext := next.Sub(maths.YAxis.Scale(depth))
		keckMember(assembly, pose, front, next, girth)
		keckMember(assembly, pose, back, backNext, girth)
		keckMember(assembly, pose, back, front, girth)
		keckMember(assembly, pose, backNext, front, girth)
		if corner%2 == 0 {
			keckMember(assembly, pose, keckCorner(corner, outer, length), front, girth)
			keckMember(assembly, pose, keckCorner(corner, outer, length), back, girth)
		}
	}
}
