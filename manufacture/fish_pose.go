package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// A fish swims by a wave that runs down its body from head to tail,
// growing as it goes: nothing ahead of where its species begins to bend
// but the small recoil of the head, and the most at the tail. The pose is
// that wave at one instant, made by turning each joint of the column by
// the change in the body's slope across it, about the up axis, so the
// body bends from side to side.
//
// A tuna begins to bend far back and a
// sardine near its head; the row says where.
//
// The posed fish is fixed meshes, its skin carried by the bones that hold
// it by nearness, its fins' webs by their rays, its eyes by its skull.

// How the stroke is drawn. Picked.
const (
	fishRecoil = .02  // picked: the head's swing, of the tail's
	fishSlope  = 1e-3 // picked: the share over which a slope is taken
	fishTurn   = 2 * math.Pi
)

// The cut of the fins' webs and the eyes. Chosen: enough to read.
const (
	fishWebSteps  = 4
	fishEyeRings  = 10
	fishEyeAround = 16
	fishEyeSunk   = .55   // picked: of an eye's radius, under the skin
	fishCornea    = 1.376 // picked: a cornea's index; not looked up
)

// fishPart is one surface of a posed fish: the look it wears by name,
// its points posed and at rest, and its triangles.
type fishPart struct {
	look         string
	points, rest []maths.Vec
	faces        [][3]int
	smooth       bool
}

// sway is how far the body's middle line stands to the side at share s,
// in a stroke whose tail swings amplitude of the length to either side,
// at a phase of the wave.
func (body *fishBody) sway(s, amplitude, phase float64) float64 {
	plan := body.plan
	from := plan.figure("bend-from")
	u := math.Max(0, (s-from)/(1-from))
	grow := fishRecoil + (1-fishRecoil)*u*u
	wave := fishTurn*s/plan.figure("wavelength") - phase
	return amplitude * body.outline.length * grow * math.Sin(wave)
}

// slope is the turn of the body's middle line at share s, from lying
// along the fish, as a turn about the up axis.
func (body *fishBody) slope(s, amplitude, phase float64) float64 {
	rise := body.sway(s+fishSlope, amplitude, phase) -
		body.sway(s-fishSlope, amplitude, phase)
	return math.Atan(rise / (2 * fishSlope * body.outline.length))
}

// stroke is the fish's pose at the seed's phase of a stroke whose tail
// swings amplitude of its length to either side: the whole fish turned
// at the skull's joint by the head's recoil, and each vertebra turned at
// its head by the change of slope from the one before.
func (body *fishBody) stroke(amplitude float64) skeleton.Pose {
	phase := fishTurn * body.plan.share("phase")
	skull, _ := body.bones.Find("neurocranium")
	condyle := body.bones.Bone(skull).Head()
	previous := body.slope(body.outline.shareAt(condyle.X), amplitude, phase)
	pose := skeleton.Pose{}.Carried(skeleton.TurnAbout(condyle, maths.YAxis,
		previous))
	for _, name := range body.column {
		at, _ := body.bones.Find(name)
		s := body.outline.shareAt(body.bones.Bone(at).Head().X)
		turn := body.slope(s, amplitude, phase)
		pose = pose.With(name, skeleton.NewTurn(maths.YAxis, turn-previous))
		previous = turn
	}
	return pose
}

// skin is the body's surface posed: each point carried by the bones
// that hold it, by how near it lies to each.
func (body *fishBody) skin(frames []skeleton.Frame, rings,
	around int) fishPart {
	rest, faces := body.outline.loft(rings, around)
	holders := body.holders()
	points := make([]maths.Vec, len(rest))
	for i, p := range rest {
		points[i] = skeleton.Carry(p, frames, skeleton.Nearness(p, holders))
	}
	return fishPart{look: "body", points: points, rest: rest, faces: faces,
		smooth: true}
}

// webs is the fins' webs posed, one surface for each look they wear.
func (body *fishBody) webs(frames []skeleton.Frame) ([]fishPart, error) {
	sheets := map[string]*skeleton.Sheet{}
	order := []string{}
	thick := body.plan.figure("fin-thickness") * body.outline.length
	for _, fin := range body.fins {
		sheet, known := sheets[fin.look]
		if !known {
			sheet = skeleton.NewSheet()
			sheets[fin.look] = sheet
			order = append(order, fin.look)
		}
		rays := make([]skeleton.WebRay, len(fin.rays))
		for i, ray := range fin.rays {
			at, _ := body.bones.Find(ray.bone)
			rays[i] = skeleton.NewWebRay(ray.base, ray.tip, frames[at])
		}
		if err := sheet.Web(rays, fishWebSteps, thick, fin.facing); err != nil {
			return nil, err
		}
	}
	parts := make([]fishPart, len(order))
	for i, look := range order {
		sheet := sheets[look]
		parts[i] = fishPart{look: look, points: sheet.Points(),
			rest: sheet.Rest(), faces: sheet.Faces()}
	}
	return parts, nil
}

