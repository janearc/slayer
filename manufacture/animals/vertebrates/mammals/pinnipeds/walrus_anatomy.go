package pinnipeds

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// walrusDigitCount is the number of digits on every flipper.
const walrusDigitCount = 5

// walrusRest is the walrus skeleton at rest, laid out once. The hide is
// built over it, and the face's landmarks are placed in its skull's frame.
var walrusRest = walrusSkeleton()

// walrusFrame is the skull's frame at rest.
var walrusFrame = walrusSkullOf(walrusRest)

// walrusRig keeps all anatomy derived from one skeleton together.
type walrusRig struct {
	body          *walrusBones
	frame         walrusSkullFrame
	rays          [][walrusDigitCount][]maths.Vec
	restSurfaces  [2]walrusRestSurface
	outlineOnce   sync.Once
	cachedOutline *walrusOutline
	outlineFault  error
}

func newWalrusRig(profile Profile) (*walrusRig, error) {
	body := newWalrusBones(profile)
	frame := walrusSkullOf(body)
	if body.fault != nil {
		return nil, body.fault
	}
	rig := &walrusRig{body: body, frame: frame, rays: walrusFlipperRays(body)}
	if err := rig.checkWalrusOutline(); err != nil {
		return nil, err
	}
	return rig, nil
}

func defaultWalrusRig() *walrusRig {
	return &walrusRig{body: walrusRest, frame: walrusFrame, rays: walrusRayLines}
}

// errWalrusForm is a skeleton that lacks a bone the hide is built on, or
// has it in a form the hide cannot read.
var errWalrusForm = errors.New("walrus: the skeleton lacks a bone the hide is built on")

// walrusSkullOf is the frame of the skull of the skeleton given. A
// skeleton with no skull is given a fault, which refuses every pose.
func walrusSkullOf(body *walrusBones) walrusSkullFrame {
	skull, ok := body.forms["skull"].(walrusCranium)
	if !ok && body.fault == nil {
		body.fault = fmt.Errorf("%w: the skull", errWalrusForm)
	}
	return skull.frame
}

// walrusMasses is the flesh of the four flippers on their bones and the
// lids of the eyes on the outline given. The body, neck and head are that
// one outline, from the stations in walrus_outline.go, and these are
// joined to it.
func (rig *walrusRig) walrusMasses(outline *walrusOutline) ([]walrusMass, error) {
	lids, err := rig.walrusLids(outline)
	if err != nil {
		return nil, err
	}
	return append(walrusLimbMasses(rig.body), lids...), nil
}

// walrusRayLines is the middle line of every ray of every flipper at
// rest, from the base of its palm or sole bone to the web's rim: four
// flippers, fore then hind on the left, then the right.
var walrusRayLines = walrusFlipperRays(walrusRest)

// walrusFlipperRays is the rays of the four flippers of the skeleton.
func walrusFlipperRays(body *walrusBones) [][walrusDigitCount][]maths.Vec {
	var flippers [][walrusDigitCount][]maths.Vec
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		flippers = append(flippers,
			walrusRayLinesOf(body, "metacarpal", "fore", name, walrusForeDigits),
			walrusRayLinesOf(body, "metatarsal", "hind", name, walrusHindDigits))
	}
	return flippers
}

// walrusRayLinesOf is the five rays of one flipper as lines of points.
func walrusRayLinesOf(body *walrusBones, palm, kind, side string,
	digits [walrusDigitCount][]float64) [walrusDigitCount][]maths.Vec {
	var rays [walrusDigitCount][]maths.Vec
	for ray := 1; ray <= walrusDigitCount; ray++ {
		for _, name := range walrusRayNames(palm, kind, side, ray, digits) {
			from, to := walrusBoneLine(body, name)
			if len(rays[ray-1]) == 0 {
				rays[ray-1] = append(rays[ray-1], from)
			}
			rays[ray-1] = append(rays[ray-1], to)
		}
		line := rays[ray-1]
		end, before := line[len(line)-1], line[len(line)-2]
		rays[ray-1] = append(line, end.Add(end.Sub(before).Unit().Scale(walrusWebRim)))
	}
	return rays
}

// walrusLineDistance is how far p lies from a line of points.
func walrusLineDistance(p maths.Vec, line []maths.Vec) float64 {
	nearest := math.Inf(1)
	for i := 1; i < len(line); i++ {
		nearest = math.Min(nearest, walrusSegmentDistance(p, line[i-1], line[i]))
	}
	return nearest
}

// walrusLimbShare is how much of a flipper's own skin there is at p: one
// on and between its rays, nought on the body away from them. The hide's
// colour uses it to darken the flippers.
func (rig *walrusRig) walrusLimbShare(p maths.Vec) float64 {
	const near, far = .05, .11
	nearest := math.Inf(1)
	for _, flipper := range rig.rays {
		for _, ray := range flipper {
			nearest = math.Min(nearest, walrusLineDistance(p, ray))
		}
	}
	return 1 - maths.Smooth(near, far, nearest)
}

