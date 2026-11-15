package manufacture

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// the trucks. 2026-11-10: "OOOOH hemtt, kamaz, unimog, tatra, DAF,
// Scania? i *love* me some trucks". One truck kind, a row a truck, each
// row's length, width and height and its axles' spacing and tyres as
// published where a source was found; the vanguard's
// plans/2026-11-10-truck-sources.md says which page each came from.
//
// Every other number is picked, from memory, to look right, and the row
// says so.
//
// A truck stands on its tyres on the ground, its foot centred on the
// thing's origin and its front to +z, the way a stage tower faces its
// audience: x across, y up.

// truck is a truck's row: its overall length, width and height; its
// axles, each as far behind the first as it says, and how many at the
// rear carry twin tyres; how clear of the ground it is under its axles,
// when that is published, zero when not.
//
// Then its tyres' diameter and tread width and the track between their
// centres; how far its front stands ahead of its first axle; the top of
// its frame above the ground, which a cargo bed sits on; its cab and
// what it carries.
//
// And whether it stands on a central backbone tube with swinging
// half-axles, as a Tatra does, instead of two frame rails; whether it
// carries its spare tyre lying behind the cab as its tallest point; and
// how it is painted.
type truck struct {
	length, width, height measure.Length
	axles                 []measure.Length
	clearance             measure.Length
	twins                 int
	tyre, tread, track    measure.Length
	ahead, frame          measure.Length
	cab                   cabin
	carries               bodywork
	backbone, spare       bool
	dressed               looks
}

// cabin is a truck's cab: how long it is, front to back, and the bonnet
// ahead of it, so long and its top so high, when it has one; with no
// bonnet the cab sits over the engine, flat-fronted.
type cabin struct {
	long         measure.Length
	nose, bonnet measure.Length
}

// bodywork is what a truck carries behind its cab: a cargo bed with
// sides so high, or a tractor's fifth wheel for a trailer's kingpin.
type bodywork struct {
	tractor bool
	sides   measure.Length
}

// a truck's matters: its paint, its chassis's dark steel, its tyres'
// rubber, its glass, and its lamps' lenses. the slots are only keys of
// its looks, so the truck borrows five distinct ones already made.
const (
	truckPaint   = body
	truckChassis = spaceStructure
	truckRubber  = spaceFoil
	truckGlass   = spaceMirror
	truckLamp    = spaceSecondary
)

// recessed is how far a truck's paint stops short of its front, so its
// windscreen, grille and lamps stand proud of the paint, not inside it.
const recessed = 0.04

// errTruck is a truck that cannot stand: no axles, axles out of order,
// tyres wider than the truck, or a cab or a body that does not fit.
var errTruck = errors.New("manufacture: a truck needs axles in order " +
	"behind its first, tyres inside its width, and a cab and a body " +
	"that fit its length and height")

// truckLooks is a paint and the rest of a truck, which every truck
// shares: dark steel, black rubber, smoked glass and pale lenses.
func truckLooks(paint string, red, green, blue float64) looks {
	return looks{
		truckPaint:   plain(paint, red, green, blue),
		truckChassis: plain("chassis steel", 0.06, 0.06, 0.065),
		truckRubber:  plain("tyre rubber", 0.025, 0.025, 0.027),
		truckGlass:   plain("smoked glass", 0.03, 0.035, 0.045),
		truckLamp:    plain("lamp lens", 0.8, 0.78, 0.7)}
}

// truckMetres is so many metres, for the rows.
func truckMetres(m float64) measure.Length { return measure.Metres(m) }

// truckRows is the trucks by name.
func truckRows() map[string]maker {
	return map[string]maker{
		"hemtt":    hemttRow(),
		"kamaz":    kamazRow(),
		"unimog":   unimogRow(),
		"tatra":    tatraRow(),
		"scania-r": scaniaRow(),
		"daf-xf":   dafRow(),
	}
}

