package pinnipeds

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
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
		geometry.SmoothNormals(points, rig.faces), true)
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
	skin := geometry.Ball[[3]maths.Vec](20)
	normals := make([][3]maths.Vec, len(skin))
	for i, face := range skin {
		for j, point := range face {
			skin[i][j] = maths.V(point.X*wide, point.Y*tall, point.Z*depth)
			normals[i][j] = maths.V(point.X/wide, point.Y/tall, point.Z/depth).Unit()
		}
	}
	return geometry.FaceMesh(skin, normals)
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

// sealBound is a rig's skin tied to the seal's skeleton: the bones with
// their joints, the sets of bones each part of the skin may follow, and
// each rest vertex's influences. A rig with one is posed by its bones.
type sealBound struct {
	made       *sealBones
	holders    [][]skeleton.Holder
	influences [][]skeleton.Influence
}

// sealReach is how far, in metres, a bone holds the skin by the shared
// nearness weighing; chosen, about a vertebra's length times two.
const sealReach = .08

// sealCaudalShare is how much of a turn of the hind body the tail takes,
// against the lumbar column and the sacrum, per unit of its joints'
// range: the tail is short and follows. Chosen.
const sealCaudalShare = .1

// sealShoulderShare is the share of a fore flipper's sweep the shoulder
// takes as a bend of the humerus; chosen.
const sealShoulderShare = .75

// trunkGroup is the binding set of the trunk, neck and head; the limbs'
// sets follow it in the order of the rig's openings.
const trunkGroup = 0

// NewSealRigOnBones builds the rest skin from the seal's skeleton: each
// ring of the trunk, neck and head reaches as far as the masses on the
// bones under it and the blubber over them, and every vertex is bound
// to the bones nearest it, so a pose turns joints and the skin follows.
func NewSealRigOnBones() (*SealRig, error) {
	made, err := harborSealBones()
	if err != nil {
		return nil, err
	}
	body, err := newSealBody(made)
	if err != nil {
		return nil, err
	}
	rig := buildSealRig(sealShoulder,
		body.shape(sealRestRings()), false)
	return rig, rig.bindTo(made)
}

// bindTo ties every rest vertex to its bones: a flipper's vertices to
// that limb and the trunk beside it, the rest to the column and skull.
func (rig *SealRig) bindTo(made *sealBones) error {
	sets := sealBindingSets(made.names())
	holders := make([][]skeleton.Holder, len(sets))
	for group, names := range sets {
		held, err := made.bones.Holders(names, made.forms, sealReach)
		if err != nil {
			return err
		}
		holders[group] = held
	}
	influences := make([][]skeleton.Influence, len(rig.vertices))
	for index, vertex := range rig.vertices {
		group := trunkGroup
		if vertex.weight > 0 {
			group = vertex.limb + 1
		}
		influences[index] = skeleton.Nearness(rig.rest[index],
			holders[group])
	}
	rig.bound = &sealBound{made: made, holders: holders,
		influences: influences}
	return nil
}

// sealBindingSets is the trunk's set, then each opening's limb with the
// column beside it, in the order of sealOpenings.
func sealBindingSets(names []string) [][]string {
	trunk := sealNamesWith(names, "neck", "thoracic", "lumbar", "sacral",
		"caudal", "skull")
	shoulder := sealNamesWith(names, "thoracic1", "thoracic2",
		"thoracic3", "thoracic4")
	hips := sealNamesWith(names, "lumbar4", "lumbar5", "sacral",
		"caudal1", "caudal2")
	sets := [][]string{trunk}
	for _, opening := range sealOpenings {
		side := sealSideName(opening.side)
		near, limb := shoulder, sealNamesWith(names, "fore-"+side+"-")
		if opening.hind {
			near, limb = hips, sealNamesWith(names, "hind-"+side+"-")
		}
		sets = append(sets, append(append([]string{}, near...), limb...))
	}
	return sets
}