// The walrus skeleton is laid out at rest lying on land, in the hide's own
// space: x forward, y up, z to the right, the ground at y of minus .41.
// Every count and length says where it is from; "chosen" means no source
// was read for it and the number was picked to fit the others.
// Lengths along the backbone, in metres. Chosen, so that the hide over
// skull and column comes to about the 315.2 cm asymptotic standard length
// of bulls (Garlich-Miller and Stewart 1998, as quoted in the COSEWIC
// Atlantic walrus report), inside the 2.2 to 3.5 m of Fay 1985.
const (
	walrusCervicalLength = .075
	walrusThoracicLength = .080
	walrusLumbarLength   = .090
	walrusSacralLength   = .065
	// walrusCaudalLength is the first tail vertebra; the rest shrink.
	walrusCaudalLength = .040
	// walrusSkullLength is condyle to the front of the premaxilla.
	walrusSkullLength = .40
	// walrusSkullTilt is how far the skull's axis drops toward the snout,
	// in radians, in the rest pose.
	walrusSkullTilt = .25
)

// Fore limb lengths, in metres. The four are a measured Pacific female,
// museum specimen UWBM 35479, on a virtual zooarchaeology page, given
// there as scapula 255.882, humerus 253.959, ulna 265.384 and radius
// 210.408, read as millimetres; walrusBullScale makes them a bull's.
const (
	walrusScapulaLength = .255882 * walrusBullScale
	walrusHumerusLength = .253959 * walrusBullScale
	walrusUlnaLength    = .265384 * walrusBullScale
	walrusRadiusLength  = .210408 * walrusBullScale
	// walrusBullScale is a bull's limb bone over that female's. Chosen.
	walrusBullScale = 1.15
	// walrusFemurLength and walrusTibiaLength are chosen: a short, broad
	// thigh bone and a long shank, as a pinniped has.
	walrusFemurLength = .20
	walrusTibiaLength = .38
)

// walrusRegion is a run of vertebrae: its name, how many, the length of
// each, the centrum's radius, the neural spine's height, the transverse
// process's reach, and how much the run curls down at each joint.
type walrusRegion struct {
	name                 string
	count                int
	length               float64
	radius, spine, reach float64
	curl                 float64
}

// walrusThoracolumbar is the joint between the last thoracic vertebra and
// the first lumbar: the root of the skeleton.
var walrusThoracolumbar = maths.V(-.29, .14, 0)

// walrusBones is the walrus's skeleton: the shared tree of bones, the form
// of each bone, and the first fault met in laying them out.
type walrusBones struct {
	profile Profile
	tree    *skeleton.Skeleton
	forms   map[string]skeleton.BoneForm
	fault   error
}

// add hangs a bone from the bone named parent, or makes it the root when
// parent is empty: its joint, its far end, its half thickness and its
// form. It gives back the bone's name; a fault is kept, not lost.
func (body *walrusBones) add(name, parent string, joint, tail maths.Vec,
	thick float64, form skeleton.BoneForm) string {
	if err := body.tree.Add(name, parent, joint, tail, thick, thick); err != nil {
		if body.fault == nil {
			body.fault = err
		}
		return name
	}
	body.forms[name] = form
	return name
}

// bone is the bone of that name, and its place in the tree.
func (body *walrusBones) bone(name string) (skeleton.Bone, int) {
	at, _ := body.tree.Find(name)
	return body.tree.Bone(at), at
}

// rod is the two ends of the straight bone of that name.
func (body *walrusBones) rod(name string) (maths.Vec, maths.Vec) {
	piece, _ := body.forms[name].(skeleton.Rod)
	return piece.Ends()
}

// walrusSkeleton lays out the whole walrus at rest.
func walrusSkeleton() *walrusBones {
	return newWalrusBones(defaultPinniped("walrus"))
}

func newWalrusBones(profile Profile) *walrusBones {
	body := &walrusBones{profile: profile, tree: skeleton.New(),
		forms: map[string]skeleton.BoneForm{}}
	neck := walrusForeColumn(body)
	walrusSkull(body, neck)
	sacrum := walrusHindColumn(body)
	walrusRibcage(body)
	// The authored limb positions are relative to the default spine. Move
	// the whole girdle with its parent when a profile changes that spine.
	var foreShift, hindShift maths.Vec
	standard := defaultPinniped("walrus")
	if profile.Thoracic != standard.Thoracic || profile.Lumbar != standard.Lumbar {
		reference := &walrusBones{profile: standard, tree: skeleton.New(), forms: map[string]skeleton.BoneForm{}}
		walrusForeColumn(reference)
		walrusHindColumn(reference)
		fore, _ := body.bone("thoracic 4")
		oldFore, _ := reference.bone("thoracic 4")
		hind, _ := body.bone("sacral 1")
		oldHind, _ := reference.bone("sacral 1")
		foreShift = fore.Head().Sub(oldFore.Head())
		hindShift = hind.Head().Sub(oldHind.Head())
	}
	for _, side := range []float64{-1, 1} {
		walrusForeLimb(body, side, foreShift)
		walrusHindLimb(body, side, sacrum, hindShift)
	}
	return body
}

