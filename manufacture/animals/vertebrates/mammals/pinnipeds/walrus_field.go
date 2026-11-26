package pinnipeds

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// The face is cut into the outline: a shallow hollow round each eye, a
// pit for each nostril, and the groove between the two muzzle pads. Each
// is placed on the outline's own surface, so a head of another size
// carries them with it. Widths and depths are picked; a depth is in the
// density's units, and walrusCutDepth says how far in metres it moves
// the hide.
const (
	// walrusEyeStation, walrusEyeDrop and walrusEyeOut place the eye in the
	// skull's frame: its share of the skull's length forward of the
	// condyle, its drop below the axis (above it, so less than nought) and
	// about how far out from the middle it lies, in metres. Picked: small
	// eyes set high and wide, behind the pads.
	walrusEyeStation = .47
	walrusEyeDrop    = -.08
	walrusEyeOut     = .17
	// walrusNostrilStation, walrusNostrilDrop and walrusNostrilOut place a
	// nostril on top of the muzzle, behind the pads' face. Picked.
	walrusNostrilStation = 1.04
	walrusNostrilDrop    = -.06
	walrusNostrilOut     = .04
	// walrusEyeWidth and walrusEyeDepth are the hollow round an eye.
	walrusEyeWidth = .05
	walrusEyeDepth = .10
	// walrusNostrilWidth and walrusNostrilDepth are a nostril's pit.
	walrusNostrilWidth = .020
	walrusNostrilDepth = .45
)

// walrusEyeSocket is about where each eye lies, in the skull's frame:
// high on the side of the head. The eye itself is set on the hide near
// here, by walrusEyeOn and walrusEyeAt.
func (rig *walrusRig) walrusEyeSocket(side float64) maths.Vec {
	return rig.frame.at(walrusEyeStation, side*walrusEyeOut, walrusEyeDrop)
}

// walrusNostril is about where each nostril opens, in the skull's frame:
// on the top of the muzzle, over the front of the premaxilla.
func (rig *walrusRig) walrusNostril(side float64) maths.Vec {
	return rig.frame.at(walrusNostrilStation, side*walrusNostrilOut,
		walrusNostrilDrop)
}

// walrusEyeOn is the eye's place on the outline's own surface: at the
// socket's place along the body and its height, as far out as the
// outline's flank lies there. It refuses a socket the outline has no
// flank beside.
func (rig *walrusRig) walrusEyeOn(outline *walrusOutline, side float64) (maths.Vec, error) {
	socket := rig.walrusEyeSocket(side)
	out, ok := outline.flank(socket.X, socket.Y)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the outline has no flank beside "+
			"the eye on side %g", errWalrusLandmark, side)
	}
	return maths.V(socket.X, socket.Y, side*out), nil
}

// walrusNostrilOn is a nostril's place on the outline's own surface: at
// the pit's place along the body and across it, on the outline's back. It
// refuses a pit the outline has no back over.
func (rig *walrusRig) walrusNostrilOn(outline *walrusOutline, side float64) (maths.Vec, error) {
	pit := rig.walrusNostril(side)
	top, ok := outline.back(pit.X, pit.Z)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the outline has no back over "+
			"the nostril on side %g", errWalrusLandmark, side)
	}
	return maths.V(pit.X, top, pit.Z), nil
}

// walrusPit is a round dip of the given width and depth at centre.
func walrusPit(p, centre maths.Vec, width, depth float64) float64 {
	far := p.Sub(centre).Len() / width
	return depth * math.Exp(-far*far)
}

// walrusCleft cuts the shallow groove down the middle of the muzzle that
// divides the two upper lip pads, in the skull's frame. Picked.
func (rig *walrusRig) walrusCleft(p maths.Vec) float64 {
	const (
		start, width, depth = .85, .016, .14
		high, low           = -.12, .21
	)
	t, side, down := rig.frame.local(p)
	if t < start || down < high || down > low {
		return 0
	}
	off := side / width
	return depth * math.Exp(-off*off)
}