// sealNamesWith is the names that begin with any of the beginnings. A
// beginning that is a whole name, as "thoracic1", also takes the names
// it begins ("thoracic10" to "thoracic15"); the sets above allow it.
func sealNamesWith(names []string, beginnings ...string) []string {
	var found []string
	for _, name := range names {
		for _, beginning := range beginnings {
			if strings.HasPrefix(name, beginning) {
				found = append(found, name)
				break
			}
		}
	}
	return found
}

// turns is the rig's pose as turns of the bones: the trunk's yaw and
// pitch shared over the column aft of the flex start, the nod over the
// neck, a fore flipper's sweep at the shoulder, and a hind flipper's
// feather at the ankle. A turn the joints refuse is an error.
func (bound *sealBound) turns(pose SealRigPose,
	start float64,
) (skeleton.Pose, error) {
	turns, err := bound.spine(skeleton.Pose{}, pose, start)
	if err != nil {
		return turns, err
	}
	neck := sealNamesWith(bound.made.names(), "neck")
	for _, name := range neck {
		turns, err = bound.turn(turns, name,
			pose.HeadPitch/float64(len(neck)), 0, 0)
		if err != nil {
			return turns, err
		}
	}
	for limb, opening := range sealOpenings {
		side := sealSideName(opening.side)
		angle := opening.side * pose.angle(limb)
		if opening.hind {
			turns, err = bound.turn(turns, "hind-"+side+"-astragalus", 0, 0,
				angle)
		} else {
			turns, err = bound.turn(turns, "fore-"+side+"-humerus", 0,
				sealShoulderShare*angle, 0)
		}
		if err != nil {
			return turns, err
		}
	}
	return turns, nil
}

// turn is the pose with one more bone turned within its joint.
func (bound *sealBound) turn(pose skeleton.Pose, name string, flex, bend,
	twist float64,
) (skeleton.Pose, error) {
	return bound.made.joints.Turned(bound.made.bones, pose, name, flex,
		bend, twist)
}

// spine shares the trunk's yaw, as a bend, and its pitch, as a flex,
// over the column's bones aft of start, each by its joint's range. The
// rig pitches its tail down where a bone's flex lifts it, so the sign
// of the pitch is turned.
func (bound *sealBound) spine(turns skeleton.Pose, pose SealRigPose,
	start float64,
) (skeleton.Pose, error) {
	names := bound.aft(start)
	return SpinePose(turns, names, -pose.Pitch, pose.Yaw, bound.ranges,
		func(turns skeleton.Pose, name string, flex, bend float64) (skeleton.Pose, error) {
			return bound.turn(turns, name, flex, bend, 0)
		})
}

// aft is the column's bones whose heads lie behind start.
func (bound *sealBound) aft(start float64) []string {
	var names []string
	for _, name := range sealNamesWith(bound.made.names(), "thoracic",
		"lumbar", "sacral", "caudal") {
		head, _, _ := bound.made.ends(name)
		if head.X < start {
			names = append(names, name)
		}
	}
	return names
}

// ranges is how far a column bone flexes and bends, the tail's scaled
// down by the share of a turn it takes.
func (bound *sealBound) ranges(name string) (float64, float64) {
	_, flex, _ := bound.made.joints.Limit(name, skeleton.Flex)
	_, bend, _ := bound.made.joints.Limit(name, skeleton.Bend)
	if strings.HasPrefix(name, "caudal") {
		return flex * sealCaudalShare, bend * sealCaudalShare
	}
	return flex, bend
}

// frames is each bone's motion for the rig's pose.
func (bound *sealBound) frames(pose SealRigPose,
	start float64,
) ([]skeleton.Frame, error) {
	turns, err := bound.turns(pose, start)
	if err != nil {
		return nil, err
	}
	return bound.made.bones.Frames(turns), nil
}

// posedOnBones is every rest vertex carried by its bones in the pose,
// then the folds the bend bunches up. A pose the joints refuse gives
// points of no number, which the mesh then refuses where it is made.
func (rig *SealRig) posedOnBones(pose SealRigPose) []maths.Vec {
	frames, err := rig.bound.frames(pose, rig.flexStart)
	if err != nil {
		return sealUnplaced(len(rig.rest))
	}
	points := make([]maths.Vec, len(rig.rest))
	for index, rest := range rig.rest {
		points[index] = skeleton.Carry(rest, frames,
			rig.bound.influences[index]).Add(rig.fold(index, pose))
	}
	return points
}

