package pinnipeds

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

const (
	// walrusGridX, walrusGridY and walrusGridZ are the cells the field is
	// triangulated on, about thirteen millimetres to a cell.
	walrusGridX, walrusGridY, walrusGridZ = 300, 120, 160
	// walrusEyeRadius and walrusNostrilRadius are the radii of the dark
	// spheres set in the eye hollows and the nostril pits, in metres.
	walrusEyeRadius     = .018
	walrusNostrilRadius = .011
	// walrusNostrilSink is how far a nostril's sphere sits below the hide.
	walrusNostrilSink = .004
)

// WalrusStudy builds an adult bull walrus lying on land, as geometry: one
// smooth hide for head, neck and trunk, cut from an outline of stations
// from nose to tail and laid over a skeleton of the species' bones.
//
// The chest is propped on the fore flippers and the belly lies flat where its
// weight rests. It has a small head on a neck as wide as the shoulders,
// broad muzzle pads with a bristle in every follicle, tusks from under
// the pads, small eyes set high, fore flippers and turned-forward hind
// flippers of five digits each, and a short tail.
//
// The hide is bound to the skeleton: yaw bends the loins, phase is the
// stroke of the flippers, and at zero of each the walrus lies at rest. A
// study, not a specimen.
func WalrusStudy(yaw, phase float64) ([]thing.Thing, error) {
	return defaultWalrusRig().walrusAtPose(walrusLying, yaw, phase)
}

// walrusAtPose builds the walrus in a pose, bent and stroked. Only a
// walrus lying or hauled out has the underside its weight flattens.
func (rig *walrusRig) walrusAtPose(pose walrusPose, yaw, phase float64) ([]thing.Thing, error) {
	skin, head, err := rig.skin(pose, yaw, phase)
	if err != nil {
		return nil, err
	}
	outline, err := rig.outline()
	if err != nil {
		return nil, err
	}
	masses, err := rig.walrusMasses(outline)
	if err != nil {
		return nil, err
	}
	field, err := rig.walrusField(outline, masses, pose != walrusSwimming)
	if err != nil {
		return nil, err
	}
	hide, err := rig.walrusRestMesh(field, skin, pose != walrusSwimming)
	if err != nil {
		return nil, err
	}
	parts := []thing.Thing{{Shape: hide, Material: rig.hideMaterial(),
		Motion: walrusStill(maths.Vec{})}}
	for _, side := range []float64{-1, 1} {
		attached, err := rig.walrusHeadParts(field, head, side)
		if err != nil {
			return nil, err
		}
		parts = append(parts, attached...)
	}
	return parts, nil
}

// walrusRestSurface is a private, lazily meshed rest shape for one field.
// It lives with the rig, with separate ground and swimming entries. At the
// default grid a populated entry retains about 17 MB until the rig is freed.
type walrusRestSurface struct {
	once   sync.Once
	points []maths.Vec
	faces  [][3]int
	err    error
}

// restSurface caches the field in its rest space for this rig and ground condition.
// Each entry is used with one fixed field and grid for the lifetime of the rig.
func (rig *walrusRig) restSurface(field func(maths.Vec) float64, ground bool, grid [3]int) (*walrusRestSurface, error) {
	slot := &rig.restSurfaces[0]
	if ground {
		slot = &rig.restSurfaces[1]
	}
	slot.once.Do(func() {
		rest := func(p maths.Vec) maths.Vec { return p }
		still, err := geometry.FieldMeshGrid(field, rest, grid[0], grid[1], grid[2])
		if err != nil {
			slot.err = err
			return
		}
		slot.points = slices.Clone(still.Vertices)
		slot.faces = slices.Clone(still.Faces)
	})
	return slot, slot.err
}