// walrusForeColumn lays the thoracic vertebrae from the root forward, then
// the neck, rising to the skull, and gives back the name of the atlas.
func walrusForeColumn(body *walrusBones) string {
	thorax := walrusRegion{name: "thoracic", count: body.profile.Thoracic,
		length: walrusThoracicLength, radius: .033, spine: .07, reach: .03,
		curl: .0015}
	neck := walrusRegion{name: "cervical", count: body.profile.Cervical,
		length: walrusCervicalLength, radius: .030, spine: .03, reach: .05,
		curl: -.012}
	heading := maths.V(1, .02, 0).Unit()
	last, at := walrusRun(body, "", walrusThoracolumbar, heading, thorax, true)
	atlas, _ := walrusRun(body, last, at, maths.V(.9255, .3786, 0).Unit(),
		neck, true)
	return atlas
}

// walrusHindColumn lays the lumbar, sacral and tail vertebrae from the
// root back, and gives back the name of the first sacral vertebra. The
// headings are chosen so the column falls with the back of the outline
// over it, from the loins down over the rump to the tail.
func walrusHindColumn(body *walrusBones) string {
	loins := walrusRegion{name: "lumbar", count: body.profile.Lumbar,
		length: walrusLumbarLength, radius: .038, spine: .045, reach: .07,
		curl: .006}
	sacrum := walrusRegion{name: "sacral", count: body.profile.Sacral,
		length: walrusSacralLength, radius: .035, spine: .03, reach: .06,
		curl: .01}
	tail := walrusRegion{name: "caudal", count: body.profile.Caudal,
		length: walrusCaudalLength, radius: .024, spine: .012, reach: .025,
		curl: .06}
	last, at := walrusRun(body, fmt.Sprintf("thoracic %d", body.profile.Thoracic), walrusThoracolumbar,
		maths.V(-1, -.14, 0).Unit(), loins, false)
	sacral, end := walrusRun(body, last, at, maths.V(-1, -.45, 0).Unit(),
		sacrum, false)
	walrusRun(body, sacral, end, maths.V(-1, -.7, 0).Unit(), tail, false)
	return "sacral 1"
}

// walrusRun lays one run of vertebrae from at along heading, each hung
// from the one before, curling heading down by the run's curl at each
// joint and tapering toward the run's end when the run is the tail.
// Numbering runs away from the root, except that a run counted toward the
// root (reversed) numbers its last bone first. It gives back the name of
// the run's last bone and where the run ends.
func walrusRun(body *walrusBones, after string, at, heading maths.Vec,
	run walrusRegion, reversed bool) (string, maths.Vec) {
	parent := after
	down := -run.curl
	if heading.X < 0 {
		down = run.curl
	}
	for i := 0; i < run.count; i++ {
		number := i + 1
		if reversed {
			number = run.count - i
		}
		taper := walrusTailTaper(run, i)
		tail := at.Add(heading.Scale(run.length * taper))
		name := fmt.Sprintf("%s %d", run.name, number)
		radius := run.radius * taper
		parent = body.add(name, parent, at, tail, radius,
			skeleton.NewCentrum(at, tail, radius, run.spine*taper,
				run.reach*taper))
		at = tail
		heading = maths.Rotation(maths.V(0, 0, 1), down).Out(heading)
	}
	return parent, at
}

// walrusTailTaper is how much the i-th vertebra of the run is shrunk: not
// at all, except down the tail, which ends at a third of its first size.
func walrusTailTaper(run walrusRegion, i int) float64 {
	const end = .35
	if run.name != "caudal" {
		return 1
	}
	return 1 - (1-end)*float64(i)/float64(run.count-1)
}

// walrusRibcage hangs a rib pair from every thoracic vertebra and the
// breastbone under the first ribs.
func walrusRibcage(body *walrusBones) {
	const ribThickness = .015
	_ = Ribs(body.profile.Thoracic, func(number int, side float64) error {
		owner := fmt.Sprintf("thoracic %d", number)
		vertebra, _ := body.bone(owner)
		at := vertebra.Head().Add(vertebra.Tail()).Scale(.5)
		name := fmt.Sprintf("rib %d %s", number, walrusSideName(side))
		rib := walrusRib(number, body.profile.SternalRibs, at, side)
		body.add(name, owner, at, rib.At(1), ribThickness, rib)
		return body.fault
	})
	walrusSternum(body)
}

