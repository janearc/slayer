package pinnipeds

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// sealMold gathers the small closed pieces of one material, such as
// corneas, lips and claws, into one mesh with smooth shading.
type sealMold struct {
	faces   [][3]maths.Vec
	normals [][3]maths.Vec
}

// mesh is the gathered pieces as one mesh, or nil when nothing was added.
func (mold *sealMold) mesh() (*mesh.Mesh, error) {
	if len(mold.faces) == 0 {
		return nil, nil
	}
	return geometry.FaceMesh(mold.faces, mold.normals)
}

// sealFrame is a right-handed frame whose Z axis is normal, and whose X
// axis is along as near as it can lie across that normal.
func sealFrame(normal, along maths.Vec) maths.Orientation {
	z := normal.Unit()
	x := along.Sub(z.Scale(along.Dot(z)))
	if x.Len() < sealFrameLeast {
		x = z.Cross(maths.Up)
	}
	if x.Len() < sealFrameLeast {
		x = z.Cross(maths.V(1, 0, 0))
	}
	x = x.Unit()
	return maths.Orientation{X: x, Y: z.Cross(x), Z: z}
}

// sealFrameLeast is the shortest an along direction may be, once its part
// along the normal is taken away, before another is chosen.
const sealFrameLeast = 1e-6

// ellipsoid adds a closed ellipsoid with the given centre, turned by
// turn, whose half-lengths along the turned axes are radii. Fine is the
// cut of the ball it is made from.
func (mold *sealMold) ellipsoid(centre maths.Vec, turn maths.Orientation,
	radii maths.Vec, fine int,
) {
	for _, one := range geometry.Ball[[3]maths.Vec](fine) {
		var placed [3]maths.Vec
		var shading [3]maths.Vec
		for corner, point := range one {
			placed[corner] = centre.Add(turn.Out(maths.V(
				point.X*radii.X, point.Y*radii.Y, point.Z*radii.Z)))
			shading[corner] = turn.Out(maths.V(
				point.X/radii.X, point.Y/radii.Y, point.Z/radii.Z)).Unit()
		}
		mold.faces = append(mold.faces, placed)
		mold.normals = append(mold.normals, shading)
	}
}

// Cuts of a lip: how many steps run round its outline and round its tube.
const (
	sealLipAround = 18
	sealLipTube   = 6
)

// lip adds a closed ring, a torus whose centre line is an ellipse with
// half-lengths across and tall in the local X and Y axes, with a tube of
// the given radius. Local Z is the way the ring faces.
func (mold *sealMold) lip(centre maths.Vec, turn maths.Orientation,
	across, tall, tube float64,
) {
	point := func(step, round int) (maths.Vec, maths.Vec) {
		angle := 2 * math.Pi * float64(step) / sealLipAround
		phase := 2 * math.Pi * float64(round) / sealLipTube
		sine, cosine := math.Sincos(angle)
		out := maths.V(cosine, sine, 0)
		normal := out.Scale(math.Cos(phase)).Add(maths.V(0, 0, math.Sin(phase)))
		local := maths.V(across*cosine, tall*sine, 0).Add(normal.Scale(tube))
		return centre.Add(turn.Out(local)), turn.Out(normal)
	}
	for step := range sealLipAround {
		for round := range sealLipTube {
			var corners [4]maths.Vec
			var shading [4]maths.Vec
			for index, offset := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
				corners[index], shading[index] = point(step+offset[0], round+offset[1])
			}
			mold.quad(corners, shading)
		}
	}
}

// quad adds a quadrilateral as two triangles, with its corner normals.
func (mold *sealMold) quad(corners, shading [4]maths.Vec) {
	for _, three := range [2][3]int{{0, 1, 2}, {0, 2, 3}} {
		one := [3]maths.Vec{corners[three[0]], corners[three[1]], corners[three[2]]}
		if !geometry.HasArea(one) {
			continue
		}
		mold.faces = append(mold.faces, one)
		mold.normals = append(mold.normals,
			[3]maths.Vec{shading[three[0]], shading[three[1]], shading[three[2]]})
	}
}