// walrusFaceOf is every pit and dip of the head on the outline given: the
// hollows round the eyes, the nostrils and the cleft between the pads.
// Nothing behind the back of the skull is cut. It refuses an outline that
// one of them does not lie on.
func (rig *walrusRig) walrusFaceOf(outline *walrusOutline) (func(maths.Vec) float64, error) {
	const behind = -.25
	var eyes, nostrils [2]maths.Vec
	for i, side := range []float64{-1, 1} {
		eye, err := rig.walrusEyeOn(outline, side)
		if err != nil {
			return nil, err
		}
		nostril, err := rig.walrusNostrilOn(outline, side)
		if err != nil {
			return nil, err
		}
		eyes[i], nostrils[i] = eye, nostril
	}
	return func(p maths.Vec) float64 {
		if t, _, _ := rig.frame.local(p); t < behind {
			return 0
		}
		total := rig.walrusCleft(p)
		for i := range eyes {
			total += walrusPit(p, eyes[i], walrusEyeWidth, walrusEyeDepth)
			total += walrusPit(p, nostrils[i], walrusNostrilWidth,
				walrusNostrilDepth)
		}
		return total
	}, nil
}

// walrusDensity is the summed density of every mass at p.
func walrusDensity(masses []walrusMass, p maths.Vec) float64 {
	density := 0.
	for _, mass := range masses {
		density += mass.density(p)
	}
	return density
}

// walrusOutlineReach is how far, in metres, the outline's density falls
// by a factor of e outside it: the width over which a flipper blends into
// the body. Chosen.
const walrusOutlineReach = .04

// walrusBodyDensity is the density of the body at p: the outline's, which
// is walrusThreshold on the outline itself and falls off outside it, and
// the masses' added to it, so that the masses join the outline smoothly
// and leave it alone where they are not.
func walrusBodyDensity(outline *walrusOutline, masses []walrusMass,
	p maths.Vec) float64 {
	gap := math.Max(outline.distance(p), -walrusOutlineReach*walrusOutlineDeep)
	return walrusThreshold*math.Exp(-gap/walrusOutlineReach) +
		walrusDensity(masses, p)
}

// walrusOutlineDeep is how many reaches inside the outline its density
// stops growing, so that it stays a finite number. Chosen.
const walrusOutlineDeep = 20

const (
	// walrusFaint is the least density the height is taken of, so that
	// far from the body it stays a finite number. Chosen.
	walrusFaint = 1e-12
	// walrusFloorRim is the width over which the edge is rounded where
	// the belly meets the ground, in metres. Chosen.
	walrusFloorRim = .02
)

// walrusHeight turns a density into the field the hide is meshed from:
// nought where the density is walrusThreshold, negative inside. Where
// only the outline gives the density it is the distance outside the
// outline, in metres, so it runs evenly through the hide and the mesh's
// straight-line placing of each point leaves no ripple at the grid's
// spacing, as the density itself, which is curved there, would.
func walrusHeight(density float64) float64 {
	return -walrusOutlineReach *
		math.Log(math.Max(density, walrusFaint)/walrusThreshold)
}

// walrusCutDepth is how far, in metres, a cut of the given depth moves
// the hide in: the height at which the density is walrusThreshold and
// the cut's depth together.
func walrusCutDepth(cut float64) float64 {
	return walrusOutlineReach * math.Log(1+cut/walrusThreshold)
}

// walrusOnGround is the field of a body lying on the ground: hide, a
// height, with everything under the ground cut away and the edge
// between them rounded.
func walrusOnGround(hide float64, p maths.Vec) float64 {
	return walrusSoftMax(hide, walrusFloor-p.Y, walrusFloorRim)
}

// walrusVolumeField is the field of the outline and the masses, flattened
// on the ground: no face cut. It shows the form before any detail.
func walrusVolumeField(outline *walrusOutline,
	masses []walrusMass) func(maths.Vec) float64 {
	return func(p maths.Vec) float64 {
		return walrusOnGround(
			walrusHeight(walrusBodyDensity(outline, masses, p)), p)
	}
}

// walrusSoftMax is the larger of two numbers with the corner between them
// rounded over round.
func walrusSoftMax(a, b, round float64) float64 {
	return .5 * (a + b + math.Sqrt((a-b)*(a-b)+round*round))
}