// walrusRib is the rib of the given number on one side, from its head at
// the vertebra round the barrel of the chest: the sternal ribs to the
// breastbone, the floating ribs short of it. The chest is narrow at the
// first rib and widest from the sixth. Sizes chosen.
func walrusRib(number, sternalRibs int, at maths.Vec, side float64) skeleton.Arc {
	const (
		centre = -.05
		breast = -.30
		sweep  = .12
	)
	half := walrusRibHalf(number)
	head := maths.V(at.X, at.Y, side*.045)
	end := maths.V(at.X-sweep, breast, side*.03)
	if number > sternalRibs {
		beyond := float64(number - sternalRibs)
		end = maths.V(at.X-sweep-.02, -.20+.025*beyond, side*half*.85)
	}
	out := side * (half - .125*(.045+math.Abs(end.Z))) / .75
	return skeleton.NewArc([4]maths.Vec{head,
		maths.V(at.X-.02, at.Y, out), maths.V(at.X-sweep*.7, centre-.15, out),
		end}, .010, .021, maths.V(1, 0, 0))
}

// walrusRibHalf is the half width of the chest at the rib of the given
// number: narrow at the first rib, widest from the sixth. Chosen.
func walrusRibHalf(number int) float64 {
	const halfWidth, firstRibShare = .30, .55
	grow := firstRibShare +
		(1-firstRibShare)*maths.Smooth(0, 5, float64(number-1))
	return halfWidth * grow
}

// walrusSternum is the breastbone: a row of sternebrae along the floor of
// the chest, from under the first rib to under the last sternal rib, hung
// from the fifth thoracic vertebra.
func walrusSternum(body *walrusBones) {
	const breast, radius = -.31, .020
	first, _ := body.bone("thoracic 1")
	last, _ := body.bone(fmt.Sprintf("thoracic %d", body.profile.SternalRibs))
	front, back := first.Head().X-.02, last.Head().X-.14
	step := (front - back) / float64(body.profile.Sternebrae)
	for i := 0; i < body.profile.Sternebrae; i++ {
		from := maths.V(front-float64(i)*step, breast, 0)
		to := from.Sub(maths.V(step*.85, 0, 0))
		body.add(fmt.Sprintf("sternebra %d", i+1), "thoracic 5", from, to,
			radius, skeleton.NewRod(from, to, radius, .7))
	}
}

// walrusSideName is the word for one side.
func walrusSideName(side float64) string {
	if side < 0 {
		return "left"
	}
	return "right"
}

// WalrusSkeleton is the walrus's skeleton alone, at rest lying on
// land, every bone a closed surface: the study of the bones the hide is
// built over.
func WalrusSkeleton() ([]thing.Thing, error) {
	return walrusSkeletonThings(walrusSkeleton())
}

func walrusSkeletonThings(body *walrusBones) ([]thing.Thing, error) {
	if body.fault != nil {
		return nil, body.fault
	}
	points, faces := body.tree.Draw(body.tree.Frames(skeleton.Pose{}), body.forms)
	bones, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		return nil, err
	}
	material := &thing.Material{Name: "walrus bone",
		Colour: maths.RGB(.72, .67, .57), Gloss: .08, Satin: .5}
	return []thing.Thing{{Shape: bones, Material: material,
		Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))}}, nil
}

// The tusks are the upper canines, rooted deep in the maxilla. Their
// length and the share of it outside the bone are from the Digimorph
// walrus page: "A single male tusk can achieve a length of one meter"
// (Berta and Sumich 1999) and "roughly 80% of the tusk is exposed above
// the gumline" (Nowak 1991).
const (
	// walrusTuskLength is the whole tusk, root to worn tip, in metres.
	// Chosen inside the one metre the source gives as the most.
	walrusTuskLength = .65
	// walrusTuskExposed is the share of the tusk outside the jaw.
	walrusTuskExposed = .80
	// walrusTuskStart is the angle, in radians from straight down toward
	// the snout in the skull's own frame, at which the tusk leaves its
	// root; walrusTuskCurve is how fast that angle falls along the tusk,
	// in radians a metre, curving the tusk down and back. Both chosen.
	walrusTuskStart = .80
	walrusTuskCurve = -1.15
)

// walrusSkullFrame is the skull's own frame at rest: the occipital condyle,
// the axis toward the snout, and down, square to the axis.
type walrusSkullFrame struct {
	condyle, axis, down maths.Vec
}

// walrusSkullFrameAt is the skull's frame for a condyle at the given point.
func walrusSkullFrameAt(condyle maths.Vec) walrusSkullFrame {
	axis := maths.V(math.Cos(walrusSkullTilt), -math.Sin(walrusSkullTilt), 0)
	return walrusSkullFrame{condyle: condyle, axis: axis,
		down: maths.V(axis.Y, -axis.X, 0)}
}

// at is the point t of the skull's length forward of the condyle, side
// metres to the right and drop metres down.
func (frame walrusSkullFrame) at(t, side, drop float64) maths.Vec {
	return frame.condyle.Add(frame.axis.Scale(walrusSkullLength * t)).
		Add(frame.down.Scale(drop)).Add(maths.V(0, 0, side))
}