// headPointOnBones is a rest landmark carried by the trunk's bones
// nearest it, as the skin there is; of no number if the pose is refused.
func (rig *SealRig) headPointOnBones(pose SealRigPose,
	rest maths.Vec,
) maths.Vec {
	bound := rig.bound
	frames, err := bound.frames(pose, rig.flexStart)
	if err != nil {
		return sealUnplaced(1)[0]
	}
	return skeleton.Carry(rest, frames,
		skeleton.Nearness(rest, bound.holders[trunkGroup]))
}

// sealUnplaced is count points of no number.
func sealUnplaced(count int) []maths.Vec {
	points := make([]maths.Vec, count)
	for index := range points {
		points[index] = maths.V(math.NaN(), math.NaN(), math.NaN())
	}
	return points
}

// The neck folds. When the head nods, the skin on the inside of the bend
// has nowhere to go and stands out in rolls across the neck. All of these
// numbers are chosen.
const (
	// sealNeckStart is the X, in metres, where the neck's rolls begin.
	sealNeckStart = .30
	// sealNeckEnd is the X where they end, at the back of the skull.
	sealNeckEnd = .60
	// sealNeckBlend is how far along X a roll's window fades in and out.
	sealNeckBlend = .05
	// sealNeckPeriod is the distance between two rolls.
	sealNeckPeriod = .075
	// sealNeckDepth is how far a roll stands out at the largest nod.
	sealNeckDepth = .016
	// sealNeckLimit is the largest head pitch, in radians, the rig allows.
	sealNeckLimit = .92
)

// The flipper-root folds. A turned flipper gathers the trunk skin around
// its root into rings.
const (
	// sealRootPeriod is the distance between two rings around a root.
	sealRootPeriod = .05
	// sealRootDepth is how far a ring stands out at a full turn.
	sealRootDepth = .008
	// sealRootTurn is the flipper turn, in radians, that gives full depth.
	sealRootTurn = .6
	// sealRootNear and sealRootFar bound where rings form, as the distance
	// from the root in metres, each with a blend.
	sealRootNear, sealRootFar, sealRootBlend = .03, .13, .03
)

// fold is how far a vertex stands out along its skin's normal where the
// pose bunches the skin, as an offset in the spine's local axes. At rest
// there is none.
func (rig *SealRig) fold(index int, pose SealRigPose) maths.Vec {
	vertex := rig.vertices[index]
	normal := rig.normals[index]
	amount := sealNeckFold(vertex.point, normal, pose.HeadPitch)
	if vertex.weight > 0 {
		limb := rig.limbs[vertex.limb]
		amount += sealRootFold(vertex.point.Sub(limb.root).Len(),
			pose.angle(vertex.limb))
	}
	if amount == 0 {
		return maths.Vec{}
	}
	lifted := pose.nod(vertex.point.Add(normal)).Sub(pose.nod(vertex.point))
	return lifted.Scale(amount)
}

// sealNeckFold is the height of the neck roll at a point whose skin faces
// along normal, for a head pitch. Only the inside of the bend rolls: the
// underside when the head nods down, the nape when it tips back.
func sealNeckFold(point, normal maths.Vec, pitch float64) float64 {
	inside := math.Copysign(1, pitch) * normal.Y
	window := maths.Smooth(sealNeckStart, sealNeckStart+sealNeckBlend, point.X) *
		(1 - maths.Smooth(sealNeckEnd-sealNeckBlend, sealNeckEnd, point.X))
	ripple := .5 * (1 - math.Cos(2*math.Pi*(point.X-sealNeckStart)/sealNeckPeriod))
	return sealNeckDepth * math.Abs(pitch) / sealNeckLimit * window * ripple *
		maths.Smooth(-.1, .6, inside)
}