// hemttRow is Oshkosh's HEMTT, the M977 A4 cargo truck, as Oshkosh's
// HEMTT A4 brochure gives it: 10,389 mm long, 2,438 wide and 3,003 high
// over its spare tyre; a track of 2,007 mm; a wheelbase of 5,334 mm,
// taken here between the middles of its front and rear pairs of axles;
// eight 16.00 R20 tyres and a spare.
//
// The tyres are 1.345 m across (retailers' catalogues); their 438 mm
// tread is trimmed to 431 so that on the published track they keep
// inside the published width.
//
// Picked: the axles of each pair 1.524 m apart (60 inches), its front
// 1.1 m ahead of its first axle, its bed 1.45 m up, its cab 2.2 m long
// over its front axles, its bed's sides 0.6 m, and CARC desert tan.
func hemttRow() truck {
	m := truckMetres
	return truck{length: m(10.389), width: m(2.438), height: m(3.003),
		axles: []measure.Length{m(0), m(1.524), m(5.334), m(6.858)},
		tyre:  m(1.345), tread: m(0.431), track: m(2.007),
		ahead: m(1.1), frame: m(1.45), cab: cabin{long: m(2.2)},
		carries: bodywork{sides: m(0.6)}, spare: true,
		dressed: truckLooks("CARC tan", 0.45, 0.36, 0.22)}
}

// kamazRow is the KAMAZ-43118, a 6x6 cargo truck: 9.27 by 2.55 by 3.11 m
// on 425/85 R21 tyres, 1.256 m across (published in listings); its
// wheelbase 3,690 + 1,320 mm (a body-builder's catalogue, gird.ru, for
// the 43118-46). Picked: its front 1.3 m ahead of its first axle, a
// track of 2.1 m, its bed 1.3 m up, its cab over the engine and 2.0 m
// long, its bed's sides 0.5 m (Wikipedia's 500 mm), and army khaki.
func kamazRow() truck {
	m := truckMetres
	return truck{length: m(9.27), width: m(2.55), height: m(3.11),
		axles: []measure.Length{m(0), m(3.69), m(5.01)},
		tyre:  m(1.256), tread: m(0.425), track: m(2.1),
		ahead: m(1.3), frame: m(1.3), cab: cabin{long: m(2.0)},
		carries: bodywork{sides: m(0.5)},
		dressed: truckLooks("army khaki", 0.2, 0.24, 0.12)}
}

// unimogRow is the Mercedes-Benz Unimog U 5023, as Mercedes-Benz's own
// technical data table gives it: 5,950 mm long, 2,402 wide and 2,873
// high, its axles 3,850 mm apart, 460 mm clear of the ground under its
// axles, on 365/80 R20 to 365/85 R20 tyres; the larger here, 1.129 m
// across.
//
// Picked: a track of 2.0 m; its front 0.85 m ahead of its front
// axle, under a bonnet 0.7 m long whose top is 1.9 m up; its bed 1.2 m
// up; its cab 1.6 m long; a short bed with 0.4 m sides; and Unimog
// orange.
func unimogRow() truck {
	m := truckMetres
	return truck{length: m(5.95), width: m(2.402), height: m(2.873),
		axles: []measure.Length{m(0), m(3.85)}, clearance: m(0.46),
		tyre: m(1.1285), tread: m(0.365), track: m(2.0),
		ahead: m(0.85), frame: m(1.2),
		cab:     cabin{long: m(1.6), nose: m(0.7), bonnet: m(1.9)},
		carries: bodywork{sides: m(0.4)},
		dressed: truckLooks("Unimog orange", 0.85, 0.33, 0.04)}
}

// tatraRow is the Tatra T 815-7, 8x8.1R, the cargo truck of Tatra's own
// sheet for the T 815-7N0R99: 2,500 mm wide, a track of 2,072 mm front
// and rear, 410 mm clear of the ground, on 16.00 R20 tyres, under a
// medium two-door cab over its engine, standing on its backbone tube
// with swinging half-axles.
//
// The sheet gives no length or height, so its 10,720 mm by 3,420 and
// its axles 2,150, 3,660 and 1,450 mm apart are valka.cz's, for an
// 8x8.1R with an armoured cab. The tyres' 438 mm tread is trimmed to
// 428 so that on the published track they keep inside the published
// width.
//
// Picked: its front 1.4 m ahead of its
// first axle, its bed 1.45 m up, its cab 2.2 m long, its bed's sides
// 0.5 m, and army green.
func tatraRow() truck {
	m := truckMetres
	return truck{length: m(10.72), width: m(2.5), height: m(3.42),
		axles:     []measure.Length{m(0), m(2.15), m(5.81), m(7.26)},
		clearance: m(0.41),
		tyre:      m(1.345), tread: m(0.428), track: m(2.072),
		ahead: m(1.4), frame: m(1.45), cab: cabin{long: m(2.2)},
		carries: bodywork{sides: m(0.5)}, backbone: true,
		dressed: truckLooks("army green", 0.13, 0.17, 0.1)}
}