// strand is a tube three-sided, so that a strand costs few faces.
func (mold *sealMold) strand(path []maths.Vec, radius []float64) {
	const sides = 3
	mold.tube(path, radius, sides)
}

// tube adds a closed tube along a polyline, with a radius at each point
// and the given number of sides. The first point may lie inside the skin
// it grows from.
func (mold *sealMold) tube(path []maths.Vec, radius []float64, sides int) {
	rings := make([][]maths.Vec, len(path))
	for index, centre := range path {
		tangent := path[min(index+1, len(path)-1)].Sub(path[max(index-1, 0)]).Unit()
		turn := sealFrame(tangent, maths.Up)
		ring := make([]maths.Vec, sides)
		for side := range ring {
			sine, cosine := math.Sincos(2 * math.Pi * float64(side) / float64(sides))
			ring[side] = centre.Add(turn.Out(maths.V(cosine, sine, 0).Scale(radius[index])))
		}
		rings[index] = ring
	}
	skin := geometry.Loft[[3]maths.Vec](rings)
	mold.faces = append(mold.faces, skin...)
	mold.normals = append(mold.normals, geometry.Creased(skin)...)
}

// SealLook is how a seal is dressed over its skin. Wet runs from dry to
// soaked and Nostrils from open to shut, each in [0, 1]. A phocid's
// nostrils are shut at rest and open only to breathe. Seed varies the
// pelage's spots.
type SealLook struct {
	Wet, Nostrils float64
	Seed          uint64
}

// check refuses a look whose wet or nostril share is not a number from
// nought to one.
func (look SealLook) check() error {
	for _, share := range [...]float64{look.Wet, look.Nostrils} {
		if math.IsNaN(share) || share < 0 || share > 1 {
			return fmt.Errorf("seal look: share outside [0, 1]")
		}
	}
	return nil
}

// sealAnchor is a landmark on the rest skin carried to the pose, with the
// posed skin's outward, forward, upward and leftward directions there.
type sealAnchor struct{ point, normal, forward, up, across maths.Vec }

// sealAnchorStep is how far along a direction a rest point is moved to
// find where that direction goes in the pose, in metres.
const sealAnchorStep = .01

// sealProbe is how far outside the skin, in metres, a landmark's ray
// starts.
const sealProbe = .5

// anchor finds where a ray first meets the rest skin and carries that
// landmark, and the directions at it, to the pose.
func (rig *SealRig) anchor(pose SealRigPose, origin, direction maths.Vec) (sealAnchor, error) {
	rest, normal, found := rig.restHit(origin, direction)
	if !found {
		return sealAnchor{}, fmt.Errorf("harbor seal: no skin at landmark %v", origin)
	}
	at := rig.HeadPoint(pose, rest)
	toward := func(direction maths.Vec) maths.Vec {
		moved := rig.HeadPoint(pose, rest.Add(direction.Scale(sealAnchorStep)))
		return moved.Sub(at).Unit()
	}
	return sealAnchor{at, toward(normal), toward(maths.V(1, 0, 0)),
		toward(maths.Up), toward(maths.V(0, 0, 1))}, nil
}

// sealMolds are the five materials the small anatomy is made in.
type sealMolds struct{ cornea, rim, dark, hair, horn sealMold }

// Parts makes what stands on or in the skin as meshes in the posed
// space: corneas and their lids, ear openings and nostrils with their
// lips, the umbilicus, every whisker as its own strand, and the claws.
// Each part's root lies inside the skin it grows from.
func (rig *SealRig) Parts(pose SealRigPose, look SealLook) ([]thing.Thing, error) {
	if err := pose.check(); err != nil {
		return nil, err
	}
	if err := look.check(); err != nil {
		return nil, err
	}
	var molds sealMolds
	for _, side := range [...]float64{-1, 1} {
		for _, build := range []func(SealRigPose, float64, SealLook, *sealMolds) error{
			rig.eye, rig.ear, rig.nostril, rig.whiskers, rig.brow,
		} {
			if err := build(pose, side, look, &molds); err != nil {
				return nil, err
			}
		}
	}
	if err := rig.umbilicus(pose, &molds); err != nil {
		return nil, err
	}
	rig.claws(rig.posed(pose), &molds)
	return molds.things(look)
}

