package manufacture

import (
	"errors"
	"fmt"
	"math"

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

// WalrusFresh5 builds an adult bull walrus lying on land, as geometry: one
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
func WalrusFresh5(yaw, phase float64) ([]thing.Thing, error) {
	return walrusFresh5(walrusLying, yaw, phase)
}

// walrusFresh5 builds the walrus in a pose, bent and stroked. Only a
// walrus lying or hauled out has the underside its weight flattens.
func walrusFresh5(pose walrusPose, yaw, phase float64) ([]thing.Thing, error) {
	skin, head, err := walrusSkin(pose, yaw, phase)
	if err != nil {
		return nil, err
	}
	outline, err := walrusTheOutline()
	if err != nil {
		return nil, err
	}
	masses, err := walrusMasses(outline)
	if err != nil {
		return nil, err
	}
	field, err := walrusField(outline, masses, pose != walrusSwimming)
	if err != nil {
		return nil, err
	}
	hide, err := walrusRestMesh(field, skin)
	if err != nil {
		return nil, err
	}
	parts := []thing.Thing{{Shape: hide, Material: walrusHideMaterial(),
		Motion: walrusStill(maths.Vec{})}}
	for _, side := range []float64{-1, 1} {
		attached, err := walrusHeadParts(field, head, side)
		if err != nil {
			return nil, err
		}
		parts = append(parts, attached...)
	}
	return parts, nil
}

// walrusRestMesh is the hide: the field meshed at rest, then every vertex
// moved by skin, with each one's rest place kept beside it so the hide's
// colour, read at the rest place, stays on the body as it bends. The
// processor tracer reads the rest place; the GPU's paint does not yet.
func walrusRestMesh(field func(maths.Vec) float64,
	skin func(maths.Vec) maths.Vec) (*mesh.Mesh, error) {
	rest := func(p maths.Vec) maths.Vec { return p }
	still, err := bioformFieldMeshGrid(field, rest, walrusGridX, walrusGridY,
		walrusGridZ)
	if err != nil {
		return nil, err
	}
	posed := make([]maths.Vec, len(still.Vertices))
	for i, p := range still.Vertices {
		posed[i] = skin(p)
	}
	return mesh.NewWithRestPoints(posed, still.Vertices, still.Faces,
		walrusVertexNormals(posed, still.Faces), true)
}

// walrusVertexNormals is the normal at each vertex: the sum of its faces'
// normals, each as long as its face is large.
func walrusVertexNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := make([]maths.Vec, len(points))
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		normal := b.Sub(a).Cross(c.Sub(a))
		for _, corner := range face {
			normals[corner] = normals[corner].Add(normal)
		}
	}
	return normals
}

// walrusFresh5Volumes is the walrus's volumes alone, at rest: the hide the
// outline and the flippers' flesh make before any face cut, in a plain
// grey, for seeing the form.
func walrusFresh5Volumes() ([]thing.Thing, error) {
	outline, err := walrusTheOutline()
	if err != nil {
		return nil, err
	}
	masses, err := walrusMasses(outline)
	if err != nil {
		return nil, err
	}
	rest := func(p maths.Vec) maths.Vec { return p }
	hide, err := bioformFieldMeshGrid(walrusVolumeField(outline, masses),
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

// walrusPartIndex is the place in the parts of one head part on one side,
// nought for the left and one for the right.
func walrusPartIndex(sideIndex, part int) int {
	return walrusHidePart + 1 + sideIndex*walrusHeadPartCount + part
}

// walrusHeadParts is the eye, nostril, tusk and bristles of one side, in
// the order walrusEyePart to walrusWhiskerPart, each carried rigidly with
// the head by head.
func walrusHeadParts(field func(maths.Vec) float64, head func(maths.Vec) maths.Vec,
	side float64) ([]thing.Thing, error) {
	ivory := &thing.Material{Name: "walrus ivory", Colour: maths.RGB(.66, .55, .39),
		Texture: walrusIvoryTexture{}, Gloss: .15, Satin: .42}
	dark := &thing.Material{Name: "nostril", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	bristle := &thing.Material{Name: "vibrissae", Colour: maths.RGB(.68, .58, .43), Gloss: .08}
	eye, err := walrusEyeAt(field, side)
	if err != nil {
		return nil, err
	}
	nostril, err := walrusNostrilAt(field, side)
	if err != nil {
		return nil, err
	}
	tusk, err := walrusTusk(side, head)
	if err != nil {
		return nil, err
	}
	whiskers, err := walrusVibrissae(side, field, head)
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

// walrusHideMaterial is the hide's material: its colour by region, and a
// satin sheen.
func walrusHideMaterial() *thing.Material {
	return &thing.Material{Name: "walrus hide", Colour: walrusCinnamon,
		Texture: walrusHideTexture{}, Gloss: .13, Satin: .59}
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
func walrusEyeAt(field func(maths.Vec) float64, side float64) (maths.Vec, error) {
	socket := walrusEyeSocket(side)
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
func walrusNostrilAt(field func(maths.Vec) float64, side float64) (maths.Vec, error) {
	pit := walrusNostril(side)
	y, ok := walrusSurfaceY(field, pit.X, pit.Z,
		pit.Y-walrusLandmarkReach, pit.Y+walrusLandmarkReach)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the nostril on side %g",
			errWalrusLandmark, side)
	}
	return maths.V(pit.X, y-walrusNostrilSink, pit.Z), nil
}