// scaniaRow is a Scania R tractor, 4x2, its normal cab 3.53 m high
// (published). Picked: 2.55 m wide, as a European truck may be; a
// wheelbase of 3.7 m; 315/80 R22.5 tyres, 1.076 m across, twinned on its
// drive axle; its front 1.45 m ahead of its first axle and 6.05 m long;
// its frame 1.0 m up; its cab 2.3 m long; and Scania red.
func scaniaRow() truck {
	m := truckMetres
	return truck{length: m(6.05), width: m(2.55), height: m(3.53),
		axles: []measure.Length{m(0), m(3.7)}, twins: 1,
		tyre: m(1.076), tread: m(0.315), track: m(2.2),
		ahead: m(1.45), frame: m(1.0), cab: cabin{long: m(2.3)},
		carries: bodywork{tractor: true},
		dressed: truckLooks("Scania red", 0.55, 0.04, 0.03)}
}

// dafRow is a DAF XF tractor, 4x2, its wheelbase 3.8 m (published).
// Picked: its Super Space Cab 3.95 m high and 2.3 m long, 2.55 m wide,
// 315/70 R22.5 tyres, 1.013 m across, twinned on its drive axle; its
// front 1.45 m ahead of its first axle and 6.1 m long; its frame 1.0 m
// up; and white.
func dafRow() truck {
	m := truckMetres
	return truck{length: m(6.1), width: m(2.55), height: m(3.95),
		axles: []measure.Length{m(0), m(3.8)}, twins: 1,
		tyre: m(1.013), tread: m(0.315), track: m(2.2),
		ahead: m(1.45), frame: m(1.0), cab: cabin{long: m(2.3)},
		carries: bodywork{tractor: true},
		dressed: truckLooks("DAF white", 0.8, 0.8, 0.78)}
}

// rig is a truck's measures in the scene's unit: half its length and
// width, its height, the z of each axle, its tyres' radius and tread,
// its track, the top of its frame, and its cab: where its paint's front
// is, where the cab's front face and back are, and its floor, which
// clears the tyres it sits over.
type rig struct {
	row                  truck
	half, side, high     float64
	axles                []float64
	radius, tread, track float64
	frame, clearance     float64
	front, face, behind  float64
	floor                float64
}

// beam is the girth of a truck's axle beams: its underside at its
// published clearance, or 8 cm when none is published.
func (at rig) beam() float64 {
	if at.clearance > 0 {
		return at.radius - at.clearance
	}
	return 0.08
}

// spine is the height of a backbone's axis, and its girth.
func (at rig) spine() (height, girth float64) {
	return at.frame - 0.17, 0.17
}

// rigged is the row in the scene's unit, or errTruck when it cannot
// stand.
func (row truck) rigged() (rig, error) {
	at := rig{row: row, half: inScene(row.length) / 2,
		side: inScene(row.width) / 2, high: inScene(row.height),
		radius: inScene(row.tyre) / 2, tread: inScene(row.tread),
		track: inScene(row.track), frame: inScene(row.frame),
		clearance: inScene(row.clearance)}
	if len(row.axles) == 0 {
		return rig{}, errTruck
	}
	first := at.half - inScene(row.ahead)
	behind := -1.0
	for _, axle := range row.axles {
		back := inScene(axle)
		if !(back > behind) {
			return rig{}, errTruck
		}
		behind = back
		at.axles = append(at.axles, first-back)
	}
	at.front = at.half - recessed
	at.face = at.front - inScene(row.cab.nose)
	at.behind = at.face - inScene(row.cab.long)
	at.floor = math.Max(at.frame, 2*at.radius+0.05)
	sides := at.frame + 0.12 + inScene(row.carries.sides)
	inner := at.track/2 > at.tread
	if row.twins > 0 {
		inner = at.track/2 > 2*at.tread+0.02
	}
	bed := row.carries.tractor || (at.frame > 2*at.radius && sides < at.high)
	bonnet := inScene(row.cab.bonnet)
	axis, girth := at.spine()
	clear := at.clearance == 0 || (at.clearance > 0 && at.clearance < at.radius &&
		(!row.backbone || at.clearance < axis-girth))
	fits := at.radius > 0 && inner && clear && row.twins >= 0 &&
		row.twins <= len(row.axles) && at.track/2+at.tread/2 <= at.side+1e-9 &&
		first+at.radius <= at.half &&
		at.axles[len(at.axles)-1]-at.radius >= -at.half &&
		at.face <= at.front && at.behind < at.face &&
		at.behind-0.25 > -at.half && at.floor+0.5 < at.high && bed &&
		(at.face == at.front || (bonnet > at.frame && bonnet < at.high))
	if !fits {
		return rig{}, errTruck
	}
	return at, nil
}