// things makes each mold a thing in its material.
func (molds *sealMolds) things(look SealLook) ([]thing.Thing, error) {
	skin := thing.Material{Name: "seal lip", Colour: maths.RGB(.075, .062, .052),
		Gloss: .16 + .2*look.Wet, Satin: .3}
	entries := []struct {
		mold     *sealMold
		material thing.Material
	}{
		{&molds.cornea, thing.Material{Name: "harbor seal eye",
			Colour: maths.RGB(.010, .008, .007), Gloss: .55, Satin: .28}},
		{&molds.rim, skin},
		{&molds.dark, thing.Material{Name: "seal recess",
			Colour: maths.RGB(.006, .005, .005), Gloss: .10, Satin: .1}},
		{&molds.hair, thing.Material{Name: "seal whisker",
			Colour: maths.RGB(.44, .37, .28), Gloss: .21, Satin: .4}},
		{&molds.horn, thing.Material{Name: "seal claw",
			Colour: maths.RGB(.085, .072, .058), Gloss: .30, Satin: .25}},
	}
	var parts []thing.Thing
	for _, entry := range entries {
		made, err := entry.mold.mesh()
		if err != nil {
			return nil, fmt.Errorf("harbor seal %s: %w", entry.material.Name, err)
		}
		if made != nil {
			material := entry.material
			parts = append(parts, thing.Thing{Shape: made, Material: &material,
				Motion: thing.Still(maths.Local())})
		}
	}
	return parts, nil
}

// The eye. It sits where a ray from the side meets the head at this
// height, faces forward and outward, and is ringed by a lid.
const (
	sealEyeX, sealEyeY = .73, .110 // chosen: where the head is widest at the brow
	sealEyeForward     = .55       // chosen: share of the forward axis in its gaze
	sealCorneaRound    = .0195     // chosen: the cornea's radius in metres
	sealCorneaDepth    = .0085     // chosen: its depth, from the sunk centre out
	sealCorneaSunk     = .0050     // chosen: how far its centre lies under the skin
	sealLidRound       = .0235     // chosen: the lid ring's radius
	sealLidTube        = .0028     // chosen: the lid ring's thickness
)

// eye sets the eye of one side into the skin at its anchor on the head;
// side is the sign of z.
func (rig *SealRig) eye(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	anchor, err := rig.anchor(pose,
		maths.V(sealEyeX, sealEyeY, side*sealProbe), maths.V(0, 0, -side))
	if err != nil {
		return err
	}
	gaze := anchor.normal.Add(anchor.forward.Scale(sealEyeForward)).Unit()
	turn := sealFrame(gaze, anchor.up)
	molds.cornea.ellipsoid(anchor.point.Sub(gaze.Scale(sealCorneaSunk)), turn,
		maths.V(sealCorneaRound, sealCorneaRound, sealCorneaDepth), 20)
	molds.rim.lip(anchor.point.Sub(gaze.Scale(sealLidTube/2)), turn,
		sealLidRound, sealLidRound, sealLidTube)
	return nil
}

// The ear is an opening with a raised lip and no pinna. All chosen.
const (
	sealEarX, sealEarY = .655, .112
	sealEarLong        = .0105
	sealEarTall        = .0075
	sealEarTube        = .0020
)

// ear sets the ear of one side at its anchor on the head; side is the
// sign of z.
func (rig *SealRig) ear(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	anchor, err := rig.anchor(pose,
		maths.V(sealEarX, sealEarY, side*sealProbe), maths.V(0, 0, -side))
	if err != nil {
		return err
	}
	turn := sealFrame(anchor.normal, anchor.forward)
	molds.rim.lip(anchor.point.Sub(anchor.normal.Scale(sealEarTube/4)), turn,
		sealEarLong, sealEarTall, sealEarTube)
	molds.dark.ellipsoid(anchor.point.Sub(anchor.normal.Scale(sealEarTube)), turn,
		maths.V(sealEarLong-sealEarTube/2, sealEarTall-sealEarTube/2, sealEarTube), 10)
	return nil
}