// walrusField is the field whose zero is the hide: negative inside. It is
// the outline with the masses joined to it, the face's pits cut and the
// neck and shoulders creased; on the ground, it flattens the underside
// where the body's weight rests.
func (rig *walrusRig) walrusField(outline *walrusOutline, masses []walrusMass,
	ground bool) (func(maths.Vec) float64, error) {
	face, err := rig.walrusFaceOf(outline)
	if err != nil {
		return nil, err
	}
	creases := walrusCreasesOf(outline)
	return func(p maths.Vec) float64 {
		hide := walrusHeight(walrusBodyDensity(outline, masses, p)) +
			walrusCutDepth(face(p)+creases(p))
		if !ground {
			return hide
		}
		return walrusOnGround(hide, p)
	}, nil
}

const (
	// walrusTuskSections and walrusTuskRadial are the rings along a tusk
	// and the points round each ring.
	walrusTuskSections = 60
	walrusTuskRadial   = 32
	// walrusTuskBase and walrusTuskTip are the tusk's radius at the root,
	// in the jaw, and just before its worn end, in metres.
	walrusTuskBase = .046
	walrusTuskTip  = .012
	// walrusTuskCap is how far the worn end rounds out beyond the last
	// ring.
	walrusTuskCap = .008
)

// walrusTuskRadius is the tusk's radius at t along it and angle round it.
// The surface carries fine growth rings and shallow ridges along its
// length, and narrows to a worn tip.
func walrusTuskRadius(t, angle float64) float64 {
	const (
		taper      = 1.2
		ridgeDepth = .04
		ridgeCount = 6
		ridgeTwist = 2.
		ringDepth  = .015
		ringCount  = 18
		fullTurn   = 2 * math.Pi
	)
	base := walrusTuskTip + (walrusTuskBase-walrusTuskTip)*math.Pow(1-t, taper)
	ridge := 1 - ridgeDepth*(1+math.Cos(ridgeCount*angle+ridgeTwist*t))/2
	ring := 1 + ringDepth*math.Sin(fullTurn*ringCount*t)
	return base * ridge * ring
}

// walrusTusk is one tusk, a closed ring-by-ring mesh whose root sits in
// the pit cut for it in the upper jaw. The last two vertices are the middle
// of the root and the middle of the worn end. Every vertex is moved by skin
// so the tusk follows the head, and keeps its rest place so the ivory's
// stain and rings are read along the tusk as it lies at rest.
func (rig *walrusRig) walrusTusk(side float64, skin func(maths.Vec) maths.Vec) (*mesh.Mesh, error) {
	const sections, radial = walrusTuskSections, walrusTuskRadial
	rest := make([]maths.Vec, 0, (sections+1)*radial+2)
	fs := make([][3]int, 0, 2*sections*radial+2*radial)
	canine, ok := rig.body.forms["tusk "+walrusSideName(side)].(walrusCanine)
	if !ok {
		return nil, errWalrusForm
	}
	path := canine.at
	for i := 0; i <= sections; i++ {
		t := float64(i) / sections
		centre := path(t)
		across, normal := skeleton.SweepFrame(path, t, maths.V(1, 0, 0))
		for j := 0; j < radial; j++ {
			angle := 2 * math.Pi * float64(j) / radial
			r := walrusTuskRadius(t, angle)
			// normal first, then across, so the rings turn the way the
			// faces below are wound.
			ring := normal.Scale(r * math.Cos(angle)).Add(across.Scale(r * math.Sin(angle)))
			rest = append(rest, centre.Add(ring))
		}
	}
	fs = geometry.AppendIndexedLoftSides(fs, 0, sections+1, radial)
	root := len(rest)
	rest = append(rest, path(0))
	end := len(rest)
	tip := path(1)
	rounded := tip.Add(tip.Sub(path(1 - 1./sections)).Unit().Scale(walrusTuskCap))
	rest = append(rest, rounded)
	for j := 0; j < radial; j++ {
		k := (j + 1) % radial
		fs = append(fs, [3]int{root, k, j},
			[3]int{end, sections*radial + j, sections*radial + k})
	}
	posed := make([]maths.Vec, len(rest))
	for i, p := range rest {
		posed[i] = skin(p)
	}
	return mesh.NewWithRestPoints(posed, rest, fs, nil, false)
}

// walrusLandmarkReach is how far either side of a landmark's own place the
// hide is looked for, in metres. Chosen: wider than the eye's or a
// nostril's pit.
const walrusLandmarkReach = .10