// local is where p lies in the skull's frame: the share of the skull's
// length forward of the condyle, the metres to the right and the metres
// down. It undoes at.
func (frame walrusSkullFrame) local(p maths.Vec) (t, side, drop float64) {
	offset := p.Sub(frame.condyle)
	return offset.Dot(frame.axis) / walrusSkullLength, offset.Z,
		offset.Dot(frame.down)
}

// walrusSkull hangs the skull from the atlas, and from the skull the lower
// jaw and the two tusks.
func walrusSkull(body *walrusBones, atlas string) {
	const skullThick, jawThick, tuskThick = .10, .03, .03
	first, _ := body.bone(atlas)
	frame := walrusSkullFrameAt(first.Tail())
	body.add("skull", atlas, frame.condyle, frame.at(1, 0, 0), skullThick,
		walrusCranium{frame})
	jaw := walrusMandible{frame}
	line := jaw.Middle()
	body.add("mandible", "skull", line[0], line[1], jawThick, jaw)
	for _, side := range []float64{-1, 1} {
		canine := walrusCanine{frame: frame, side: side}
		body.add("tusk "+walrusSideName(side), "skull", canine.at(0),
			canine.at(1), tuskThick, canine)
	}
}

// walrusCranium is the skull without its lower jaw: the braincase behind,
// narrow between the eyes, then the broad, deep maxilla that holds the
// tusks, and an arch of bone under each eye.
type walrusCranium struct{ frame walrusSkullFrame }

// walrusCraniumRings is the rings along the skull.
const walrusCraniumRings = 24

// walrusCraniumProfile is the skull's half width, half height and drop
// below its axis at stations along it, from the condyle to the snout.
// Chosen: a broad braincase, a narrow waist between the orbits, a broad
// deep maxilla over the tusk roots.
var walrusCraniumProfile = [][4]float64{
	{0, .11, .08, 0}, {.2, .15, .11, 0}, {.45, .09, .08, -.01},
	{.7, .15, .12, .03}, {.88, .13, .11, .04}, {1, .08, .06, .03},
}