// The nostrils are two slits that meet below, a V seen from the front.
// A slit's lips close it; its length is fixed. All chosen.
const (
	sealNoseProbe      = 1.1   // x, in metres, from which a ray finds the nose
	sealNoseY          = .067  // height of the nostrils
	sealNoseApart      = .022  // each nostril's distance from the midline
	sealNostrilSlant   = .70   // radians above level the outer end sits
	sealNostrilLong    = .0085 // half the slit's length
	sealNostrilTube    = .0008 // the lip's thickness
	sealNostrilOpening = .0020 // the lip's added half-width when fully open
	// sealNostrilShutGap is the least the lip's half-width stands above
	// its thickness, a tenth of a millimetre: at the thickness the ring's
	// inner side is a line and its faces lie on one another.
	sealNostrilShutGap = .0001
)

// nostril sets the nostril of one side at the nose, as the look's share of
// nostrils says; side is the sign of z.
func (rig *SealRig) nostril(pose SealRigPose, side float64, look SealLook, molds *sealMolds) error {
	anchor, err := rig.anchor(pose,
		maths.V(sealNoseProbe, sealNoseY, side*sealNoseApart), maths.V(-1, 0, 0))
	if err != nil {
		return err
	}
	slit := anchor.up.Scale(math.Sin(sealNostrilSlant)).
		Add(anchor.across.Scale(side * math.Cos(sealNostrilSlant)))
	turn := sealFrame(anchor.normal, slit)
	half := math.Max(sealNostrilTube+sealNostrilOpening*(1-look.Nostrils),
		sealNostrilTube+sealNostrilShutGap)
	molds.rim.lip(anchor.point.Sub(anchor.normal.Scale(sealNostrilTube/2)), turn,
		sealNostrilLong, half, sealNostrilTube)
	molds.dark.ellipsoid(anchor.point.Sub(anchor.normal.Scale(sealNostrilTube*1.6)), turn,
		maths.V(sealNostrilLong-sealNostrilTube/2, half-sealNostrilTube/2+sealNostrilTube/4,
			sealNostrilTube*1.5), 10)
	return nil
}

// The umbilicus is a small scar on the belly, a raised ring round a pit.
// All chosen.
const (
	sealNavelX    = -.09
	sealNavelRing = .0085
	sealNavelTube = .0022
	sealNavelPit  = .0050
)