// walrusRestMesh is the hide: the field meshed at rest, then every vertex
// moved by skin, with each one's rest place kept beside it so the hide's
// colour, read at the rest place, stays on the body as it bends. The
// processor tracer reads the rest place; the GPU's paint does not yet.
func (rig *walrusRig) walrusRestMesh(field func(maths.Vec) float64,
	skin func(maths.Vec) maths.Vec, ground bool) (*mesh.Mesh, error) {
	slot, err := rig.restSurface(field, ground, [3]int{walrusGridX, walrusGridY, walrusGridZ})
	if err != nil {
		return nil, err
	}
	posed := make([]maths.Vec, len(slot.points))
	for i, p := range slot.points {
		posed[i] = skin(p)
	}
	return mesh.NewWithRestPoints(posed, slot.points, slot.faces, geometry.AreaNormals(posed, slot.faces), true)
}

// volumes is the walrus's volumes alone, at rest: the hide the
// outline and the flippers' flesh make before any face cut, in a plain
// grey, for seeing the form.
func (rig *walrusRig) volumes() ([]thing.Thing, error) {
	outline, err := rig.outline()
	if err != nil {
		return nil, err
	}
	masses, err := rig.walrusMasses(outline)
	if err != nil {
		return nil, err
	}
	rest := func(p maths.Vec) maths.Vec { return p }
	hide, err := geometry.FieldMeshGrid(walrusVolumeField(outline, masses),
		rest, walrusGridX, walrusGridY, walrusGridZ)
	if err != nil {
		return nil, err
	}
	plain := &thing.Material{Name: "walrus volumes", Colour: maths.Grey(.45),
		Gloss: .1, Satin: .5}
	return []thing.Thing{{Shape: hide, Material: plain,
		Motion: walrusStill(maths.Vec{})}}, nil
}

// The order of the walrus's parts: the hide first, then each side's head
// parts, the left side's before the right's, in the order below.
// walrusHidePart is the hide's place in the parts.
const walrusHidePart = 0

const (
	// walrusEyePart, walrusNostrilPart, walrusTuskPart and
	// walrusWhiskerPart are each head part's place among one side's.
	walrusEyePart = iota
	walrusNostrilPart
	walrusTuskPart
	walrusWhiskerPart
	// walrusHeadPartCount is the head parts of one side.
	walrusHeadPartCount
)