// walrusSurfaceZ is the depth at which the hide crosses the line at x and
// y, going out on side from near to far (both distances from the middle),
// or false if the hide does not cross between them.
func walrusSurfaceZ(field func(maths.Vec) float64, x, y, side,
	near, far float64) (float64, bool) {
	const steps = 30
	if field(maths.V(x, y, side*near)) >= 0 ||
		field(maths.V(x, y, side*far)) < 0 {
		return 0, false
	}
	low, high := near, far
	low, high = maths.Bisect(low, high, steps, func(z float64) bool { return field(maths.V(x, y, side*z)) < 0 })
	return side * (low + high) / 2, true
}

// walrusSurfaceY is the height at which the hide crosses the line at x and
// z between bottom and top, looking down from above, or false if it does
// not cross between them.
func walrusSurfaceY(field func(maths.Vec) float64, x, z, bottom,
	top float64) (float64, bool) {
	const steps = 26
	if field(maths.V(x, top, z)) < 0 || field(maths.V(x, bottom, z)) >= 0 {
		return 0, false
	}
	low, high := bottom, top
	low, high = maths.Bisect(low, high, steps, func(y float64) bool { return field(maths.V(x, y, z)) < 0 })
	return (low + high) / 2, true
}

// walrusGradient is the outward direction of the hide at p.
func walrusGradient(field func(maths.Vec) float64, p maths.Vec) maths.Vec {
	const step = .004
	grad := maths.V(
		field(p.Add(maths.V(step, 0, 0)))-field(p.Add(maths.V(-step, 0, 0))),
		field(p.Add(maths.V(0, step, 0)))-field(p.Add(maths.V(0, -step, 0))),
		field(p.Add(maths.V(0, 0, step)))-field(p.Add(maths.V(0, 0, -step))))
	if grad.Len() == 0 {
		return maths.V(1, 0, 0)
	}
	return grad.Unit()
}

const (
	// walrusThreshold is the summed density at which the hide begins.
	walrusThreshold = .30
	// walrusFalloff is the gaussian exponent of one mass's density.
	walrusFalloff = 2.1
	// walrusCutoff is how many radii out a mass is taken to be zero.
	walrusCutoff = 3.0
)

// walrusMass is one smooth lump of the body: a gaussian density over an
// ellipsoid whose axes need not line up with the world. The hide is where
// the summed density of every mass falls to walrusThreshold.
type walrusMass struct {
	centre   maths.Vec
	axes     [3]maths.Vec
	radius   [3]float64
	strength float64
	cutoff   float64
}

// walrusReach is how far, in radii, the hide lies from the centre of one
// mass of the given strength standing alone.
func walrusReach(strength float64) float64 {
	return math.Sqrt(math.Log(strength/walrusThreshold) / walrusFalloff)
}

// walrusBlob is a mass lined up with the world whose own hide would reach
// half out from its centre along each axis.
func walrusBlob(centre, half maths.Vec, strength float64) walrusMass {
	reach := walrusReach(strength)
	radius := [3]float64{half.X / reach, half.Y / reach, half.Z / reach}
	return walrusMass{
		centre: centre,
		axes: [3]maths.Vec{
			maths.V(1, 0, 0), maths.V(0, 1, 0), maths.V(0, 0, 1),
		},
		radius:   radius,
		strength: strength,
		cutoff:   walrusCutoff * math.Max(radius[0], math.Max(radius[1], radius[2])),
	}
}

// walrusPaddle is a flat mass lying along direction, with its thin axis
// as near to vertical as the direction allows. Half holds the half
// length along the direction, the half thickness and the half width.
func walrusPaddle(centre, direction, half maths.Vec, strength float64) walrusMass {
	along := direction.Unit()
	up := maths.V(0, 1, 0)
	thin := up.Sub(along.Scale(up.Dot(along))).Unit()
	wide := along.Cross(thin)
	reach := walrusReach(strength)
	radius := [3]float64{half.X / reach, half.Y / reach, half.Z / reach}
	return walrusMass{
		centre:   centre,
		axes:     [3]maths.Vec{along, thin, wide},
		radius:   radius,
		strength: strength,
		cutoff:   walrusCutoff * math.Max(radius[0], math.Max(radius[1], radius[2])),
	}
}

