package pinnipeds

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// sealStation is one cross-section of a seal's outline at rest: where
// it lies along the body (+X toward the nose, metres), the half height
// and half width of its ellipse, and the height of the ellipse's centre.
type sealStation struct{ x, high, wide, centre float64 }

// sealOutline is a seal's outline as stations from the tail's tip to
// the nose, in the rest space every lofted seal shares.
type sealOutline []sealStation

// The rest space's length, nose to the tail's tip on the rig's X axis:
// 1.75 m, the harbor seal's, inside the adult male's 161 to 186 cm
// (National Museum of Nature and Science, Tokyo, Marine Mammals
// Information Database, harbor seal). Picked so the skeleton's ankles
// lie at the ankle site. Another seal is drawn in it and then scaled.
const (
	sealNoseX    = .83
	sealTailTipX = -.92
	sealLength   = sealNoseX - sealTailTipX
)

// sealSourceSite is one girth site of a measured phocid: its name, how
// far behind the nose it lies in a straight line, and the radius of the
// circle of its girth, both in centimetres as the source prints them.
type sealSourceSite struct {
	Name          string
	Behind, Round float64
}

type sealSource struct {
	Citation         string
	Sites            []sealSourceSite
	Length, Axillary float64
}

// place is where a site lies on the rest space's X axis: the same share
// of the length behind the nose.
func (source sealSource) place(site sealSourceSite) float64 {
	return sealNoseX - sealLength*site.Behind/source.Length
}

// radius is the mean radius a site gives a seal whose own axillary
// radius is axillary, in metres: the source's radius there as a share
// of its axillary radius, times the seal's.
func (source sealSource) radius(site sealSourceSite,
	axillary float64,
) float64 {
	return axillary * site.Round / source.Axillary
}

// harborSealOutline is the harbor seal's outline: one spindle, with no
// waist between head and body. The eight named sites are the source's,
// each with its mean radius, the neck and the ears a little over; how
// much wider than high each is, the centre heights, and the muzzle and
// tail stations between the sites and the ends are picked.
var harborSealOutline = defaultPinniped("harbor-seal").sealOutline()

// sealOutlineAt is the harbor seal's cross-section at x, which the
// seal's skeleton is laid inside.
func sealOutlineAt(x float64) sealSection { return harborSealOutline.at(x) }

// at is the outline's cross-section at x: the cubic through the
// stations, with slopes from each station's neighbours, held at the
// ends.
func (outline sealOutline) at(x float64) sealSection {
	if x <= outline[0].x {
		return sealStationSection(outline[0], x)
	}
	if x >= outline[len(outline)-1].x {
		return sealStationSection(outline[len(outline)-1], x)
	}
	at := 0
	for at < len(outline)-2 && x > outline[at+1].x {
		at++
	}
	blend := func(pick func(sealStation) float64) float64 {
		low, high := outline[at], outline[at+1]
		return mammals.Hermite(low.x, high.x, pick(low), pick(high),
			outline.slope(at, pick), outline.slope(at+1, pick), x)
	}
	return sealSection{x: x,
		y:      blend(func(s sealStation) float64 { return s.centre }),
		width:  blend(func(s sealStation) float64 { return s.wide }),
		height: blend(func(s sealStation) float64 { return s.high })}
}

// sealStationSection is a station as a section at x.
func sealStationSection(station sealStation, x float64) sealSection {
	return sealSection{x: x, y: station.centre, width: station.wide,
		height: station.high}
}

// slope is the slope of one station quantity at station at, taken
// across its neighbours.
func (outline sealOutline) slope(at int,
	pick func(sealStation) float64,
) float64 {
	low, high := max(at-1, 0), min(at+1, len(outline)-1)
	return (pick(outline[high]) - pick(outline[low])) /
		(outline[high].x - outline[low].x)
}

// meanRadius is the outline's mean radius at x: half the sum of its
// half height and half width.
func (outline sealOutline) meanRadius(x float64) float64 {
	section := outline.at(x)
	return (section.height + section.width) / 2
}

// radial is the point of the outline's ellipse at x, turn radians round
// from the top toward the right side, and the way out of the skin
// there, square to the ellipse.
func (outline sealOutline) radial(x, turn float64) (point, out maths.Vec) {
	section := outline.at(x)
	sin, cos := math.Sincos(turn)
	point = maths.V(x, section.y+section.height*cos, section.width*sin)
	out = maths.V(0, cos/section.height, sin/section.width).Unit()
	return point, out
}