// walrusHeadParts is the eye, nostril, tusk and bristles of one side, in
// the order walrusEyePart to walrusWhiskerPart, each carried rigidly with
// the head by head.
func (rig *walrusRig) walrusHeadParts(field func(maths.Vec) float64, head func(maths.Vec) maths.Vec,
	side float64) ([]thing.Thing, error) {
	ivory := &thing.Material{Name: "walrus ivory", Colour: maths.RGB(.66, .55, .39),
		Texture: walrusIvoryTexture{}, Gloss: .15, Satin: .42}
	dark := &thing.Material{Name: "nostril", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	bristle := &thing.Material{Name: "vibrissae", Colour: maths.RGB(.68, .58, .43), Gloss: .08}
	eye, err := rig.walrusEyeAt(field, side)
	if err != nil {
		return nil, err
	}
	nostril, err := rig.walrusNostrilAt(field, side)
	if err != nil {
		return nil, err
	}
	tusk, err := rig.walrusTusk(side, head)
	if err != nil {
		return nil, err
	}
	whiskers, err := rig.walrusVibrissae(side, field, head)
	if err != nil {
		return nil, err
	}
	parts := make([]thing.Thing, walrusHeadPartCount)
	parts[walrusEyePart] = thing.Thing{Shape: shape.Sphere{Radius: walrusEyeRadius},
		Material: walrusCorneaMaterial(), Motion: walrusStill(head(eye))}
	parts[walrusNostrilPart] = thing.Thing{Shape: shape.Sphere{Radius: walrusNostrilRadius},
		Material: dark, Motion: walrusStill(head(nostril))}
	parts[walrusTuskPart] = thing.Thing{Shape: tusk, Material: ivory,
		Motion: walrusStill(maths.Vec{})}
	parts[walrusWhiskerPart] = thing.Thing{Shape: whiskers, Material: bristle,
		Motion: walrusStill(maths.Vec{})}
	return parts, nil
}

// hideMaterial is the hide's material: its colour by region, and a
// satin sheen.
func (rig *walrusRig) hideMaterial() *thing.Material {
	// Paint reads only the immutable frame and ray lines. Do not retain the
	// recipe's cached rest surfaces merely because its output is still alive.
	paint := &walrusRig{frame: rig.frame, rays: rig.rays}
	return &thing.Material{Name: "walrus hide", Colour: walrusCinnamon,
		Texture: walrusHideTexture{rig: paint}, Gloss: .13, Satin: .59}
}

// walrusStill is a part standing still at a point.
func walrusStill(at maths.Vec) thing.Motion {
	return thing.Still(maths.NewSubspace(at, maths.Identity, 1))
}

// errWalrusLandmark is the refusal of a head part whose place could not be
// found on the hide near where the skull puts it.
var errWalrusLandmark = errors.New("walrus: no hide near a head landmark")

// walrusEyeAt is the centre of the eye on one side: on the hide itself,
// at the bottom of the hollow cut for it, looked for within
// walrusLandmarkReach of the socket's own depth.
func (rig *walrusRig) walrusEyeAt(field func(maths.Vec) float64, side float64) (maths.Vec, error) {
	socket := rig.walrusEyeSocket(side)
	depth := math.Abs(socket.Z)
	z, ok := walrusSurfaceZ(field, socket.X, socket.Y, side,
		math.Max(0, depth-walrusLandmarkReach), depth+walrusLandmarkReach)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the eye on side %g", errWalrusLandmark, side)
	}
	return maths.V(socket.X, socket.Y, z), nil
}

// walrusNostrilAt is the centre of the dark sphere in one nostril, just
// under the hide at the pit cut for it, looked for within
// walrusLandmarkReach of the pit's own height.
func (rig *walrusRig) walrusNostrilAt(field func(maths.Vec) float64, side float64) (maths.Vec, error) {
	pit := rig.walrusNostril(side)
	y, ok := walrusSurfaceY(field, pit.X, pit.Z,
		pit.Y-walrusLandmarkReach, pit.Y+walrusLandmarkReach)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the nostril on side %g",
			errWalrusLandmark, side)
	}
	return maths.V(pit.X, y-walrusNostrilSink, pit.Z), nil
}

// The hide is bound to the skeleton: every point of it at rest is held by
// the bones near it and goes where they go in the pose. On top of a pose,
// yaw bends the loins side to side, shared over the lumbar and first
// sacral joints, and phase is the stroke of the flippers.
const (
	// walrusFlipperSwing is the angle a flipper swings through, in radians
	// at the top of the stroke. Chosen.
	walrusFlipperSwing = .30
)

// walrusBendJoints are the joints a yaw is shared over.
var walrusBendJoints = walrusBendingJoints(defaultPinniped("walrus"))

func walrusBendingJoints(profile Profile) []string {
	names := make([]string, 0, profile.Lumbar+1)
	for i := 1; i <= profile.Lumbar; i++ {
		names = append(names, fmt.Sprintf("lumbar %d", i))
	}
	return append(names, "sacral 1")
}

// stroke is pose with the loins bent by yaw and the flippers at
// phase of their stroke: the hind flippers sweep side to side at the
// ankle, the fore flippers at the shoulder.
func (rig *walrusRig) stroke(pose skeleton.Pose, yaw, phase float64) skeleton.Pose {
	pose, _ = SpinePose(pose, walrusBendingJoints(rig.body.profile), 0, yaw,
		func(string) (float64, float64) { return 0, 1 },
		func(p skeleton.Pose, joint string, _, bend float64) (skeleton.Pose, error) {
			return p.Compose(joint, skeleton.NewTurn(walrusUpAxis, bend)), nil
		})
	swing := walrusFlipperSwing * math.Sin(phase)
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		pose = pose.Compose("astragalus "+name, skeleton.NewTurn(walrusUpAxis, swing)).
			Compose("humerus "+name, skeleton.NewTurn(walrusUpAxis, side*swing))
	}
	return pose
}