// density is this mass's share of the summed density at p.
func (mass walrusMass) density(p maths.Vec) float64 {
	offset := p.Sub(mass.centre)
	if math.Abs(offset.X) > mass.cutoff || math.Abs(offset.Y) > mass.cutoff ||
		math.Abs(offset.Z) > mass.cutoff {
		return 0
	}
	sum := 0.
	for i, axis := range mass.axes {
		along := offset.Dot(axis) / mass.radius[i]
		sum += along * along
	}
	return mass.strength * math.Exp(-walrusFalloff*sum)
}

// walrusSegmentDistance is how far p is from the nearest point of the
// segment from a to b.
func walrusSegmentDistance(p, a, b maths.Vec) float64 {
	axis := b.Sub(a)
	share := p.Sub(a).Dot(axis) / axis.Dot(axis)
	share = math.Max(0, math.Min(1, share))
	return p.Sub(a.Add(axis.Scale(share))).Len()
}

// walrusHash is a number from nought to one that is the same each time for
// the same lattice point and salt.
func walrusHash(x, y, z, salt int) float64 {
	const (
		first, second, third, fourth = 73856093, 19349663, 83492791, 2654435761
		mix                          = 0x45d9f3b
		shift                        = 15
		span                         = 1 << 24
	)
	bits := uint32(x)*first ^ uint32(y)*second ^ uint32(z)*third ^
		uint32(salt)*fourth
	bits ^= bits >> shift
	bits *= mix
	bits ^= bits >> shift
	return float64(bits%span) / span
}

// walrusNoise is smooth value noise at p, from nought to one, with one
// lattice point to the unit.
func walrusNoise(p maths.Vec) float64 {
	x, y, z := math.Floor(p.X), math.Floor(p.Y), math.Floor(p.Z)
	fx, fy, fz := maths.Smooth(0, 1, p.X-x), maths.Smooth(0, 1, p.Y-y),
		maths.Smooth(0, 1, p.Z-z)
	ix, iy, iz := int(x), int(y), int(z)
	row := func(dy, dz int) float64 {
		low := walrusHash(ix, iy+dy, iz+dz, 0)
		high := walrusHash(ix+1, iy+dy, iz+dz, 0)
		return low*(1-fx) + high*fx
	}
	near := row(0, 0)*(1-fy) + row(1, 0)*fy
	far := row(0, 1)*(1-fy) + row(1, 1)*fy
	return near*(1-fz) + far*fz
}

// The flippers' flesh is laid on their bones: the muscle of the upper arm
// and thigh inside the body wall, the forearm and shank, then on every
// bone of the palm or sole and of each digit a flat mass as wide as the
// space between rays, so neighbouring rays meet in one web, and a thin
// rim of web beyond the last bone. Sizes chosen.
const (
	// walrusWebThickness is the half thickness of the flipper over the
	// palm or sole; it thins to walrusWebTip at the end of each digit.
	walrusWebThickness = .028
	walrusWebTip       = .008
	// walrusWebShare is the half width of a ray's flesh as a share of the
	// spacing between rays: over one, so that neighbouring rays run into
	// one smooth paddle with no groove between them.
	walrusWebShare = 1.15
	// walrusWebRim is how far the web runs past the last bone of a digit.
	walrusWebRim = .05
)

// walrusLimbMasses is the flesh of all four flippers.
func walrusLimbMasses(body *walrusBones) []walrusMass {
	var masses []walrusMass
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		masses = append(masses, walrusArmFlesh(body, name)...)
		masses = append(masses, walrusRays(body, "metacarpal", "fore", name,
			walrusForeDigits, walrusForeSpacing)...)
		masses = append(masses, walrusLegFlesh(body, name)...)
		masses = append(masses, walrusRays(body, "metatarsal", "hind", name,
			walrusHindDigits, walrusHindSpacing)...)
	}
	return masses
}

// walrusBoneLine is the two ends of the straight bone of that name.
func walrusBoneLine(body *walrusBones, name string) (maths.Vec, maths.Vec) {
	return body.rod(name)
}

// walrusFleshOn is a flat mass along the bone from a to b, its half
// thickness thick and half width wide, reaching past each end by over.
func walrusFleshOn(a, b maths.Vec, thick, wide, over, strength float64) walrusMass {
	length := b.Sub(a).Len()
	return walrusPaddle(a.Add(b).Scale(.5), b.Sub(a),
		maths.V(length/2+over, thick, wide), strength)
}