// sealRootFold is the height of the ring a given distance from a flipper
// root, for the flipper's turn.
func sealRootFold(distance, angle float64) float64 {
	zone := maths.Smooth(sealRootNear, sealRootNear+sealRootBlend, distance) *
		(1 - maths.Smooth(sealRootFar-sealRootBlend, sealRootFar, distance))
	ripple := .5 * (1 - math.Cos(2*math.Pi*distance/sealRootPeriod))
	return sealRootDepth * math.Min(1, math.Abs(angle)/sealRootTurn) * zone * ripple
}

// sealStudyExtent conservatively includes the two drawn-rig studies' whiskers and flippers.
var sealStudyExtent = maths.Box{Min: maths.V(-1.3, -.8, -.8), Max: maths.V(1.3, .8, .8)}

// harborSealKind is a harbor seal with its anatomy and coat: the rigged
// skin, hair as blades along the coat's flow, five-digit flippers with
// claws, a short tail, ear openings, shut or open nostrils, vibrissae in
// rows, and a spotted pelage that reads wet or dry. It is a static
// specimen in a pose, not a clock.
type harborSealKind struct {
	look SealLook
	pose SealRigPose
}

// harborSealRelaxed is a seal at rest with the head a little tucked, so
// the neck shows its rolls, and the foreflippers swept back.
var harborSealRelaxed = SealRigPose{
	Yaw: .10, Pitch: .05, HeadPitch: -.50,
	Fore: [2]float64{-.30, -.22}, Hind: [2]float64{.10, -.10},
}

// made is the harbor seal in the kind's look, seeded from the seed. It
// refuses a seed that was not made.
func (kind harborSealKind) made(number uint64) (Result, error) {
	look := kind.look
	look.Seed = number
	rig := NewHarborSealRig()
	skin, err := rig.Skin(kind.pose)
	if err != nil {
		return Result{}, err
	}
	fur, err := rig.Fur(kind.pose, look)
	if err != nil {
		return Result{}, err
	}
	anatomy, err := rig.Parts(kind.pose, look)
	if err != nil {
		return Result{}, err
	}
	skinLook, hairLook := rig.Materials(look)
	parts := []thing.Thing{
		{Shape: skin, Material: skinLook, Motion: thing.Still(maths.Local())},
		{Shape: fur, Material: hairLook, Motion: thing.Still(maths.Local())},
	}
	parts = append(parts, anatomy...)
	// Conservative static extent includes the whiskers and the flippers.
	box := sealStudyExtent
	return Result{Parts: parts, Bounds: box}, nil
}

// harborSealBonesKind is the harbor seal's skeleton alone, at rest, in
// the rig's space: the study each step of the animal is built over.
type harborSealBonesKind struct{ profile pinnipedProfile }

// sealBone is the colour of dry bone, chosen.
var sealBone = maths.RGB(.62, .57, .48)

// made is the harbor seal's skeleton alone, bones without skin. It refuses
// a seed that was not made.
func (kind harborSealBonesKind) made() (Result, error) {
	made, err := newSealBones(kind.profile)
	if err != nil {
		return Result{}, err
	}
	points, faces := made.bones.Draw(made.bones.Frames(skeleton.Pose{}),
		made.forms)
	solid, err := mesh.NewWithRestPoints(points, points, faces, nil, false)
	if err != nil {
		return Result{}, err
	}
	look := &thing.Material{Name: "harbor seal bone", Colour: sealBone,
		Gloss: .08, Satin: .5}
	parts := []thing.Thing{{Shape: solid, Material: look,
		Motion: thing.Still(maths.Local())}}
	box := solid.Bounds()
	return Result{Parts: parts, Bounds: box}, nil
}

// baySealKind preserves the original bay scene's phase-zero bottling pose,
// materials, corneas and whiskers. It is a static specimen, not a clock.
type baySealKind struct{}