// The trunk's rings. Each has sealLoftSections points; the gap to the
// next is a share of the outline's mean radius there, so the muzzle and
// the tail are cut finer than the belly, between a least and a most.
// All picked: fine enough that no facet shows in a 1600-wide picture.
const (
	sealLoftSections  = 72
	sealLoftStepShare = .12
	sealLoftStepLeast = .003
	sealLoftStepMost  = .020
	// sealLoftPole is how far past the last ring each end's closing
	// point stands, as a share of that ring's mean radius.
	sealLoftPole = .5
)

// sealSurface is one closed surface at rest: its points and the
// triangles that join them, each wound so its normal points out.
type sealSurface struct {
	points []maths.Vec
	faces  [][3]int
}

// rings is the X of each ring of the trunk, from the tail's tip to the
// nose.
func (outline sealOutline) rings() []float64 {
	first, last := outline[0].x, outline[len(outline)-1].x
	stations := []float64{first}
	at := first
	for {
		step := sealLoftStepShare * outline.meanRadius(at)
		at += math.Min(math.Max(step, sealLoftStepLeast), sealLoftStepMost)
		if at >= last-sealLoftStepLeast {
			return append(stations, last)
		}
		stations = append(stations, at)
	}
}

// surface is the head, neck, trunk and tail as one surface lofted
// through the outline: a ring of points at each station, joined to the
// next, and closed by a point past each end.
func (outline sealOutline) surface() sealSurface {
	stations := outline.rings()
	surface := sealSurface{}
	for _, x := range stations {
		for column := range sealLoftSections {
			turn := 2 * math.Pi * float64(column) / sealLoftSections
			point, _ := outline.radial(x, turn)
			surface.points = append(surface.points, point)
		}
	}
	for row := 0; row+1 < len(stations); row++ {
		for column := range sealLoftSections {
			next := (column + 1) % sealLoftSections
			low, high := row*sealLoftSections, (row+1)*sealLoftSections
			surface.faces = append(surface.faces,
				[3]int{low + column, low + next, high + column},
				[3]int{low + next, high + next, high + column})
		}
	}
	surface.close(outline, 0, stations[0], -1)
	surface.close(outline, (len(stations)-1)*sealLoftSections,
		stations[len(stations)-1], 1)
	return surface
}

// close shuts the ring that starts at point first, at station x of the
// outline, with a fan to one point past it: way is one toward the nose
// and minus one toward the tail.
func (surface *sealSurface) close(outline sealOutline, first int,
	x, way float64,
) {
	pole := len(surface.points)
	surface.points = append(surface.points, maths.V(
		x+way*sealLoftPole*outline.meanRadius(x), outline.at(x).y, 0))
	for column := range sealLoftSections {
		next := (column + 1) % sealLoftSections
		face := [3]int{pole, first + column, first + next}
		if way < 0 {
			face = [3]int{pole, first + next, first + column}
		}
		surface.faces = append(surface.faces, face)
	}
}

// sealEdge is an edge of a surface as its two points' places, in the
// order a face runs it.
type sealEdge struct{ from, to int }

// open is how many of the surface's edges are not run exactly once each
// way: none for a closed surface whose faces all face the same side.
func (surface sealSurface) open() int {
	runs := map[sealEdge]int{}
	for _, face := range surface.faces {
		for corner := range face {
			runs[sealEdge{face[corner], face[(corner+1)%len(face)]}]++
		}
	}
	open := 0
	for edge, count := range runs {
		if count != 1 || runs[sealEdge{edge.to, edge.from}] != 1 {
			open++
		}
	}
	return open
}

// errSealCarrier is a flipper whose carrying bone is not in the
// skeleton.
var errSealCarrier = errors.New("seal: a flipper has no bone to carry it")

// sealForm is one species of seal as data for the lofted maker: its
// outline's stations in the shared rest space, the measured animal the
// stations' radii come from and its own mean radius at the axillary
// site, its length nose to tail tip over the rest space's, and its
// coat. A second seal is a second form.
type sealForm struct {
	profile  pinnipedProfile
	outline  sealOutline
	source   sealSource
	axillary float64
	scale    float64
	pelt     sealPelt
}