// walrusArmFlesh is the muscle round the forearm, and the pad over the
// wrist. The upper arm lies inside the body's outline and adds nothing to
// it.
func walrusArmFlesh(body *walrusBones, side string) []walrusMass {
	_, elbow := walrusBoneLine(body, "humerus "+side)
	_, wrist := walrusBoneLine(body, "radius "+side)
	return []walrusMass{
		walrusFleshOn(elbow, wrist, .10, .12, .03, 1),
		walrusFleshOn(wrist, wrist.Add(wrist.Sub(elbow).Unit().Scale(.10)),
			.06, .12, .02, 1),
	}
}

// walrusLegFlesh is the muscle round the shank, and the pad over the ankle
// and heel. The thigh lies inside the body's outline and adds nothing to
// it.
func walrusLegFlesh(body *walrusBones, side string) []walrusMass {
	_, knee := walrusBoneLine(body, "femur "+side)
	_, ankle := walrusBoneLine(body, "tibia "+side)
	heel, toe := walrusBoneLine(body, "calcaneus "+side)
	return []walrusMass{
		walrusFleshOn(knee, ankle, .10, .13, .03, 1),
		walrusFleshOn(heel, toe, .045, .09, .03, 1),
	}
}

// walrusRays is the flesh on every ray of one flipper: on its palm or
// sole bone and on each phalanx, thinning toward the digit's end, and the
// web's rim beyond the last bone.
func walrusRays(body *walrusBones, palm, kind, side string,
	digits [walrusDigitCount][]float64, spacing float64) []walrusMass {
	var masses []walrusMass
	wide := spacing * walrusWebShare
	for ray := 1; ray <= walrusDigitCount; ray++ {
		names := walrusRayNames(palm, kind, side, ray, digits)
		var tip, along maths.Vec
		for i, name := range names {
			share := float64(i) / float64(len(names))
			thick := walrusWebThickness + (walrusWebTip-walrusWebThickness)*share
			from, to := walrusBoneLine(body, name)
			masses = append(masses, walrusFleshOn(from, to, thick, wide, .01, 1))
			tip, along = to, to.Sub(from).Unit()
		}
		rim := tip.Add(along.Scale(walrusWebRim))
		masses = append(masses, walrusFleshOn(tip, rim, walrusWebTip, wide, 0, 1))
	}
	return masses
}

// walrusRayNames is the names of the bones of one ray, from its palm or
// sole bone to the last phalanx.
func walrusRayNames(palm, kind, side string, ray int,
	digits [walrusDigitCount][]float64) []string {
	names := []string{fmt.Sprintf("%s %d %s", palm, ray, side)}
	for bone := 1; bone <= len(digits[ray-1]); bone++ {
		names = append(names,
			fmt.Sprintf("%s phalanx %d.%d %s", kind, ray, bone, side))
	}
	return names
}

// The eye is small and set high on the side of the head, in a hollow over
// the orbit, with an upper and a lower lid of thick skin round it. The
// globe shows as a wet, dark surface. Sizes chosen.
const (
	// walrusLidGap is how far above and below the eye's centre each lid
	// lies along the hide, in metres.
	walrusLidGap = .021
	// walrusCorneaIndex is the refractive index given the wet surface of
	// the eye: that of water, for the tear film over it. Chosen.
	walrusCorneaIndex = 1.333
)

// walrusLids is the upper and lower lid of each eye as masses on the
// outline's surface: each a short roll of skin lying fore and aft, above
// and below the eye. It refuses an outline the eyes do not lie on.
func (rig *walrusRig) walrusLids(outline *walrusOutline) ([]walrusMass, error) {
	var masses []walrusMass
	for _, side := range []float64{-1, 1} {
		eye, err := rig.walrusEyeOn(outline, side)
		if err != nil {
			return nil, err
		}
		for _, lid := range []float64{-1, 1} {
			centre := eye.Add(maths.V(0, lid*walrusLidGap, 0))
			masses = append(masses,
				walrusBlob(centre, maths.V(.034, .009, .014), .9))
		}
	}
	return masses, nil
}

// walrusCorneaMaterial is the eye's surface: dark, wet and glossy, with
// the tear film's index.
func walrusCorneaMaterial() *thing.Material {
	return &thing.Material{Name: "walrus eye", Colour: maths.RGB(.012, .009, .008),
		Gloss: .95, Satin: .1, Index: walrusCorneaIndex}
}
