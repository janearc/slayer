package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// SealRigPose describes a continuous harbor seal's spine and flipper pose.
// Angles are signed radians. The skin is evaluated from rest at every pose.
type SealRigPose struct {
	Yaw, Pitch, HeadPitch float64
	Fore, Hind            [2]float64
}

// SealRig retains the rest vertices, triangle connectivity and flipper
// influences of one continuous skin, in metres with +X toward the muzzle.
type SealRig struct {
	vertices  []sealVertex
	rest      []maths.Vec
	normals   []maths.Vec
	flows     []maths.Vec
	bare      []bool
	faces     [][3]int
	limbs     [4]sealLimb
	flexStart float64
	// bound ties the skin to the seal's skeleton; a rig without it is
	// posed by its own spine frames.
	bound *sealBound
	// plain is the skin with flippers of plain rings, a capped tail root
	// and no folds: what NewSealRig and NewSealRigWithFlex make.
	plain bool
}

type sealVertex struct {
	point   maths.Vec
	station float64
	limb    int
	weight  float64
}

// sealLimb is one flipper's root and tip at rest. The last two rings
// and the pole are kept so claws and bare skin can find the digit ends:
// digits holds each outer ring vertex's digit, or minus one for web.
type sealLimb struct {
	root, tip    maths.Vec
	side         float64
	hind         bool
	inner, outer []int
	digits       []int
	pole         int
}

const (
	sealShoulder = .30
	sealHip      = -.84
	sealSections = 24
)

// Skin deforms each rest vertex once, then gives Slayer the deformed
// triangles and area-weighted normals. The rest skin can be reused.
func (rig *SealRig) Skin(pose SealRigPose) (*mesh.Mesh, error) {
	if err := pose.check(); err != nil {
		return nil, err
	}
	points := rig.posed(pose)
	return mesh.NewWithRestPoints(points, rig.rest, rig.faces,
		sealSmoothNormals(points, rig.faces), true)
}

// posed is every rest vertex carried to the pose: the spine frame, the
// head nod, the flipper turn and then, but on a plain rig, the folds
// the bend bunches up.
func (rig *SealRig) posed(pose SealRigPose) []maths.Vec {
	if rig.bound != nil {
		return rig.posedOnBones(pose)
	}
	points := make([]maths.Vec, len(rig.vertices))
	frames := make(map[float64]maths.Subspace)
	for index, vertex := range rig.vertices {
		frame, found := frames[vertex.station]
		if !found {
			frame = rig.frame(pose, vertex.station)
			frames[vertex.station] = frame
		}
		local := vertex.point.Sub(maths.V(vertex.station, 0, 0))
		local = local.Add(pose.nod(vertex.point).Sub(vertex.point))
		if vertex.weight > 0 {
			limb := rig.limbs[vertex.limb]
			shifted := limb.posed(vertex.point, pose.angle(vertex.limb))
			local = local.Add(shifted.Sub(vertex.point).Scale(vertex.weight))
		}
		if !rig.plain {
			local = local.Add(rig.fold(index, pose))
		}
		points[index] = frame.Out(local)
	}
	return points
}

// check refuses a pose with an angle that is not a number or lies past
// its limit.
func (pose SealRigPose) check() error {
	limits := [7]float64{math.Pi / 2, .65, 1.15, 1.15, .8, .8, .92}
	angles := [7]float64{pose.Yaw, pose.Pitch, pose.Fore[0], pose.Fore[1],
		pose.Hind[0], pose.Hind[1], pose.HeadPitch}
	for index, angle := range angles {
		if math.IsNaN(angle) || math.Abs(angle) > limits[index] {
			return fmt.Errorf("seal rig: angle %d outside its %.3g radian bound",
				index, limits[index])
		}
	}
	return nil
}

// nod turns a point of the head by the pose's HeadPitch about a joint at
// the neck; a point behind the neck's base is not moved.
func (pose SealRigPose) nod(point maths.Vec) maths.Vec {
	const base, whole = .39, .62
	if point.X <= base || pose.HeadPitch == 0 {
		return point
	}
	joint := maths.V(.44, .060, 0)
	share := sealEase((point.X - base) / (whole - base))
	turn := maths.Rotation(maths.V(0, 0, 1), pose.HeadPitch*share)
	return joint.Add(turn.Out(point.Sub(joint)))
}

// HeadPoint poses a head landmark in the same space as the skin. Eyes
// and whisker roots share the nod instead of staying in the rest frame.
func (rig *SealRig) HeadPoint(pose SealRigPose, rest maths.Vec) maths.Vec {
	if rig.bound != nil {
		return rig.headPointOnBones(pose, rest)
	}
	return rig.frame(pose, rest.X).Out(pose.nod(rest).Sub(maths.V(rest.X, 0, 0)))
}

