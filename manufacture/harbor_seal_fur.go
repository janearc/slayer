package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
)

// The coat as geometry: short hair as thin blades, one strand each, that
// root on the skin, lie along the coat's flow and rise from it by a lift.
// Wet hair lies flatter and clumps into fewer, wider blades. Every number
// is chosen to read as a harbor seal's coat of a centimetre or so of hair.
const (
	// sealFurSpacing is the mean distance between blade roots on a dry
	// coat, in metres, and sealFurWetSpacing how much a soaked coat adds.
	sealFurSpacing, sealFurWetSpacing = .0058, .0025
	// sealFurLength is a dry blade's length, and sealFurWetLength how much
	// shorter a soaked one lies.
	sealFurLength, sealFurWetLength = .0140, .0030
	// sealFurWidth is a dry blade's width at its root, sealFurWetWidth how
	// much wider a soaked one clumps.
	sealFurWidth, sealFurWetWidth = .0016, .0020
	// sealFurLift is a dry blade's rise from the skin in radians, and
	// sealFurWetLift how much lower a soaked one lies.
	sealFurLift, sealFurWetLift = .50, .40
	// sealFurTip is the share of the lift a blade keeps toward its tip,
	// where it bends back to lie down.
	sealFurTip = .35
	// sealFurScatter is the spread, in radians, of a blade's yaw from the
	// flow, and sealFurVary the spread of its length as a share.
	sealFurScatter, sealFurVary = .6, .25
	// sealFurNose is the X from which the nose pad is bare skin.
	sealFurNose = .805
	// sealFurEye and sealFurEar are the radii, in metres, of the bare
	// skin around an eye and an ear opening.
	sealFurEye, sealFurEar = .035, .016
	// sealFurBase and sealFurRibbon weigh a blade vertex's shading toward
	// the skin's normal and toward its own face.
	sealFurBase, sealFurRibbon = .6, .4
	// sealFurMid is how much narrower a blade is half way along.
	sealFurMid = .75
)

// sealFurBlade is the shape of one blade, set by how wet the coat is.
type sealFurBlade struct{ spacing, length, width, lift float64 }

// sealFurFor is the fur blade of a coat wet by a share from nought to one:
// each figure of the dry blade moves by its wet constant in that share.
func sealFurFor(wet float64) sealFurBlade {
	return sealFurBlade{
		spacing: sealFurSpacing + sealFurWetSpacing*wet,
		length:  sealFurLength - sealFurWetLength*wet,
		width:   sealFurWidth + sealFurWetWidth*wet,
		lift:    sealFurLift - sealFurWetLift*wet,
	}
}

// sealFurCloth gathers corners, rest points, normals and faces of every
// blade into the arrays one mesh is made from.
type sealFurCloth struct {
	corners, rest, normals []maths.Vec
	faces                  [][3]int
}

// bareDisc is a round patch of the rest skin that grows no hair.
type bareDisc struct {
	centre maths.Vec
	radius float64
}

// bareDiscs finds the eyes and ear openings on the rest skin.
func (rig *SealRig) bareDiscs() ([]bareDisc, error) {
	var discs []bareDisc
	for _, side := range [...]float64{-1, 1} {
		for _, spot := range [...]struct{ x, y, radius float64 }{
			{sealEyeX, sealEyeY, sealFurEye}, {sealEarX, sealEarY, sealFurEar},
		} {
			point, _, found := rig.restHit(maths.V(spot.x, spot.y, side*sealProbe),
				maths.V(0, 0, -side))
			if !found {
				return nil, fmt.Errorf("harbor seal: no skin at bare patch %v", spot)
			}
			discs = append(discs, bareDisc{point, spot.radius})
		}
	}
	return discs, nil
}

// bareAt is whether a point has no fur: past the nose, or inside one of
// the bare discs.
func bareAt(discs []bareDisc, point maths.Vec) bool {
	if point.X > sealFurNose {
		return true
	}
	for _, disc := range discs {
		if point.Sub(disc.centre).Len() < disc.radius {
			return true
		}
	}
	return false
}

// Fur makes the coat's hair for a pose as one open mesh of blades in the
// posed space. Each blade's rest points are its root's, so the pelage's
// paint is read where the blade grows. Hair does not grow on the nose
// pad, around the eyes and ear openings, or on the digit tips.
func (rig *SealRig) Fur(pose SealRigPose, look SealLook) (*mesh.Mesh, error) {
	if err := pose.check(); err != nil {
		return nil, err
	}
	if err := look.check(); err != nil {
		return nil, err
	}
	discs, err := rig.bareDiscs()
	if err != nil {
		return nil, err
	}
	points := rig.posed(pose)
	normals := make([]maths.Vec, len(points))
	for index, normal := range sealSmoothNormals(points, rig.faces) {
		normals[index] = normal.Unit()
	}
	blade := sealFurFor(look.Wet)
	var cloth sealFurCloth
	for index, triangle := range rig.faces {
		if rig.bare[triangle[0]] || rig.bare[triangle[1]] || rig.bare[triangle[2]] {
			continue
		}
		area := points[triangle[1]].Sub(points[triangle[0]]).
			Cross(points[triangle[2]].Sub(points[triangle[0]])).Len() / 2
		expected := area / (blade.spacing * blade.spacing)
		count := int(expected)
		if noise.Hash3(look.Seed, int64(index), -1, 0) < expected-float64(count) {
			count++
		}
		for strand := range count {
			rig.grow(&cloth, sealFurRoot{triangle, int64(index), int64(strand)},
				points, normals, blade, look.Seed, discs)
		}
	}
	return mesh.NewWithRestPoints(cloth.corners, cloth.rest, cloth.faces,
		cloth.normals, true)
}