// made is the bay scene's seal as one posed specimen: skin, eyes and
// whiskers. It refuses a seed that was not made.
func (baySealKind) made() (Result, error) {
	rig := NewSealRig()
	pose := SealRigPose{Yaw: .18, Pitch: -.20, HeadPitch: -.86, Fore: [2]float64{-.26, -.20}}
	skin, err := rig.Skin(pose)
	if err != nil {
		return Result{}, err
	}
	cornea, err := SealCornea()
	if err != nil {
		return Result{}, err
	}
	coat := &thing.Material{Name: "wet dark harbor seal", Colour: maths.RGB(.068, .063, .055), Texture: baySealWetPaint{spots: HarborSealCoat(407)}, Gloss: .42, Satin: .33, Index: 1.43}
	eyeLook := &thing.Material{Name: "harbor seal eye", Colour: maths.RGB(.010, .008, .007), Gloss: .41, Satin: .28}
	whiskerLook := &thing.Material{Name: "wet muzzle whiskers", Colour: maths.RGB(.44, .37, .28), Gloss: .21, Satin: .4}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(maths.Local())}}
	for _, side := range [...]float64{-1, 1} {
		rest := maths.V(.730, .110, side*.064)
		point := rig.HeadPoint(pose, rest)
		look := rig.HeadPoint(pose, rest.Add(maths.V(.018, 0, side*.006)))
		up := rig.HeadPoint(pose, rest.Add(maths.V(0, .010, 0))).Sub(point)
		eye := maths.NewSubspace(point, maths.LookAtUp(point, look, up), 1)
		parts = append(parts, thing.Thing{Shape: cornea, Material: eyeLook, Motion: thing.Still(eye)})
		for whisker := range 5 {
			fan := float64(whisker)
			root := maths.V(.775-.006*fan, .037+.007*fan, side*(.072+.002*fan))
			previous := rig.HeadPoint(pose, root)
			for segment := 1; segment <= 3; segment++ {
				share := float64(segment) / 3
				tip := root.Add(maths.V((.022-.011*fan)*share, (.025-.011*fan)*share-.030*share*share, side*(.080+.008*fan)*share))
				at := rig.HeadPoint(pose, tip)
				middle := previous.Add(at).Scale(.5)
				turn := maths.Upright(at.Sub(previous))
				rod := maths.NewSubspace(middle, turn, 1)
				parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .00032, Thick: .00064, Height: at.Sub(previous).Len()}, Material: whiskerLook, Motion: thing.Still(rod)})
				previous = at
			}
		}
	}
	// Conservative static extent includes the whiskers and rotated corneas.
	box := sealStudyExtent
	return Result{Parts: parts, Bounds: box}, nil
}

type baySealWetPaint struct{ spots texture.Texture }

// At is the wet coat's colour at a point, with no footprint.
func (coat baySealWetPaint) At(point maths.Vec) maths.Light { return coat.Within(point, 0) }

// Within is the wet coat's colour at a point, over a footprint in metres:
// spots on a dark wet base, and the nose.
func (coat baySealWetPaint) Within(point maths.Vec, footprint float64) maths.Light {
	spots := texture.Sample(coat.spots, point, footprint)
	base := maths.RGB(.021, .019, .016)
	wet := base.Add(spots.Scale(.23))
	noseWidth := .028 + .024*maths.Smooth(.023, .077, point.Y)
	nose := 1 - maths.Smooth(.76, 1.02, math.Hypot((point.Y-.059)/.038, point.Z/noseWidth))
	nose *= maths.Smooth(.779, .804, point.X)
	nostril := 1 - maths.Smooth(.52, 1.02, math.Hypot((point.Y-.067+.33*(math.Abs(point.Z)-.022))/.007, (math.Abs(point.Z)-.022)/.013))
	noseColour := maths.RGB(.014, .012, .011).Scale(1 - .67*nostril)
	return wet.Scale(1 - nose).Add(noseColour.Scale(nose))
}

// Result is a made seal's immutable parts and conservative extent.
type Result struct {
	Parts  []thing.Thing
	Bounds maths.Box
}

func Detailed(look SealLook, number uint64) (Result, error) {
	return (harborSealKind{look: look, pose: harborSealRelaxed}).made(number)
}
func Bones(profile Profile) (Result, error) { return (harborSealBonesKind{profile: profile}).made() }
func Bay() (Result, error)                  { return (baySealKind{}).made() }