// harborSealForm is the harbor seal, Phoca vitulina, at the rest
// space's own size.
var harborSealForm = formFromProfile(defaultPinniped("harbor-seal"))

func formFromProfile(profile Profile) sealForm {
	return sealForm{profile: profile, outline: profile.sealOutline(), source: profile.Loft.Reference,
		axillary: profile.Loft.Axillary, scale: *profile.Length / sealLength,
		pelt: peltFromProfile(profile.Loft.Pelt)}
}

// Loft is a seal whose head, neck and trunk are one surface
// lofted through its form's outline, over the seal's skeleton, with a
// cushion of flesh over each flipper's digits, the skin held by the
// bones and posed by turning their joints. It is a still specimen in a
// pose.
type Loft struct {
	// form is the species.
	form sealForm
	// pose is the joints' turns, in the rig's words.
	pose SealRigPose
	// lean turns the whole animal nose up about its shoulders, radians.
	lean float64
	// rests is whether it lies on the ground, pressed where it bears;
	// the ground is then at height nought.
	rests bool
	// wet is the coat from dry at nought to soaked at one.
	wet float64
}

var (
	// sealBanana is a harbor seal hauled out with head and hind
	// flippers held up off the ground, the fore flippers out on the
	// ground beside the chest. Angles picked.
	sealBanana = Loft{form: harborSealForm, rests: true,
		wet: .45, lean: .15,
		pose: SealRigPose{Yaw: .06, Pitch: -.62, HeadPitch: .55,
			Fore: [2]float64{.35, .35}, Hind: [2]float64{.6, .6}}}
	// sealStroke is a harbor seal swimming: the hind body swept to one
	// side, the hind flippers turned the same way, the fore flippers a
	// little off the flanks. Angles picked.
	sealStroke = Loft{form: harborSealForm, wet: 1,
		pose: SealRigPose{Yaw: .38, Pitch: .04, HeadPitch: -.06,
			Fore: [2]float64{.2, .2}, Hind: [2]float64{.4, -.4}}}
)

// sealWhiskerReach pads the kind's extent past its skin for the
// whiskers, in metres at rest: the longest strand and a little.
const sealWhiskerReach = .1

// stance is the kind's pose over the skeleton: every bone's motion and
// the whole animal's lean about the first thoracic vertebra's head.
func (kind Loft) stance(made *sealBones) (sealStance, error) {
	frames, err := (&sealBound{made: made}).frames(kind.pose,
		sealFirstThoracic)
	if err != nil {
		return sealStance{}, err
	}
	shoulder := sealSpinePoint(sealFirstThoracic)
	return sealStance{frames: frames, rests: kind.rests,
		scale: kind.form.scale,
		lean:  skeleton.TurnAbout(shoulder, maths.ZAxis, kind.lean)}, nil
}

// made is the seal as one lofted body over its bones. It refuses a seed
// that was not made.
func (kind Loft) Build(number uint64) (Result, error) {
	made, err := newSealBones(kind.form.profile)
	if err != nil {
		return Result{}, err
	}
	flippers, err := sealHeldFlippers(made)
	if err != nil {
		return Result{}, err
	}
	held := append([]sealHeld{sealHeldTrunk(made, kind.form.outline)},
		flippers...)
	stance, err := kind.stance(made)
	if err != nil {
		return Result{}, err
	}
	stance = stance.onGround(held)
	return kind.dressed(made, held, stance, number)
}

// dressed is the held surfaces in the stance as the kind's parts: each
// a mesh in the coat, then the eyes and whiskers on the skull.
func (kind Loft) dressed(made *sealBones, held []sealHeld,
	stance sealStance, number uint64,
) (Result, error) {
	form := kind.form
	hide := sealHide{pelt: form.pelt, outline: form.outline, seed: number,
		wet: kind.wet}
	var parts []thing.Thing
	box := maths.Box{}
	for index, part := range held {
		skin, err := part.posed(stance)
		if err != nil {
			return Result{}, err
		}
		bounds := skin.Bounds()
		hide.limb = index > 0
		if index == 0 {
			box = bounds
		}
		box = maths.Box{Min: sealLowCorner(box.Min, bounds.Min),
			Max: sealHighCorner(box.Max, bounds.Max)}
		parts = append(parts, thing.Thing{Shape: skin,
			Material: hide.look(), Motion: thing.Still(maths.Local())})
	}
	skull, _ := made.bones.Find("skull")
	head := []skeleton.Influence{skeleton.NewInfluence(skull, 1)}
	parts = append(parts, form.outline.face(func(rest maths.Vec) maths.Vec {
		return stance.placed(rest, head)
	}, form.scale)...)
	reach := maths.V(1, 1, 1).Scale(sealWhiskerReach * form.scale)
	box = maths.Box{Min: box.Min.Sub(reach), Max: box.Max.Add(reach)}
	return Result{Parts: parts, Bounds: box}, nil
}