// sealFurRoot names one blade by its triangle, the triangle's number and
// the blade's number in it, which seed all its random choices.
type sealFurRoot struct {
	triangle      [3]int
	index, strand int64
}

// grow adds one blade rooted at a random place in a triangle.
func (rig *SealRig) grow(cloth *sealFurCloth, root sealFurRoot, points, normals []maths.Vec,
	blade sealFurBlade, seed uint64, discs []bareDisc,
) {
	random := func(channel int64) float64 {
		return noise.Hash3(seed, root.index, root.strand, channel)
	}
	across, along := random(1), random(2)
	if across+along > 1 {
		across, along = 1-across, 1-along
	}
	weights := [3]float64{1 - across - along, across, along}
	var base, rest, normal, flow maths.Vec
	for corner, index := range root.triangle {
		base = base.Add(points[index].Scale(weights[corner]))
		rest = rest.Add(rig.rest[index].Scale(weights[corner]))
		normal = normal.Add(normals[index].Scale(weights[corner]))
		flow = flow.Add(rig.flows[index].Scale(weights[corner]))
	}
	if bareAt(discs, rest) {
		return
	}
	normal = normal.Unit()
	direction := sealLay(root.triangle, rig.rest, points, flow, normal)
	direction = maths.Rotation(normal, (random(3)-.5)*sealFurScatter).Out(direction)
	length := blade.length * (1 + sealFurVary*(random(4)-.5)*2)
	cloth.add(base, rest, normal, direction, length, blade)
}

// sealLay carries a rest-space flow direction onto the posed triangle:
// the flow's part in the rest triangle's plane is written in the rest
// edges and read back along the posed edges, then laid flat on the skin
// whose unit normal is given.
func sealLay(triangle [3]int, rest, points []maths.Vec, flow, normal maths.Vec) maths.Vec {
	first, second := rest[triangle[1]].Sub(rest[triangle[0]]), rest[triangle[2]].Sub(rest[triangle[0]])
	g11, g12, g22 := first.Dot(first), first.Dot(second), second.Dot(second)
	r1, r2 := flow.Dot(first), flow.Dot(second)
	determinant := g11*g22 - g12*g12
	direction := maths.V(1, 0, 0)
	if determinant > 0 {
		a := (r1*g22 - r2*g12) / determinant
		b := (r2*g11 - r1*g12) / determinant
		direction = points[triangle[1]].Sub(points[triangle[0]]).Scale(a).
			Add(points[triangle[2]].Sub(points[triangle[0]]).Scale(b))
	}
	direction = direction.Sub(normal.Scale(direction.Dot(normal)))
	if direction.Len() < sealFrameLeast {
		return sealFrame(normal, maths.V(1, 0, 0)).X
	}
	return direction.Unit()
}

// add makes one blade: a root pair, a mid pair and a tip, bent from the
// lift at the root toward lying down at the tip.
func (cloth *sealFurCloth) add(base, rest, normal, direction maths.Vec,
	length float64, blade sealFurBlade,
) {
	side := normal.Cross(direction).Unit().Scale(blade.width / 2)
	lean := func(angle float64) maths.Vec {
		return normal.Scale(math.Sin(angle)).Add(direction.Scale(math.Cos(angle)))
	}
	middle := base.Add(lean(blade.lift).Scale(length / 2))
	tip := middle.Add(lean(blade.lift * sealFurTip).Scale(length / 2))
	corners := [5]maths.Vec{base.Sub(side), base.Add(side),
		middle.Sub(side.Scale(sealFurMid)), middle.Add(side.Scale(sealFurMid)), tip}
	ribbon := side.Cross(tip.Sub(base)).Unit()
	if ribbon.Dot(normal) < 0 {
		ribbon = ribbon.Neg()
	}
	shade := normal.Scale(sealFurBase).Add(ribbon.Scale(sealFurRibbon)).Unit()
	first := len(cloth.corners)
	for _, corner := range corners {
		cloth.corners = append(cloth.corners, corner)
		cloth.rest = append(cloth.rest, rest)
		cloth.normals = append(cloth.normals, shade)
	}
	cloth.faces = append(cloth.faces, [3]int{first, first + 1, first + 3},
		[3]int{first, first + 3, first + 2}, [3]int{first + 2, first + 3, first + 4})
}