// made is a truck from a seed: its wheels, its chassis, its cab and
// what it carries. It refuses a seed never sown and a row that cannot
// stand.
func (row truck) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	at, err := row.rigged()
	if err != nil {
		return good{}, err
	}
	fine, err := cutInto(24)
	if err != nil {
		return good{}, err
	}
	assembly := &instrumentAssembly{fine: fine,
		object: good{dressed: row.dressed}}
	at.wheels(assembly)
	at.chassis(assembly)
	at.cabin(assembly)
	at.body(assembly)
	made, err := assembly.finish(at.bounds())
	if err != nil {
		return good{}, err
	}
	if err := assembly.shareComponents(); err != nil {
		return good{}, err
	}
	return *made, nil
}

// truckBox lays a box by its centre's place across, its foot's height
// and its centre's place along, so wide, tall and long.
func truckBox(assembly *instrumentAssembly, x, y, z, wide, tall, long float64,
	material slot) {
	assembly.box(wide, tall, long, maths.V(x-wide/2, y, z-long/2), material)
}

// wheels is each axle's tyres, a hub on each outer face, and the axle
// between them: or, on a backbone, a half-axle swinging from the
// backbone to each wheel. The rear axles carry twin tyres, the inner
// one 2 cm inside the outer.
func (at rig) wheels(assembly *instrumentAssembly) {
	for k, z := range at.axles {
		twinned := k >= len(at.axles)-at.row.twins
		for _, side := range []float64{-1, 1} {
			outer := side * (at.track/2 + at.tread/2)
			tyre(assembly, outer, z, at.radius, at.tread, side)
			inside := outer - side*at.tread
			if twinned {
				inside -= side * 0.02
				tyre(assembly, inside, z, at.radius, at.tread, side)
				inside -= side * at.tread
			}
			hub := maths.V(outer-side*0.05, at.radius, z)
			assembly.rodAlong(hub, maths.V(side*0.04, 0, 0), at.radius*0.35,
				truckChassis)
			if at.row.backbone {
				axis, _ := at.spine()
				assembly.rod(maths.V(0, axis, z),
					maths.V(inside, at.radius, z), 0.07, truckChassis)
			}
		}
		if at.row.backbone {
			// the differential under the backbone, its foot at the
			// published clearance, when there is one
			if at.clearance > 0 {
				axis, girth := at.spine()
				truckBox(assembly, 0, at.clearance, z, 0.4,
					axis-girth-at.clearance, 0.4, truckChassis)
			}
			continue
		}
		reach := at.track/2 - at.tread/2
		if twinned {
			reach -= at.tread + 0.02
		}
		assembly.rodAlong(maths.V(-reach, at.radius, z),
			maths.V(2*reach, 0, 0), at.beam(), truckChassis)
	}
}

// tyre is one tyre, its outer face at outer across, on its axle at z,
// laid inward from that face.
func tyre(assembly *instrumentAssembly, outer, z, radius, tread,
	side float64) {
	assembly.rodAlong(maths.V(outer, radius, z),
		maths.V(-side*tread, 0, 0), radius, truckRubber)
}

// chassis is what the truck stands on: two frame rails, or a Tatra's
// backbone tube, from behind its bumper to its tail; a fuel tank on its
// left, in the widest gap between its wheels; and an exhaust stack just
// behind its cab on its right.
func (at rig) chassis(assembly *instrumentAssembly) {
	front, tail := at.half-0.3, -at.half+0.2
	if at.row.backbone {
		axis, girth := at.spine()
		assembly.rod(maths.V(0, axis, tail), maths.V(0, axis, front), girth,
			truckChassis)
	} else {
		for _, side := range []float64{-1, 1} {
			truckBox(assembly, side*0.43, at.frame-0.3, (front+tail)/2, 0.1,
				0.3, front-tail, truckChassis)
		}
	}
	widest, middle := 0.0, 0.0
	for k := 0; k+1 < len(at.axles); k++ {
		gap := at.axles[k] - at.axles[k+1] - 2*at.radius - 0.3
		if gap > widest {
			widest, middle = gap, (at.axles[k]+at.axles[k+1])/2
		}
	}
	if long := math.Min(1.2, widest); long >= 0.4 {
		tank := maths.V(-(at.side - 0.35), at.frame-0.35, middle+long/2)
		assembly.rodAlong(tank, maths.V(0, 0, -long), 0.3, truckChassis)
	}
	stack := maths.V(at.side-0.35, at.frame, at.behind-0.1)
	assembly.rodAlong(stack, maths.V(0, at.high-at.frame-0.05, 0), 0.07,
		truckChassis)
}