// eye is where the left eye's middle lies at rest, and its radius; the
// right is its mirror.
func (body *fishBody) eye() (maths.Vec, float64) {
	plan, out := body.plan, body.outline
	s := plan.figure("eye-at") * body.head()
	y := plan.figure("eye-height") * out.halfDepth(s)
	radius := plan.figure("eye") * body.head() * out.length / 2
	return body.at(s, y, out.side(s, y)-fishEyeSunk*radius), radius
}

// eyes is both eyes posed with the skull: each a ball set into the head.
func (body *fishBody) eyes(frames []skeleton.Frame) fishPart {
	skull, _ := body.bones.Find("neurocranium")
	middle, radius := body.eye()
	part := fishPart{look: "eye", smooth: true}
	for _, side := range fishSides {
		centre := maths.V(middle.X, middle.Y, side*middle.Z)
		rest, faces := fishBall(centre, radius)
		for _, face := range faces {
			start := len(part.rest)
			part.faces = append(part.faces, [3]int{face[0] + start,
				face[1] + start, face[2] + start})
		}
		part.rest = append(part.rest, rest...)
	}
	for _, p := range part.rest {
		part.points = append(part.points, frames[skull].Apply(p))
	}
	return part
}

// fishBall is a ball about centre: a point at its front and back along
// x, and rings between.
func fishBall(centre maths.Vec, radius float64) ([]maths.Vec, [][3]int) {
	points := []maths.Vec{centre.Add(maths.V(radius, 0, 0))}
	for ring := 1; ring <= fishEyeRings; ring++ {
		polar := math.Pi * float64(ring) / (fishEyeRings + 1)
		along, round := radius*math.Cos(polar), radius*math.Sin(polar)
		for k := 0; k < fishEyeAround; k++ {
			angle := fishTurn * float64(k) / fishEyeAround
			points = append(points, centre.Add(maths.V(along,
				round*math.Sin(angle), round*math.Cos(angle))))
		}
	}
	points = append(points, centre.Sub(maths.V(radius, 0, 0)))
	return points, fishOutward(points, fishLoftFaces(fishEyeRings,
		fishEyeAround))
}

// posed is every surface of the fish in a pose, its skin cut by fine.
func (body *fishBody) posed(pose skeleton.Pose, fine bool) ([]fishPart,
	error) {
	frames := body.bones.Frames(pose)
	rings, around := fishSchoolRings, fishSchoolAround
	if fine {
		rings, around = fishRings, fishAround
	}
	webs, err := body.webs(frames)
	if err != nil {
		return nil, err
	}
	parts := append([]fishPart{body.skin(frames, rings, around)}, webs...)
	return append(parts, body.eyes(frames)), nil
}

// meshed is a part as slayer's mesh, with its rest points for its paint;
// shaded smooth where it is smooth and smooth is allowed.
func (part fishPart) meshed(smooth bool) (*mesh.Mesh, error) {
	smooth = smooth && part.smooth
	var normals []maths.Vec
	if smooth {
		normals = fishNormals(part.points, part.faces)
	}
	return mesh.NewWithRestPoints(part.points, part.rest, part.faces,
		normals, smooth)
}

// fleshGood is the fish in a pose as a good: each surface a mesh in its
// look, cut finely where fine.
func (body *fishBody) fleshGood(pose skeleton.Pose, fine bool) (good,
	error) {
	parts, err := body.posed(pose, fine)
	if err != nil {
		return good{}, err
	}
	looks := body.looks()
	var native []thing.Thing
	for _, part := range parts {
		made, err := part.meshed(true)
		if err != nil {
			return good{}, err
		}
		native = append(native, thing.Thing{Shape: made,
			Material: looks[part.look], Motion: thing.Still(maths.Local())})
	}
	return fishGood(native), nil
}

// bonesGood is the fish's skeleton at rest as a good, each bone in its
// form.
func (body *fishBody) bonesGood() (good, error) {
	frames := body.bones.Frames(skeleton.Pose{})
	points, faces := body.bones.Draw(frames, body.forms)
	made, err := mesh.New(points, faces, nil, false)
	if err != nil {
		return good{}, err
	}
	bone := body.looks()["bone"]
	return fishGood([]thing.Thing{{Shape: made, Material: bone,
		Motion: thing.Still(maths.Local())}}), nil
}

// fishGood is native surfaces as a good, its extent theirs.
func fishGood(native []thing.Thing) good {
	box := native[0].Shape.Bounds()
	for _, one := range native[1:] {
		box = enclosing(box, one.Shape.Bounds())
	}
	return good{native: native, extent: &box}
}