// posed is the surface in a stance as a mesh that keeps its rest points
// for the coat.
func (part sealHeld) posed(stance sealStance) (*mesh.Mesh, error) {
	points := make([]maths.Vec, len(part.surface.points))
	for index, rest := range part.surface.points {
		points[index] = stance.placed(rest, part.holds[index])
	}
	return mesh.NewWithRestPoints(points, part.surface.points,
		part.surface.faces, geometry.SmoothNormals(points, part.surface.faces),
		true)
}

// sealLowCorner is the least of two points' coordinates, each on its
// own.
func sealLowCorner(a, b maths.Vec) maths.Vec {
	return maths.V(math.Min(a.X, b.X), math.Min(a.Y, b.Y),
		math.Min(a.Z, b.Z))
}

// sealHighCorner is the most of two points' coordinates, each on its
// own.
func sealHighCorner(a, b maths.Vec) maths.Vec {
	return maths.V(math.Max(a.X, b.X), math.Max(a.Y, b.Y),
		math.Max(a.Z, b.Z))
}

// How the lofted skin is held by the bones. A point of the trunk follows
// the skull and the vertebrae near it along the body, each by a share
// that falls from one at the bone's middle to nothing a reach away; a
// flipper follows one bone whole. The numbers are picked: a reach wide
// enough that a bent back shows no crease.
const (
	// sealHoldReach is a vertebra's hold in its own lengths, each side.
	sealHoldReach = 1.5
	// sealSkullHold is how far ahead of the condyles the skull's hold is
	// whole, and sealSkullRun the length it is given for its reach.
	sealSkullHold, sealSkullRun = .03, .05
)

// sealAxis is the skull and the column's vertebrae as spans, the skull
// first and the tail's tip last.
func sealAxis(made *sealBones) []mammals.Span {
	var spans []mammals.Span
	at, _ := made.bones.Find("skull")
	skull := made.bones.Bone(at)
	spans = append(spans, mammals.Span{Bone: at,
		Middle: skull.Head().X + sealSkullHold, Run: sealSkullRun})
	neck := sealNamesWith(made.names(), "neck")
	for index := len(neck) - 1; index >= 0; index-- {
		spans = append(spans, made.span(neck[index]))
	}
	for _, name := range sealNamesWith(made.names(), "thoracic",
		"lumbar", "sacral", "caudal") {
		spans = append(spans, made.span(name))
	}
	return spans
}

// span is a named vertebra as a span of the axis.
func (made *sealBones) span(name string) mammals.Span {
	at, _ := made.bones.Find(name)
	bone := made.bones.Bone(at)
	head, tail := bone.Head().X, bone.Tail().X
	return mammals.Span{Bone: at, Middle: (head + tail) / 2,
		Run: math.Abs(head - tail)}
}

// sealHeld is a surface with the bones that carry each of its points.
type sealHeld struct {
	surface sealSurface
	holds   [][]skeleton.Influence
}

// sealHeldTrunk is an outline's lofted trunk held along the axis.
func sealHeldTrunk(made *sealBones, outline sealOutline) sealHeld {
	spans := sealAxis(made)
	surface := outline.surface()
	holds := make([][]skeleton.Influence, len(surface.points))
	for index, point := range surface.points {
		holds[index] = mammals.AxialHold(spans, point.X, sealHoldReach, nil)
	}
	return sealHeld{surface: surface, holds: holds}
}