// skin binds the hide to the skeleton in the pose, bent by yaw and
// stroked by phase. It gives back the function that carries a point of the
// rest hide to its place, and the motion of the skull, which carries every
// tusk, bristle, eye and nostril rigidly with the head.
func (rig *walrusRig) skin(pose walrusPose, yaw, phase float64) (func(maths.Vec) maths.Vec,
	func(maths.Vec) maths.Vec, error) {
	body := rig.body
	frames, err := walrusPosedFrames(body,
		rig.stroke(walrusPoseOf(body, pose), yaw, phase))
	if err != nil {
		return nil, nil, err
	}
	holders := walrusHolders(body)
	_, skull := body.bone("skull")
	hide := func(p maths.Vec) maths.Vec {
		return skeleton.Carry(p, frames, skeleton.Nearness(p, holders))
	}
	return hide, frames[skull].Apply, nil
}

// checkFieldGrid refuses a rig whose field would be cut at a mesher face.
// The mesher samples this exact lattice before skinning the result, so a
// negative boundary sample would produce an open, silently clipped hide.
// Checking mass centres also catches a flipper wholly outside the grid.
func (rig *walrusRig) checkFieldGrid(outline *walrusOutline, pose walrusPose) error {
	masses, err := rig.walrusMasses(outline)
	if err != nil {
		return err
	}
	field, err := rig.walrusField(outline, masses, pose != walrusSwimming)
	if err != nil {
		return err
	}
	return checkWalrusGrid(masses, field)
}

func (rig *walrusRig) checkVolumeGrid(outline *walrusOutline) error {
	masses, err := rig.walrusMasses(outline)
	if err != nil {
		return err
	}
	return checkWalrusGrid(masses, walrusVolumeField(outline, masses))
}

func checkWalrusGrid(masses []walrusMass, field func(maths.Vec) float64) error {
	point := func(i, j, k int) maths.Vec {
		return maths.V(-1.9+float64(i)*(3.8/walrusGridX)+.000137,
			-.75+float64(j)*(1.5/walrusGridY)+.000193,
			-1+float64(k)*(2.0/walrusGridZ)+.000233)
	}
	low, high := point(0, 0, 0), point(walrusGridX, walrusGridY, walrusGridZ)
	for _, mass := range masses {
		p := mass.centre
		if p.X < low.X || p.X > high.X || p.Z < low.Z || p.Z > high.Z || p.Y < low.Y || p.Y > high.Y {
			return fmt.Errorf("manufacture: walrus field mass lies outside its grid")
		}
	}
	check := func(i, j, k int) error {
		value := field(point(i, j, k))
		if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
			return fmt.Errorf("manufacture: walrus field reaches its grid boundary")
		}
		return nil
	}
	for j := 0; j <= walrusGridY; j++ {
		for k := 0; k <= walrusGridZ; k++ {
			if err := check(0, j, k); err != nil {
				return err
			}
			if err := check(walrusGridX, j, k); err != nil {
				return err
			}
		}
	}
	for i := 0; i <= walrusGridX; i++ {
		for k := 0; k <= walrusGridZ; k++ {
			if err := check(i, 0, k); err != nil {
				return err
			}
			if err := check(i, walrusGridY, k); err != nil {
				return err
			}
		}
		for j := 0; j <= walrusGridY; j++ {
			if err := check(i, j, 0); err != nil {
				return err
			}
			if err := check(i, j, walrusGridZ); err != nil {
				return err
			}
		}
	}
	return nil
}