// umbilicus sets the navel on the belly at its anchor, facing down.
func (rig *SealRig) umbilicus(pose SealRigPose, molds *sealMolds) error {
	anchor, err := rig.anchor(pose, maths.V(sealNavelX, -sealProbe, 0), maths.Up)
	if err != nil {
		return err
	}
	turn := sealFrame(anchor.normal, anchor.forward)
	molds.rim.lip(anchor.point.Sub(anchor.normal.Scale(sealNavelTube/2)), turn,
		sealNavelRing, sealNavelRing, sealNavelTube)
	molds.dark.ellipsoid(anchor.point.Sub(anchor.normal.Scale(sealNavelTube)), turn,
		maths.V(sealNavelPit, sealNavelPit, sealNavelTube), 10)
	return nil
}

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
	for index, normal := range geometry.SmoothNormals(points, rig.faces) {
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

// The mystacial vibrissae grow from a pad in rows and columns, one strand
// to a follicle. A harbor seal's array has "~44 whiskers in seven rows",
// "a grid-like pattern that includes 7 rows and 9 columns", arclength
// falling from caudal to rostral (Graff and others 2024, Journal of
// Experimental Biology 227(22), Europe PMC PMC11634035), read here as
// one side's array.
//
// The longest "tended to be found in the lower rows
// and posterior columns", and phocid whiskers run "from about 1.5 cm to
// 10 cm" (Ginter Summarell and others 2015, PLoS ONE 10(7) e0127941).
// Each row's count, the places, lengths and angles are chosen.
const (
	sealWhiskerRows    = 7
	sealWhiskerColumns = 9
	// sealMystacials is the strands on one side: the row counts' sum.
	sealMystacials = 44
	// sealPadX is the X of the rearmost column, sealPadStep the gap between
	// columns, sealPadY the height of the lowest row and sealPadRise the
	// gap between rows, all in metres.
	sealPadX, sealPadStep = .745, .0065
	sealPadY, sealPadRise = .030, .008
	// sealWhiskerLong is the rear lower strand's length, shortened by
	// sealWhiskerLess per column and sealWhiskerRowLess per row.
	sealWhiskerLong    = .082
	sealWhiskerLess    = .0050
	sealWhiskerRowLess = .0040
	// sealWhiskerFan tilts each row's strands up or down.
	sealWhiskerFan = .09
	// sealWhiskerOut and sealWhiskerAhead weigh the skin's normal and the
	// forward axis in a strand's first direction.
	sealWhiskerOut, sealWhiskerAhead = .50, .80
	// sealWhiskerSag is how far a strand's tip hangs below its straight
	// line, as a share of its length, and sealWhiskerSweep how far it
	// sweeps forward.
	sealWhiskerSag, sealWhiskerSweep = .30, .18
	// sealWhiskerBead is the amplitude, in metres, of the wave a strand
	// makes along its length, and sealWhiskerBeads how many waves it makes.
	sealWhiskerBead  = .0012
	sealWhiskerBeads = 5
	// sealWhiskerRoot and sealWhiskerTip are a strand's radius at the skin
	// and at its end, in metres. sealWhiskerSteps is the segments in one.
	sealWhiskerRoot, sealWhiskerTip = .00042, .00008
	sealWhiskerSteps                = 7
	// sealWhiskerBury is how deep a strand's root lies in the skin.
	sealWhiskerBury = .002
)

// sealStrand is one vibrissa: its root on the skin, its first direction,
// its length, and the phase of its wave.
type sealStrand struct {
	anchor    sealAnchor
	direction maths.Vec
	length    float64
	phase     float64
}

// vibrissa adds one strand to the mold as its own thin tube. It leaves
// the skin along its direction, sags and sweeps forward and beads.
func (mold *sealMold) vibrissa(strand sealStrand) {
	root := strand.anchor.point.Sub(strand.anchor.normal.Scale(sealWhiskerBury))
	perpendicular := strand.direction.Cross(strand.anchor.up).Unit()
	droop := strand.anchor.up.Scale(-sealWhiskerSag).
		Add(strand.anchor.forward.Scale(sealWhiskerSweep)).Scale(strand.length)
	path := make([]maths.Vec, sealWhiskerSteps+1)
	radius := make([]float64, len(path))
	for step := range path {
		share := float64(step) / sealWhiskerSteps
		wave := math.Sin(2*math.Pi*sealWhiskerBeads*share+strand.phase) * sealWhiskerBead * share
		path[step] = root.Add(strand.direction.Scale(strand.length * share)).
			Add(droop.Scale(share * share)).Add(perpendicular.Scale(wave))
		radius[step] = sealWhiskerRoot + (sealWhiskerTip-sealWhiskerRoot)*share
	}
	mold.strand(path, radius)
}

// sealWhiskersInRow is how many strands each row holds, from the lowest,
// filling its columns from the rearmost forward. They sum to
// sealMystacials; chosen.
var sealWhiskersInRow = [sealWhiskerRows]int{7, 8, 7, 7, 6, 5, 4}

// whiskers sets the array of whiskers of one side on the pad, row by row,
// each strand at its anchor; side is the sign of z.
func (rig *SealRig) whiskers(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	for row, count := range sealWhiskersInRow {
		for column := range count {
			anchor, err := rig.anchor(pose, maths.V(sealPadX+sealPadStep*float64(column),
				sealPadY+sealPadRise*float64(row), side*sealProbe), maths.V(0, 0, -side))
			if err != nil {
				return err
			}
			tilt := (float64(row) - float64(sealWhiskerRows-1)/2) * sealWhiskerFan
			direction := anchor.normal.Scale(sealWhiskerOut).
				Add(anchor.forward.Scale(sealWhiskerAhead)).
				Add(anchor.up.Scale(tilt)).Unit()
			molds.hair.vibrissa(sealStrand{anchor, direction,
				sealWhiskerLong - sealWhiskerLess*float64(column) -
					sealWhiskerRowLess*float64(row),
				float64(row*sealWhiskerColumns+column) * sealWhiskerPhase})
		}
	}
	return nil
}

// sealWhiskerPhase offsets the wave of neighbouring strands so they do
// not bead in step, in radians.
const sealWhiskerPhase = 1.3

// The supraorbital vibrissae grow above each eye, a short row of stiff
// strands. Counts and numbers are chosen.
const (
	sealBrowCount = 4
	sealBrowX     = .700
	sealBrowStep  = .012
	sealBrowZ     = .05
	sealBrowLong  = .052
	sealBrowLess  = .004
	// sealBrowOut, sealBrowAhead and sealBrowAside weigh the skin's
	// normal, the forward axis and the outward side in a strand's first
	// direction.
	sealBrowOut, sealBrowAhead, sealBrowAside = .50, .70, .30
)

// brow sets the row of brow whiskers above the eye of one side; side is the
// sign of z.
func (rig *SealRig) brow(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	for index := range sealBrowCount {
		step := float64(index)
		anchor, err := rig.anchor(pose, maths.V(sealBrowX+sealBrowStep*step,
			sealProbe, side*sealBrowZ), maths.V(0, -1, 0))
		if err != nil {
			return err
		}
		direction := anchor.normal.Scale(sealBrowOut).
			Add(anchor.forward.Scale(sealBrowAhead)).
			Add(anchor.across.Scale(side * sealBrowAside)).Unit()
		molds.hair.vibrissa(sealStrand{anchor, direction,
			sealBrowLong - sealBrowLess*step, step * sealWhiskerPhase})
	}
	return nil
}

// The claws. Each digit tip carries one, a curved horn cone whose root
// lies inside the digit's end. Fore claws are larger. All chosen.
const (
	sealForeClawLong, sealForeClawRound = .017, .0042
	sealHindClawLong, sealHindClawRound = .009, .0028
	// sealClawBury is how far back along the claw, as a share of its
	// length, the root lies inside the digit.
	sealClawBury = .25
	// sealClawCurl is how far the tip curls down, as a share of length.
	sealClawCurl = .28
	// sealClawSteps and sealClawSides cut a claw along and round.
	sealClawSteps, sealClawSides = 6, 6
	// sealClawBlunt is the least radius a claw's tip keeps, in metres.
	sealClawBlunt = .00006
)

// claws adds a claw at each digit's end, found where the outermost ring's
// digit vertices lie in the posed skin and pointing along the digit.
func (rig *SealRig) claws(points []maths.Vec, molds *sealMolds) {
	for _, limb := range rig.limbs {
		long, round := sealForeClawLong, sealForeClawRound
		if limb.hind {
			long, round = sealHindClawLong, sealHindClawRound
		}
		for digit := range sealDigits {
			end := sealDigitMean(points, limb.outer, limb.digits, digit)
			back := sealDigitMean(points, limb.inner, limb.digits, digit)
			molds.horn.claw(end, end.Sub(back).Unit(), long, round)
		}
	}
}

// sealDigitMean is the mean of the posed points of a ring's vertices that
// end one digit.
func sealDigitMean(points []maths.Vec, ring, digits []int, digit int) maths.Vec {
	var sum maths.Vec
	var count float64
	for place, index := range ring {
		if digits[place] == digit {
			sum = sum.Add(points[index])
			count++
		}
	}
	return sum.Scale(1 / count)
}

// claw adds a curved cone from root along axis, curling toward the belly.
func (mold *sealMold) claw(root, axis maths.Vec, long, round float64) {
	path := make([]maths.Vec, sealClawSteps+1)
	radius := make([]float64, len(path))
	for step := range path {
		share := -sealClawBury + (1+sealClawBury)*float64(step)/sealClawSteps
		ahead := math.Max(share, 0)
		path[step] = root.Add(axis.Scale(long * share)).
			Add(maths.V(0, -sealClawCurl*long*ahead*ahead, 0))
		radius[step] = math.Max(round*math.Pow(1-ahead, .7), sealClawBlunt)
	}
	mold.tube(path, radius, sealClawSides)
}