// heldPaddle is a flipper's cushion over the digits whose bones' names
// bones gives, carried whole by the bone named carrier.
func (made *sealBones) heldPaddle(bones func(digit int) []string,
	shape sealPaddleShape, carrier string,
) (sealHeld, error) {
	lines, err := made.digitLines(bones, shape)
	if err != nil {
		return sealHeld{}, err
	}
	at, found := made.bones.Find(carrier)
	if !found {
		return sealHeld{}, errSealCarrier
	}
	surface := sealPaddleSurface(lines, shape)
	hold := []skeleton.Influence{skeleton.NewInfluence(at, 1)}
	holds := make([][]skeleton.Influence, len(surface.points))
	for index := range holds {
		holds[index] = hold
	}
	return sealHeld{surface: surface, holds: holds}, nil
}

// sealHeldFlippers is the four flippers, each carried by its limb: a
// fore flipper by its middle metacarpal, a hind one by its ankle bone.
func sealHeldFlippers(made *sealBones) ([]sealHeld, error) {
	var flippers []sealHeld
	for _, side := range [...]float64{1, -1} {
		fore := "fore-" + sealSideName(side) + "-"
		hind := "hind-" + sealSideName(side) + "-"
		for _, limb := range [...]struct {
			bones   func(int) []string
			shape   sealPaddleShape
			carrier string
		}{
			{func(digit int) []string {
				return sealForeDigitBones(fore, digit)
			}, sealForePaddle, fore + "metacarpal3"},
			{func(digit int) []string {
				return sealHindDigitBones(hind, digit)
			}, sealHindPaddle, hind + "astragalus"},
		} {
			held, err := made.heldPaddle(limb.bones, limb.shape,
				limb.carrier)
			if err != nil {
				return nil, err
			}
			flippers = append(flippers, held)
		}
	}
	return flippers, nil
}

// How a seal lying on the ground settles. The ground is taken a sink
// above the lowest point of the posed animal; whatever lies below a
// band over it is pressed up toward it, most where it is lowest, so the
// belly flattens where it bears, and spreads sideways as it is pressed.
// All picked.
const (
	sealSlumpSink   = .045 // metres the lowest point lies under the ground
	sealSlumpBand   = .11  // metres over the ground that are pressed
	sealSlumpSpread = 3.   // sideways spread per metre a point is lifted
)

// sealStance is an animal's pose as the skin needs it: each bone's
// motion, the turn of the whole animal, for one lying down the height
// of the ground it settles on, and the animal's size over the rest
// space's, by which it is scaled last.
type sealStance struct {
	frames []skeleton.Frame
	lean   skeleton.Frame
	rests  bool
	ground float64
	scale  float64
}

// carried is a rest point moved by its bones and the whole animal's
// turn, before it settles.
func (stance sealStance) carried(rest maths.Vec,
	hold []skeleton.Influence,
) maths.Vec {
	return stance.lean.Apply(skeleton.Carry(rest, stance.frames, hold))
}

// sealPressed is the height over the ground a point at height over it
// is pressed to: itself above the band, and below it a height that
// falls ever more slowly toward the ground.
func sealPressed(height float64) float64 {
	if height >= sealSlumpBand {
		return height
	}
	under := (sealSlumpBand - height) / sealSlumpBand
	return sealSlumpBand * (1 - under/(1+under))
}

// settled is a carried point as it lies: unchanged for an animal that
// does not rest; else pressed onto the ground, which becomes height
// nought.
func (stance sealStance) settled(point maths.Vec) maths.Vec {
	if !stance.rests {
		return point
	}
	height := point.Y - stance.ground
	pressed := sealPressed(height)
	return maths.V(point.X, pressed-sealPressed(-sealSlumpSink),
		point.Z*(1+sealSlumpSpread*(pressed-height)))
}

// placed is a rest point where the skin's point held so ends up, at the
// animal's own size.
func (stance sealStance) placed(rest maths.Vec,
	hold []skeleton.Influence,
) maths.Vec {
	return stance.settled(stance.carried(rest, hold)).Scale(stance.scale)
}

// onGround is the stance with its ground set from the lowest carried
// point of the surfaces.
func (stance sealStance) onGround(parts []sealHeld) sealStance {
	lowest := math.Inf(1)
	for _, part := range parts {
		for index, rest := range part.surface.points {
			lowest = math.Min(lowest,
				stance.carried(rest, part.holds[index]).Y)
		}
	}
	stance.ground = lowest + sealSlumpSink
	return stance
}