// cabin is the cab: a bonnet ahead of it when it has one; the cab from
// its floor to its roof; its windscreen and side windows; a grille and
// two headlamps on its front; and the bumper across the front below. A
// spare tyre, when it carries one, lies flat on a stand behind the cab,
// its top the truck's highest point.
func (at rig) cabin(assembly *instrumentAssembly) {
	wide := 2*at.side - 0.05
	roof := at.high
	if at.row.spare {
		roof = at.high - 0.15
	}
	front, ahead := wide, at.floor
	if nose := at.front - at.face; nose > 0 {
		front, ahead = 0.75*wide, at.frame
		truckBox(assembly, 0, at.frame, (at.face+at.front)/2, front,
			inScene(at.row.cab.bonnet)-at.frame, nose, truckPaint)
	}
	tall := roof - at.floor
	truckBox(assembly, 0, at.floor, (at.face+at.behind)/2, wide, tall,
		at.face-at.behind, truckPaint)
	proud := at.face + 0.0175
	glass := at.floor + 0.55*tall
	truckBox(assembly, 0, glass, proud, 0.9*wide, 0.38*tall, 0.035, truckGlass)
	for _, side := range []float64{-1, 1} {
		truckBox(assembly, side*(wide/2+0.01), glass,
			at.face-0.3*(at.face-at.behind), 0.02, 0.3*tall,
			0.45*(at.face-at.behind), truckGlass)
	}
	lower := ahead + 0.12
	nose := at.front + 0.015
	truckBox(assembly, 0, lower, nose, 0.5*front, 0.35, 0.03, truckChassis)
	for _, side := range []float64{-1, 1} {
		truckBox(assembly, side*(front/2-0.25), lower, nose, 0.24, 0.14, 0.03,
			truckLamp)
	}
	truckBox(assembly, 0, at.frame-0.45, at.half-0.12, wide, 0.32, 0.24,
		truckChassis)
	if at.row.spare {
		centre := at.behind - 0.1 - at.radius
		truckBox(assembly, 0, at.frame, centre, 2*at.radius,
			at.high-at.tread-at.frame, 0.3, truckChassis)
		assembly.rodAlong(maths.V(0, at.high-at.tread, centre),
			maths.V(0, at.tread, 0), at.radius, truckRubber)
	}
}

// body is what the truck carries behind its cab: a bed's floor on its
// frame and its four sides, or a tractor's deck plate and fifth wheel
// over its rear axle.
func (at rig) body(assembly *instrumentAssembly) {
	if at.row.carries.tractor {
		rear := at.axles[len(at.axles)-1]
		truckBox(assembly, 0, at.frame, rear+0.3, 1.0, 0.12, 1.0, truckChassis)
		truckBox(assembly, 0, at.frame+0.12, rear+0.3, 0.8, 0.08, 0.8,
			truckChassis)
		return
	}
	start, tail := at.behind-0.25, -at.half
	if at.row.spare {
		start -= 2*at.radius + 0.2
	}
	long, middle := start-tail, (start+tail)/2
	wide := 2 * at.side
	sides := inScene(at.row.carries.sides)
	up := at.frame + 0.12
	truckBox(assembly, 0, at.frame, middle, wide, 0.12, long, truckPaint)
	for _, side := range []float64{-1, 1} {
		truckBox(assembly, side*(at.side-0.025), up, middle, 0.05, sides, long,
			truckPaint)
	}
	truckBox(assembly, 0, up, tail+0.025, wide-0.1, sides, 0.05, truckPaint)
	truckBox(assembly, 0, up, start-0.025, wide-0.1, sides, 0.05, truckPaint)
}

// bounds is the truck's box: its length, its width and its height, from
// the ground up.
func (at rig) bounds() maths.Box {
	return maths.SegmentBox(maths.V(-at.side, 0, -at.half),
		maths.V(at.side, at.high, at.half), spaceZero)
}
