package manufacture

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// HarborSealPose controls the side-to-side stroke and each pair of flippers.
// Stroke maps to [-1, 1]. Feather and Reach are signed radians, about the
// hind flippers' long axes and the fore flippers' roots, respectively.
type HarborSealPose struct {
	Stroke  float64
	Feather [2]float64
	Reach   [2]float64
}

// HarborSealSwim is the first shot's authored stroke, glide and bank.
// Geometry is local to the seal, with +X toward the muzzle and +Y up.
func HarborSealSwim(at time.Duration) (HarborSealPose, maths.Subspace) {
	seconds := math.Max(0, at.Seconds())
	const strokeEnd = 1.55
	phase := 4 * math.Pi * math.Min(seconds/strokeEnd, 1)
	fade := 1 - sealEase((seconds-1.18)/(strokeEnd-1.18))
	stroke := fade * math.Sin(phase)
	feather := 0.48 * fade * math.Cos(phase)
	bank := sealEase((seconds - 1.87) / 0.87)
	pose := HarborSealPose{
		Stroke: stroke, Feather: [2]float64{feather, -0.8 * feather},
		Reach: [2]float64{0.28 + 0.44*bank, 0.26 - 0.19*bank},
	}
	// Distance is the integral of cruising speed plus a smoothly
	// decaying acceleration from the stroke, in metres.
	progress := 0.53*seconds + 0.39*(1-math.Exp(-1.4*seconds))
	turn := maths.Rotation(maths.V(1, 0, 0), 0.42*bank).
		Then(maths.Rotation(maths.Up, -0.24*bank))
	origin := maths.V(-1.04+progress, 0.10*bank, 0.22*bank)
	return pose, maths.NewSubspace(origin, turn, 1)
}

// sealEase is a share clamped to nought to one and eased, so that it has
// no slope at either end.
func sealEase(value float64) float64 {
	value = math.Max(0, math.Min(1, value))
	return value * value * (3 - 2*value)
}

// sealSection locates an elliptical cross section along the seal's X axis.
// Dimensions are artistic choices in metres for a roughly 1.7 m animal.
type sealSection struct{ x, y, width, height float64 }

var harborSealProfile = []sealSection{
	{-0.84, -0.020, 0.029, 0.038},
	{-0.75, -0.010, 0.060, 0.066},
	{-0.63, -0.003, 0.112, 0.105},
	{-0.48, 0.002, 0.174, 0.159},
	{-0.28, 0.007, 0.224, 0.205},
	{-0.06, 0.019, 0.242, 0.222},
	{0.14, 0.039, 0.223, 0.205},
	{0.30, 0.052, 0.179, 0.172},
	{0.43, 0.060, 0.130, 0.134},
	{0.54, 0.082, 0.127, 0.130},
	{0.64, 0.084, 0.114, 0.110},
	{0.73, 0.056, 0.089, 0.075},
	{0.79, 0.048, 0.066, 0.051},
	{0.82, 0.049, 0.027, 0.028},
	{0.83, 0.053, 0.005, 0.008},
}

// HarborSeal makes the body, four flippers and eyes at a posed instant.
// Each skin is closed and wound by the same solid loft used in Manufactory.
func HarborSeal(pose HarborSealPose, at maths.Subspace) ([]thing.Thing, error) {
	for _, control := range [...]float64{pose.Stroke,
		pose.Feather[0], pose.Feather[1], pose.Reach[0], pose.Reach[1]} {
		if math.IsNaN(control) || math.Abs(control) > 1 {
			return nil, fmt.Errorf("harbor seal: pose control outside [-1, 1]")
		}
	}
	coat := &thing.Material{Name: "seal coat", Colour: maths.RGB(.32, .32, .30), Gloss: .21, Satin: .47}
	flipper := &thing.Material{Name: "seal flipper", Colour: maths.RGB(.23, .23, .22), Gloss: .17, Satin: .52}
	eye := &thing.Material{Name: "seal eye", Colour: maths.RGB(.016, .013, .011), Gloss: .48}
	pieces := make([]thing.Thing, 0, 7)
	body, err := sealPiece(pose.body(), coat, at)
	if err != nil {
		return nil, fmt.Errorf("harbor seal body: %w", err)
	}
	pieces = append(pieces, body)
	for index, side := range [...]float64{-1, 1} {
		for _, rings := range [][][]maths.Vec{pose.fore(index, side), pose.hind(index, side)} {
			piece, err := sealPiece(rings, flipper, at)
			if err != nil {
				return nil, fmt.Errorf("harbor seal flipper: %w", err)
			}
			pieces = append(pieces, piece)
		}
		placement := maths.NewSubspace(maths.V(.647, .115, side*.103), maths.Identity, 1)
		pieces = append(pieces, thing.Thing{
			Shape: shape.Sphere{Radius: .026}, Material: eye,
			Motion: thing.Still(placement.Within(at)),
		})
	}
	return pieces, nil
}

// sealPiece is one part of the seal: rings lofted into a smooth skin in a
// material, set at a place. It returns the error of the shading.
func sealPiece(rings [][]maths.Vec, material *thing.Material, at maths.Subspace) (thing.Thing, error) {
	loft := alfaLoft{rings: rings}
	skin := loft.skin(cut{})
	solid, err := skin.shaded(loft.normals(skin))
	return thing.Thing{Shape: solid, Material: material, Motion: thing.Still(at)}, err
}