// A flipper's skin is a flat cushion over its five digits: two sheets,
// one each side of the digits' plane, that meet all round their rim. The
// counts are picked: enough rows and columns that the rim reads round.
const (
	sealPaddleRows    = 20
	sealPaddleColumns = 17
	// sealPaddleRootCap and sealPaddleTipCap are the shares of the
	// length over which the cushion thins to nothing at each end.
	sealPaddleRootCap = .10
	sealPaddleTipCap  = .30
)

// sealPaddleShape is a flipper's flesh round its digits, in metres but
// for the margin: how far the skin runs back from the knuckles into the
// body and on past the digits' tips, how far the web stands past the
// outer digits as a share of the gap between digits, and the cushion's
// half thickness at its root and at its tip. All picked.
type sealPaddleShape struct {
	root, tip, margin, thick, thin float64
}

var (
	// sealForePaddle is the fore flipper's flesh: its root covers the
	// wrist and the forearm's end, inside the trunk's skin.
	sealForePaddle = sealPaddleShape{root: .085, tip: .014, margin: .8,
		thick: .022, thin: .005}
	// sealHindPaddle is the hind flipper's: its root covers the tarsals
	// and the ankle.
	sealHindPaddle = sealPaddleShape{root: .090, tip: .014, margin: .5,
		thick: .022, thin: .005}
)

// sealForeDigitBones is the names of one fore digit's bones from its
// metacarpal out, digits counted from one at the thumb.
func sealForeDigitBones(prefix string, digit int) []string {
	names := []string{fmt.Sprintf("%smetacarpal%d", prefix, digit)}
	for bone := 1; bone < len(sealForeDigits[digit-1]); bone++ {
		names = append(names,
			fmt.Sprintf("%sphalanx%d-%d", prefix, digit, bone))
	}
	return names
}

// sealHindDigitBones is the names of one hind digit's bones from its
// metatarsal out, digits counted from one.
func sealHindDigitBones(prefix string, digit int) []string {
	var names []string
	for bone := range sealHindDigits[digit-1] {
		names = append(names,
			fmt.Sprintf("%sdigit%d-%d", prefix, digit, bone+1))
	}
	return names
}

// digitLine is one digit's middle line at rest through the named bones'
// ends, begun back from the first bone's head and run tip metres past
// the last one's tail.
func (made *sealBones) digitLine(names []string, back maths.Vec,
	tip float64,
) ([]maths.Vec, error) {
	var line []maths.Vec
	for _, name := range names {
		head, tail, found := made.ends(name)
		if !found {
			return nil, fmt.Errorf("harbor seal: no bone %s", name)
		}
		if line == nil {
			line = append(line, head.Add(back), head)
		}
		line = append(line, tail)
	}
	last, before := line[len(line)-1], line[len(line)-2]
	return append(line, last.Add(last.Sub(before).Unit().Scale(tip))), nil
}

// digitLines is a flipper's five digits' lines, each from the names
// bones gives for it, all begun the shape's root back along the middle
// digit's way.
func (made *sealBones) digitLines(bones func(digit int) []string,
	shape sealPaddleShape,
) ([][]maths.Vec, error) {
	const digits = len(sealForeDigits)
	middle := bones((digits + 1) / 2)[0]
	head, tail, found := made.ends(middle)
	if !found {
		return nil, fmt.Errorf("harbor seal: no bone %s", middle)
	}
	back := head.Sub(tail).Unit().Scale(shape.root)
	lines := make([][]maths.Vec, digits)
	for digit := range lines {
		line, err := made.digitLine(bones(digit+1), back, shape.tip)
		if err != nil {
			return nil, err
		}
		lines[digit] = line
	}
	return lines, nil
}

// sealAlong is the point a share of the way along a line, by length.
func sealAlong(line []maths.Vec, share float64) maths.Vec {
	whole := 0.
	for at := 1; at < len(line); at++ {
		whole += line[at].Sub(line[at-1]).Len()
	}
	left := whole * share
	for at := 1; at < len(line)-1; at++ {
		run := line[at].Sub(line[at-1]).Len()
		if left <= run {
			return line[at-1].Add(line[at].Sub(line[at-1]).Scale(left / run))
		}
		left -= run
	}
	last, before := line[len(line)-1], line[len(line)-2]
	return before.Add(last.Sub(before).Unit().Scale(left))
}