// SealCornea builds a shallow eye surface, with its viewing axis along -Z.
// The globe behind that surface is enclosed by the posed head.
func SealCornea() (*mesh.Mesh, error) {
	const wide, tall, depth = .022, .017, .010
	skin := ball(cut{around: 20})
	normals := make([][3]maths.Vec, len(skin))
	for i, face := range skin {
		for j, point := range face {
			skin[i][j] = maths.V(point.X*wide, point.Y*tall, point.Z*depth)
			normals[i][j] = maths.V(point.X/wide, point.Y/tall, point.Z/depth).Unit()
		}
	}
	return skin.shaded(normals)
}

// frame follows a length-preserving spine. Integrating a unit tangent
// places its centre, and each ring stays square to that tangent. The
// head and shoulders are held still in the animal's local space.
func (rig *SealRig) frame(pose SealRigPose, station float64) maths.Subspace {
	if station >= rig.flexStart {
		return maths.NewSubspace(maths.V(station, 0, 0), maths.Identity, 1)
	}
	const substeps = 24
	stop := math.Max(station, sealHip)
	step := (rig.flexStart - stop) / substeps
	centre := maths.V(rig.flexStart, 0, 0)
	for slice := range substeps {
		middle := rig.flexStart - (float64(slice)+.5)*step
		centre = centre.Add(rig.orientation(pose, middle).Out(maths.V(-step, 0, 0)))
	}
	turn := rig.orientation(pose, stop)
	return maths.NewSubspace(
		centre.Add(turn.Out(maths.V(station-stop, 0, 0))), turn, 1)
}

// orientation is the turn of the body at a station along its length: the
// pose's pitch and yaw, eased in from the flex start toward the hip.
func (rig *SealRig) orientation(pose SealRigPose, station float64) maths.Orientation {
	share := (rig.flexStart - station) / (rig.flexStart - sealHip)
	weight := sealEase(share)
	return maths.Rotation(maths.V(0, 0, 1), pose.Pitch*weight).
		Then(maths.Rotation(maths.Up, pose.Yaw*weight))
}

// angle is the pose's angle for a limb by index: the fore flippers come
// first, then the hind.
func (pose SealRigPose) angle(limb int) float64 {
	if limb < len(pose.Fore) {
		return pose.Fore[limb]
	}
	return pose.Hind[limb-len(pose.Fore)]
}

// posed is a point of the limb turned by an angle about the limb's root:
// about the up axis for a fore flipper, the x axis for a hind one.
func (limb sealLimb) posed(point maths.Vec, angle float64) maths.Vec {
	axis := maths.Up
	if limb.hind {
		axis = maths.V(1, 0, 0)
	}
	offset := point.Sub(limb.root)
	turned := maths.Rotation(axis, limb.side*angle).Out(offset)
	if !limb.hind && angle < 0 {
		// Keep an aft-sweeping flipper out of the flank. The offset grows
		// from the joint towards the distal skin and vanishes at rest.
		const tuckClearance = 1.72
		turned.Z += tuckClearance * math.Sin(-angle) * offset.Z
	}
	return limb.root.Add(turned)
}

// Skeleton returns the spine and four flipper chains in the same posed
// coordinates used by Skin. The chains are visualization guides.
func (rig *SealRig) Skeleton(pose SealRigPose) ([][]maths.Vec, error) {
	if err := pose.check(); err != nil {
		return nil, err
	}
	spine := make([]maths.Vec, 13)
	for index := range spine {
		station := .73 + (sealHip-.73)*float64(index)/float64(len(spine)-1)
		spine[index] = rig.frame(pose, station).Origin
	}
	chains := [][]maths.Vec{spine}
	for index, limb := range rig.limbs {
		frame := rig.frame(pose, limb.root.X)
		base := maths.V(limb.root.X, 0, 0)
		chains = append(chains, []maths.Vec{frame.Origin,
			frame.Out(limb.root.Sub(base)),
			frame.Out(limb.posed(limb.tip, pose.angle(index)).Sub(base))})
	}
	return chains, nil
}

// sealSmoothNormals is a normal for every point: the sum of the normals of
// the faces that meet there, each weighted by its area.
func sealSmoothNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := make([]maths.Vec, len(points))
	for _, face := range faces {
		origin := points[face[0]]
		area := points[face[1]].Sub(origin).Cross(points[face[2]].Sub(origin))
		for _, index := range face {
			normals[index] = normals[index].Add(area)
		}
	}
	for index, normal := range normals {
		// Opening interiors leave a few unreferenced grid vertices.
		if normal == (maths.Vec{}) {
			normals[index] = maths.Up
		}
	}
	return normals
}