// Build adds the skull and its two arches to the mesh.
func (skull walrusCranium) Build(vs []maths.Vec, fs [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	frame := skull.frame
	path := func(t float64) maths.Vec {
		return frame.at(t, 0, walrusProfileAt(t, 3))
	}
	half := func(t float64) (float64, float64) {
		return walrusProfileAt(t, 1), walrusProfileAt(t, 2)
	}
	vs, fs = skeleton.Sweep(vs, fs, walrusCraniumRings, path, half,
		maths.V(0, 1, 0), place)
	for _, side := range []float64{-1, 1} {
		arch := skeleton.NewArc([4]maths.Vec{frame.at(.70, side*.12, .05),
			frame.at(.60, side*.17, .06), frame.at(.40, side*.17, .05),
			frame.at(.30, side*.12, .03)}, .012, .02, maths.V(0, 1, 0))
		vs, fs = arch.Build(vs, fs, place)
	}
	return vs, fs
}

// Middle is the skull's axis, condyle to snout.
func (skull walrusCranium) Middle() []maths.Vec {
	return []maths.Vec{skull.frame.at(0, 0, 0), skull.frame.at(1, 0, 0)}
}

// walrusProfileAt is column of walrusCraniumProfile at t, eased between
// stations.
func walrusProfileAt(t float64, column int) float64 {
	rows := walrusCraniumProfile
	for i := 1; i < len(rows); i++ {
		if t <= rows[i][0] {
			share := maths.Smooth(rows[i-1][0], rows[i][0], t)
			return rows[i-1][column] + (rows[i][column]-rows[i-1][column])*share
		}
	}
	return rows[len(rows)-1][column]
}

// walrusMandible is the lower jaw: two branches from the jaw joints under
// the braincase, meeting at the chin between the tusks. Sizes chosen.
type walrusMandible struct{ frame walrusSkullFrame }

// Build adds the jaw to the mesh.
func (jaw walrusMandible) Build(vs []maths.Vec, fs [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	frame := jaw.frame
	for _, side := range []float64{-1, 1} {
		branch := skeleton.NewArc([4]maths.Vec{frame.at(.22, side*.11, .10),
			frame.at(.38, side*.11, .16), frame.at(.58, side*.06, .19),
			frame.at(.72, side*.025, .17)}, .016, .035, maths.V(0, 1, 0))
		vs, fs = branch.Build(vs, fs, place)
	}
	chin := skeleton.NewRod(frame.at(.66, 0, .17), frame.at(.76, 0, .16), .03, .8)
	return chin.Build(vs, fs, place)
}

// Middle is the jaw's line, joint to chin.
func (jaw walrusMandible) Middle() []maths.Vec {
	return []maths.Vec{jaw.frame.at(.22, 0, .10), jaw.frame.at(.72, 0, .17)}
}

// walrusCanine is one tusk on the side given, laid in the skull's frame.
type walrusCanine struct {
	frame walrusSkullFrame
	side  float64
}

// walrusCanineRings is the rings along a tusk.
const walrusCanineRings = 30

// at is the middle line of the tusk at t, from its root in the maxilla at
// nought to its worn tip at one. The tusk turns at a steady rate in the
// skull's middle plane, so its line is an arc, and leans out a little and
// back in toward the tip across it.
func (canine walrusCanine) at(t float64) maths.Vec {
	const (
		rootStation, rootSide, rootDrop = .60, .10, .05
		outward, inward                 = .04, .06
	)
	frame := canine.frame
	s := walrusTuskLength * t
	turned := walrusTuskStart + walrusTuskCurve*s
	forward := (math.Cos(walrusTuskStart) - math.Cos(turned)) / walrusTuskCurve
	down := (math.Sin(turned) - math.Sin(walrusTuskStart)) / walrusTuskCurve
	root := frame.at(rootStation, canine.side*rootSide, rootDrop)
	lean := canine.side * (outward*t - inward*t*t)
	return root.Add(frame.axis.Scale(forward)).Add(frame.down.Scale(down)).
		Add(maths.V(0, 0, lean))
}

// Build adds the tusk to the mesh, its section that of the hide's tusk.
func (canine walrusCanine) Build(vs []maths.Vec, fs [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	half := func(t float64) (float64, float64) {
		r := walrusTuskRadius(t, 0)
		return r, r
	}
	return skeleton.Sweep(vs, fs, walrusCanineRings, canine.at, half,
		maths.V(1, 0, 0), place)
}

// Middle is the tusk's line, root to tip.
func (canine walrusCanine) Middle() []maths.Vec {
	return []maths.Vec{canine.at(0), canine.at(.5), canine.at(1)}
}

// A flipper is a hand: a row of carpals or tarsals, five long bones of the
// palm or sole, and the bones of five digits, two in the first digit and
// three in each other, as a carnivoran has (chosen; no walrus source read
// for the count). Lengths in metres, chosen. In the fore flipper "the
// digits are shorter after the first"; in the hind flipper "the first and
// fifth digits are longer" (OBIS-SEAMAP species profile, as read).
//
// The order is sourced; the lengths are chosen to keep it, and short
// enough that the flipper stays inside the hide's mesh box.
var (
	walrusForePalm   = [walrusDigitCount]float64{.11, .115, .11, .105, .10}
	walrusForeDigits = [walrusDigitCount][]float64{
		{.12, .10}, {.09, .07, .05}, {.085, .065, .045},
		{.08, .06, .04}, {.075, .055, .035},
	}
	walrusHindSole   = [walrusDigitCount]float64{.14, .12, .11, .12, .14}
	walrusHindDigits = [walrusDigitCount][]float64{
		{.13, .10}, {.09, .07, .05}, {.085, .065, .045},
		{.09, .07, .05}, {.12, .09, .06},
	}
)

// How the rays of a flipper are spread. Chosen, so that a flipper comes
// out broad and nearly square, about as wide at the ends of its digits
// as it is long from the wrist or the ankle.
const (
	// walrusForeSpacing and walrusHindSpacing are the distance between
	// neighbouring rays where they leave the wrist or the ankle, in
	// metres.
	walrusForeSpacing = .050
	walrusHindSpacing = .052
	// walrusForeFan and walrusHindFan are the angle between neighbouring
	// rays, in radians.
	walrusForeFan = .09
	walrusHindFan = .12
)

// walrusHand is how a flipper's rays are laid: the joint at its root
// (wrist or ankle), along the flipper and across it toward the first
// digit; which flipper and side, and the name of its palm bones; the
// lengths of each ray's bones; the spacing and fan of the rays; and the
// carpal or tarsal each ray hangs from.
type walrusHand struct {
	root, along, across  maths.Vec
	kind, side, palmName string
	palm                 [walrusDigitCount]float64
	digits               [walrusDigitCount][]float64
	spacing, fan         float64
	carriers             [walrusDigitCount]string
}

// walrusFlipperFrame is along and across for a flipper pointing along
// direction, its sole down: across is level and square to along, toward
// the first digit.
func walrusFlipperFrame(direction maths.Vec) (along, across maths.Vec) {
	along = direction.Unit()
	across = maths.V(along.Z, 0, -along.X).Unit()
	return along, across
}

// walrusForeLimb lays one fore limb: shoulder blade on the chest wall,
// upper arm back and down inside the body, forearm forward, down and out
// to the wrist on the ground beside the chest, and the hand of bones
// turned forward and outward, flat on the ground, where it carries the
// chest. Places and thicknesses chosen: the wrist lies where the outline's
// underside comes down to the ground.
func walrusForeLimb(body *walrusBones, side float64, shift maths.Vec) {
	const bladeThickness = .012
	name := walrusSideName(side)
	shoulder := maths.V(.85, -.04, side*.22)
	if shift != (maths.Vec{}) {
		shoulder = shoulder.Add(shift)
	}
	blade := shoulder.Add(maths.V(-.40, .88, -side*.20).Unit().
		Scale(walrusScapulaLength))
	body.add("scapula "+name, "thoracic 4", shoulder, blade, bladeThickness,
		skeleton.NewBlade(shoulder, blade, .220282*walrusBullScale/2,
			bladeThickness, .25, maths.V(0, 0, 1)))
	elbow := shoulder.Add(maths.V(-.70, -.55, side*.45).Unit().
		Scale(walrusHumerusLength))
	walrusRod(body, "humerus "+name, "scapula "+name, shoulder, shoulder,
		elbow, .052, .55)
	forearm := maths.V(.52, -.62, side*.50).Unit()
	along, across := walrusFlipperFrame(maths.V(.80, -.02, side*.60))
	wrist := elbow.Add(forearm.Scale(walrusRadiusLength)).Add(across.Scale(.02))
	behind := across.Scale(-.025)
	walrusRod(body, "radius "+name, "humerus "+name, elbow,
		elbow.Add(across.Scale(.02)), wrist, .026, .6)
	// The ulna hangs from the radius, so the forearm bends at the elbow
	// as one.
	walrusRod(body, "ulna "+name, "radius "+name, elbow,
		elbow.Sub(forearm.Scale(.06)).Add(behind),
		elbow.Add(forearm.Scale(walrusUlnaLength-.06)).Add(behind), .030, .6)
	walrusCarpus(body, wrist, along, across, name)
	walrusDigits(body, walrusHand{root: wrist, along: along, across: across,
		kind: "fore", side: name, palmName: "metacarpal", palm: walrusForePalm,
		digits: walrusForeDigits, spacing: walrusForeSpacing, fan: walrusForeFan,
		carriers: [walrusDigitCount]string{"trapezium", "trapezoid", "magnum",
			"unciform", "unciform"}})
}

// walrusCarpus lays the wrist bones, as a carnivoran has them: the
// scapholunar, cuneiform and pisiform, then the trapezium, trapezoid,
// magnum and unciform. The scapholunar hangs from the radius and turns at
// the wrist; the others hang from it. Places chosen.
func walrusCarpus(body *walrusBones, wrist, along, across maths.Vec, side string) {
	hub := "scapholunar " + side
	walrusSmallBone(body, hub, "radius "+side, wrist, wrist.Add(along.Scale(.03)),
		along, .022)
	rows := []struct {
		name          string
		ahead, toward float64
	}{
		{"cuneiform", .03, -.04}, {"pisiform", .02, -.075},
		{"trapezium", .075, .09}, {"trapezoid", .075, .03},
		{"magnum", .075, -.03}, {"unciform", .075, -.09},
	}
	for _, row := range rows {
		at := wrist.Add(along.Scale(row.ahead)).Add(across.Scale(row.toward))
		walrusSmallBone(body, row.name+" "+side, hub, at, at, along, .018)
	}
}

// walrusSmallBone hangs a short block of bone, a carpal or a tarsal,
// centred at centre and lying along along, from parent, turning at joint.
func walrusSmallBone(body *walrusBones, name, parent string, joint, centre,
	along maths.Vec, radius float64) {
	reach := along.Scale(radius)
	walrusRod(body, name, parent, joint, centre.Sub(reach), centre.Add(reach),
		radius, .85)
}

// walrusRod hangs a straight bone from parent, turning at joint, from from
// to to, of that radius and shaft. Its far end in the tree is to, or from
// when to is the joint itself.
func walrusRod(body *walrusBones, name, parent string, joint, from,
	to maths.Vec, radius, shaft float64) {
	tail := to
	if tail == joint {
		tail = from
	}
	body.add(name, parent, joint, tail, radius,
		skeleton.NewRod(from, to, radius, shaft))
}

// walrusDigits lays the five rays of a flipper: each a long bone of the
// palm or sole hung from its carpal or tarsal, then its digit's bones, the
// rays fanned out from the middle one.
func walrusDigits(body *walrusBones, hand walrusHand) {
	const (
		start  = .10
		radius = .018
		thin   = .82
	)
	normal := hand.along.Cross(hand.across)
	for ray := 0; ray < walrusDigitCount; ray++ {
		offset := float64(2 - ray)
		heading := maths.Rotation(normal, offset*hand.fan).Out(hand.along)
		at := hand.root.Add(hand.along.Scale(start)).
			Add(hand.across.Scale(offset * hand.spacing))
		parent := hand.carriers[ray] + " " + hand.side
		lengths := append([]float64{hand.palm[ray]}, hand.digits[ray]...)
		size := radius
		for bone, length := range lengths {
			name := fmt.Sprintf("%s phalanx %d.%d %s", hand.kind, ray+1, bone, hand.side)
			if bone == 0 {
				name = fmt.Sprintf("%s %d %s", hand.palmName, ray+1, hand.side)
			}
			tip := at.Add(heading.Scale(length))
			walrusRod(body, name, parent, at, at, tip, size, .55)
			parent, at, size = name, tip, size*thin
		}
	}
}

// walrusHindLimb lays one hind limb from the sacrum: the hip bone, a
// short thigh bone and knee inside the body, the long shank back to the
// ankle on the ground beside the tail's root, and the hand of bones
// turned forward along the flank, as the walrus holds it on land. Places
// and thicknesses chosen: hip bone, thigh and knee lie inside the outline
// of the rump.
func walrusHindLimb(body *walrusBones, side float64, sacrum string, shift maths.Vec) {
	const bladeThickness = .015
	name := walrusSideName(side)
	joint := maths.V(-.86, .03, side*.075)
	hip := maths.V(-.90, -.06, side*.11)
	seat := maths.V(-1.12, -.14, side*.08)
	if shift != (maths.Vec{}) {
		joint, hip, seat = joint.Add(shift), hip.Add(shift), seat.Add(shift)
	}
	body.add("innominate "+name, sacrum, joint, seat, bladeThickness,
		skeleton.NewBlade(joint, seat, .045, bladeThickness, .8,
			maths.V(0, 0, 1)))
	knee := hip.Add(maths.V(.25, -.55, side*.78).Unit().Scale(walrusFemurLength))
	walrusRod(body, "femur "+name, "innominate "+name, hip, hip, knee, .040, .65)
	kneecap := knee.Add(maths.V(1, .2, 0).Unit().Scale(.035))
	walrusSmallBone(body, "patella "+name, "femur "+name, knee, kneecap,
		maths.V(0, 1, 0), .02)
	ankle := knee.Add(maths.V(-.86, -.48, side*.05).Unit().
		Scale(walrusTibiaLength))
	walrusRod(body, "tibia "+name, "femur "+name, knee, knee, ankle, .030, .6)
	outside := maths.V(0, 0, side*.035)
	walrusRod(body, "fibula "+name, "tibia "+name, knee, knee.Add(outside),
		ankle.Add(outside), .017, .6)
	along, across := walrusFlipperFrame(maths.V(.80, -.05, side*.60))
	walrusTarsus(body, ankle, along, across, name)
	walrusDigits(body, walrusHand{root: ankle, along: along, across: across,
		kind: "hind", side: name, palmName: "metatarsal", palm: walrusHindSole,
		digits: walrusHindDigits, spacing: walrusHindSpacing, fan: walrusHindFan,
		carriers: [walrusDigitCount]string{"medial cuneiform",
			"intermediate cuneiform", "lateral cuneiform", "cuboid", "cuboid"}})
}

// walrusTarsus lays the ankle bones: the astragalus, which hangs from the
// shank and turns at the ankle, and from it the calcaneus behind and below,
// the navicular, the cuboid and the three cuneiforms. Places chosen.
func walrusTarsus(body *walrusBones, ankle, along, across maths.Vec, side string) {
	hub := "astragalus " + side
	walrusSmallBone(body, hub, "tibia "+side, ankle,
		ankle.Add(along.Scale(.02)), along, .026)
	heel := ankle.Sub(along.Scale(.03)).Sub(maths.V(0, .02, 0)).
		Sub(across.Scale(.02))
	reach := along.Scale(.045)
	walrusRod(body, "calcaneus "+side, hub, heel, heel.Sub(reach),
		heel.Add(reach), .022, .7)
	rows := []struct {
		name          string
		ahead, toward float64
	}{
		{"navicular", .06, .015}, {"cuboid", .06, -.03},
		{"medial cuneiform", .095, .045}, {"intermediate cuneiform", .095, .015},
		{"lateral cuneiform", .095, -.005},
	}
	for _, row := range rows {
		at := ankle.Add(along.Scale(row.ahead)).Add(across.Scale(row.toward))
		walrusSmallBone(body, row.name+" "+side, hub, at, at, along, .016)
	}
}

// walrusAnkleTurn is the angle at one ankle between the shank and the sole
// in the pose given by frames: nought when the flipper lies straight on
// from the shank, as it does trailing behind in the water.
func walrusAnkleTurn(body *walrusBones, frames []skeleton.Frame,
	side string) float64 {
	_, shin := body.bone("tibia " + side)
	_, foot := body.bone("metatarsal 3 " + side)
	knee, ankle := body.rod("tibia " + side)
	base, tip := body.rod("metatarsal 3 " + side)
	shank := frames[shin].Apply(ankle).Sub(frames[shin].Apply(knee)).Unit()
	ray := frames[foot].Apply(tip).Sub(frames[foot].Apply(base)).Unit()
	return math.Acos(math.Max(-1, math.Min(1, shank.Dot(ray))))
}