// sealPaddleMiddle is the point of the digits' sheet across digits from
// the first, counted in digits and running on past the outer ones, a
// share of the way along.
func sealPaddleMiddle(lines [][]maths.Vec, across, share float64,
) maths.Vec {
	first := min(max(int(math.Floor(across)), 0), len(lines)-2)
	low := sealAlong(lines[first], share)
	high := sealAlong(lines[first+1], share)
	return low.Add(high.Sub(low).Scale(across - float64(first)))
}

// thickness is the cushion's half thickness across digits from the
// first and a share of the way along: round across the flipper, thinning
// to its tip, and closing to nothing at the rim.
func (shape sealPaddleShape) thickness(across, last, share float64,
) float64 {
	offset := (across - last/2) / (last/2 + shape.margin)
	round := math.Sqrt(math.Max(0, 1-offset*offset))
	end := 0.
	if share < sealPaddleRootCap {
		end = 1 - share/sealPaddleRootCap
	}
	if share > 1-sealPaddleTipCap {
		end = (share - (1 - sealPaddleTipCap)) / sealPaddleTipCap
	}
	round *= math.Sqrt(math.Max(0, 1-end*end))
	return (shape.thick + (shape.thin-shape.thick)*share) * round
}

// sealPaddleSurface is a flipper's skin over its digits' lines: the
// upper sheet's points row by row from the root, then the lower sheet's.
func sealPaddleSurface(lines [][]maths.Vec, shape sealPaddleShape,
) sealSurface {
	const rows, columns = sealPaddleRows, sealPaddleColumns
	last := float64(len(lines) - 1)
	middle := lines[len(lines)/2]
	along := middle[len(middle)-1].Sub(middle[0]).Unit()
	span := sealAlong(lines[len(lines)-1], .5).Sub(sealAlong(lines[0], .5))
	normal := along.Cross(span).Unit()
	surface := sealSurface{points: make([]maths.Vec, 2*rows*columns)}
	for row := range rows {
		share := float64(row) / (rows - 1)
		for column := range columns {
			across := -shape.margin +
				(last+2*shape.margin)*float64(column)/(columns-1)
			lift := normal.Scale(shape.thickness(across, last, share))
			at := sealPaddleMiddle(lines, across, share)
			surface.points[row*columns+column] = at.Add(lift)
			surface.points[(rows+row)*columns+column] = at.Sub(lift)
		}
	}
	surface.faces = sealPaddleFaces()
	return surface
}

// sealPaddleRim is whether a grid point lies on the cushion's rim, where
// the two sheets share their points.
func sealPaddleRim(row, column int) bool {
	return row == 0 || row == sealPaddleRows-1 || column == 0 ||
		column == sealPaddleColumns-1
}

// sealPaddlePoint is the place of a grid point's point: in the upper
// sheet, or in the lower, which takes the upper's along the rim.
func sealPaddlePoint(row, column int, lower bool) int {
	at := row*sealPaddleColumns + column
	if lower && !sealPaddleRim(row, column) {
		at += sealPaddleRows * sealPaddleColumns
	}
	return at
}

// sealPaddleFaces is the cushion's triangles: each grid cell cut in two
// on the upper sheet and, turned over, on the lower. A triangle with all
// three corners on the rim is in both sheets at once and is left out.
func sealPaddleFaces() [][3]int {
	var faces [][3]int
	for row := 0; row+1 < sealPaddleRows; row++ {
		for column := 0; column+1 < sealPaddleColumns; column++ {
			cell := [4][2]int{{row, column}, {row + 1, column},
				{row + 1, column + 1}, {row, column + 1}}
			for _, half := range [2][3]int{{0, 1, 3}, {1, 2, 3}} {
				var upper, lower [3]int
				rim := 0
				for at, corner := range half {
					place := cell[corner]
					if sealPaddleRim(place[0], place[1]) {
						rim++
					}
					upper[at] = sealPaddlePoint(place[0], place[1], false)
					lower[len(half)-1-at] = sealPaddlePoint(place[0],
						place[1], true)
				}
				if rim < len(half) {
					faces = append(faces, upper, lower)
				}
			}
		}
	}
	return faces
}

// The Hawaiian monk seal, Neomonachus schauinslandi, as a second form
// for the lofted maker: its own stations, its own length and a plain
// coat. It is drawn over the harbor seal's skeleton, which its outline
// holds.