// bend moves a point of the body as the pose's Stroke bends it: sideways
// and turned, more the further aft of the shoulders, which stay settled.
func (pose HarborSealPose) bend(point maths.Vec) maths.Vec {
	// The shoulders stay settled. Lateral excursion builds down the
	// lumbar spine, and each cross section stays square to the spine.
	aft := math.Max(0, (0.27-point.X)/1.11)
	lateral := pose.Stroke * 0.19 * aft * aft
	yaw := math.Atan(pose.Stroke * 0.38 * aft / 1.11)
	return maths.V(point.X+point.Z*math.Sin(yaw), point.Y,
		lateral+point.Z*math.Cos(yaw))
}

// body is the rings of the body: the profile subdivided, then bent by the
// pose.
func (pose HarborSealPose) body() [][]maths.Vec {
	const subdivisions = 4
	profile := harborSealProfile
	rings := make([][]maths.Vec, 0, (len(profile)-1)*subdivisions+1)
	for index := 0; index < len(profile)-1; index++ {
		before := profile[max(0, index-1)]
		low, high := profile[index], profile[index+1]
		after := profile[min(len(profile)-1, index+2)]
		for step := range subdivisions {
			share := float64(step) / subdivisions
			section := sealSection{
				x:      low.x + (high.x-low.x)*share,
				y:      alfaHermite(before.y, low.y, high.y, after.y, share),
				width:  alfaHermite(before.width, low.width, high.width, after.width, share),
				height: alfaHermite(before.height, low.height, high.height, after.height, share),
			}
			rings = append(rings, pose.bodyRing(section))
		}
	}
	return append(rings, pose.bodyRing(profile[len(profile)-1]))
}

// bodyRing is the ring round the body at one section, bent by the pose.
func (pose HarborSealPose) bodyRing(section sealSection) []maths.Vec {
	return sealRing(maths.V(section.x, section.y, 0), maths.V(0, section.height, 0),
		maths.V(0, 0, section.width), pose.bend)
}

// sealFinSection is a flipper section, from its root toward the tip, with
// the lean of its outline built into the centres of the loft.
type sealFinSection struct{ x, y, z, chord, thick float64 }

var sealFore = []sealFinSection{
	{.12, -.099, .145, .085, .038},
	{.066, -.131, .201, .103, .032},
	{-.016, -.162, .251, .098, .019},
	{-.105, -.173, .286, .053, .010},
	{-.158, -.172, .295, .007, .003},
}

var sealHind = []sealFinSection{
	{-.755, -.030, .043, .039, .037},
	{-.830, -.038, .066, .051, .030},
	{-.914, -.039, .087, .075, .019},
	{-.978, -.043, .100, .076, .013},
	{-1.024, -.040, .103, .046, .007},
	{-1.050, -.038, .105, .008, .003},
}

// fore is the rings of a fore flipper by index, reached out by the pose's
// Reach; side is the sign of z.
func (pose HarborSealPose) fore(index int, side float64) [][]maths.Vec {
	base := sealFore[0]
	root := maths.V(base.x, base.y, side*base.z)
	turn := maths.Rotation(maths.Up, side*pose.Reach[index])
	return sealFin(sealFore, side, true, func(point maths.Vec, _ float64) maths.Vec {
		return pose.bend(root.Add(turn.Out(point.Sub(root))))
	})
}

// hind is the rings of a hind flipper by index, its digits feathered by the
// pose's Feather; side is the sign of z.
func (pose HarborSealPose) hind(index int, side float64) [][]maths.Vec {
	return sealFin(sealHind, side, false, func(point maths.Vec, share float64) maths.Vec {
		angle := side * pose.Feather[index] * share
		centre := side * (0.043 + 0.062*share)
		offset := maths.V(0, point.Y+0.038, point.Z-centre)
		feathered := maths.Rotation(maths.V(1, 0, 0), angle).Out(offset)
		return pose.bend(maths.V(point.X, feathered.Y-0.038, centre+feathered.Z))
	})
}

// sealFin lofts a flipper's sections into rings, each point moved by the
// pose function from the point and its share along the fin.
func sealFin(sections []sealFinSection, side float64, fore bool,
	pose func(maths.Vec, float64) maths.Vec,
) [][]maths.Vec {
	rings := make([][]maths.Vec, 0, len(sections))
	for index, row := range sections {
		share := float64(index) / float64(len(sections)-1)
		span := maths.V(0, 0, row.chord)
		if fore {
			span = maths.V(row.chord, 0, 0)
		}
		ring := sealRing(maths.V(row.x, row.y, side*row.z),
			maths.V(0, row.thick, 0), span,
			func(point maths.Vec) maths.Vec { return pose(point, share) })
		rings = append(rings, ring)
	}
	return rings
}

// sealRing is a ring of corners round an ellipse at a centre, with up and
// across its semi-axes, each corner moved by bend.
func sealRing(centre, up, across maths.Vec, bend func(maths.Vec) maths.Vec) []maths.Vec {
	const corners = 24
	ring := make([]maths.Vec, corners)
	for index := range ring {
		sine, cosine := math.Sincos(2 * math.Pi * float64(index) / corners)
		ring[index] = bend(centre.Add(up.Scale(cosine)).Add(across.Scale(sine)))
	}
	return ring
}