// monkSealOutline is the monk seal's outline in the rest space. The
// eight named sites are the Weddell seal's, a seal of the monk seal's
// own subfamily, each with its mean radius, the neck and the ears a
// little over; the head is flatter and broader than the harbor seal's.
// Widths over heights, centre heights and the muzzle and tail stations
// are picked.
var monkSealOutline = defaultPinniped("hawaiian-monk-seal").sealOutline()

// monkSealPelt and form come from the checked monk-seal profile.
var monkSealPelt = peltFromProfile(defaultPinniped("hawaiian-monk-seal").Loft.Pelt)
var monkSealForm = formFromProfile(defaultPinniped("hawaiian-monk-seal"))

// sealBasking is a monk seal asleep on a beach: nearly flat, the head
// a little raised, the fore flippers out on the sand. Angles picked.
var sealBasking = Loft{form: monkSealForm, rests: true,
	wet: .2, lean: .05,
	pose: SealRigPose{Yaw: -.08, Pitch: -.22, HeadPitch: .22,
		Fore: [2]float64{.35, .35}, Hind: [2]float64{.6, .6}}}

// checkLoft limits a profile to small changes around the scaffold it uses.
// The fixed skeleton and flippers are not a general rig for arbitrary seals.
// It also verifies that the posed trunk remains closed and carried.
func (row Loft) checkLoft(reference sealOutline) error {
	outline := row.form.outline
	if len(outline) != len(reference) {
		return fmt.Errorf("manufacture: seal loft must retain the scaffold stations")
	}
	for i, station := range outline {
		base := reference[i]
		if station.x != base.x || math.Abs(station.high-base.high) > .1*base.high ||
			math.Abs(station.wide-base.wide) > .1*base.wide || math.Abs(station.centre-base.centre) > .1*base.high {
			return fmt.Errorf("manufacture: seal loft station %d exceeds the shared rig's range", i)
		}
	}
	bones, err := newSealBones(row.form.profile)
	if err != nil {
		return err
	}
	trunk := sealHeldTrunk(bones, outline)
	if trunk.surface.open() != 0 {
		return fmt.Errorf("manufacture: seal loft is not closed")
	}
	for _, influences := range trunk.holds {
		weight := 0.0
		for _, influence := range influences {
			weight += influence.Weight()
		}
		if !(weight > 0) || !finite(weight) {
			return fmt.Errorf("manufacture: seal loft has an unbound point")
		}
	}
	flippers, err := sealHeldFlippers(bones)
	if err != nil {
		return err
	}
	stance, err := row.stance(bones)
	if err != nil {
		return err
	}
	stance = stance.onGround(append([]sealHeld{trunk}, flippers...))
	if _, err = trunk.posed(stance); err != nil {
		return fmt.Errorf("manufacture: invalid posed seal loft: %w", err)
	}
	return nil
}

// NewLoft selects and validates the profile's shared posed scaffold.
func NewLoft(profile Profile, pose string) (Loft, error) {
	var row Loft
	var reference sealOutline
	if profile.Form == "harbor-loft" {
		if profile.AnkleForward != 0 {
			return Loft{}, fmt.Errorf("manufacture: harbor-loft form has no forward ankle turn")
		}
		switch pose {
		case "", "hauled":
			row = sealBanana
		case "swimming":
			row = sealStroke
		default:
			return Loft{}, fmt.Errorf("manufacture: unsupported %s pose %q", profile.Kind, pose)
		}
		reference = harborSealOutline
	} else if profile.Form == "monk-loft" {
		if profile.AnkleForward != 0 {
			return Loft{}, fmt.Errorf("manufacture: monk-loft form has no forward ankle turn")
		}
		if pose != "" && pose != "hauled" {
			return Loft{}, fmt.Errorf("manufacture: unsupported %s pose %q", profile.Kind, pose)
		}
		row = sealBasking
		reference = monkSealOutline
	} else {
		return Loft{}, fmt.Errorf("manufacture: unsupported pinniped form %q", profile.Form)
	}
	if err := validateLoftProfile(profile.Loft); err != nil {
		return Loft{}, fmt.Errorf("manufacture: invalid %s loft profile: %w", profile.Kind, err)
	}
	row.form = formFromProfile(profile.snapshot())
	if err := row.checkLoft(reference); err != nil {
		return Loft{}, err
	}
	return row, nil
}
