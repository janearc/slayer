package pinnipeds

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals"
	"github.com/janearc/slayer/maths"
)

// The walrus's outline is a table of stations from the nose to the tail.
// Each gives the half width of the body there, the heights of its back
// and of its underside above the ground, and how square its section is.
// The stations are eased into one smooth profile and the hide is cut from
// that. The bones carry it into a pose; they do not shape it.
//
// What is read. A bull "can be as long as 320 cm", "weighing up to 1200
// to 1500 kg" (Animal Diversity Web, Odobenus rosmarus); "about 315 cm"
// and "about 1500 kg" (Handbook of the Mammals of the World, on Plazi's
// walrus account).
//
// What is picked. No table of widths and heights at stations was reached
// on a page that could be read, so every row below is picked: a bull
// lying with its chest propped on its fore flippers, the head carried
// level with the shoulders, the belly flat on the ground.
//
// The whole is
// held to the two numbers read. It is 3.09 m from nose to tail, and by
// the trapezoid rule over the rows it encloses about 1.4 cubic metres
// over the ground, about 1400 kg at a density of 1.03.

// walrusStation is one row of the outline: how far behind the nose it
// lies, the half width, the heights of the back and of the underside
// above the ground, all in metres, and the exponent of its section, two
// for an ellipse and more for a squarer one.
type walrusStation struct{ behind, wide, top, bottom, square float64 }

// walrusStations is the outline from the nose to the tail. Picked, as
// said above. The underside from the shoulders back lies a little under
// the ground, so that the ground flattens it.
var walrusStations = walrusStationsOf(defaultPinniped("walrus"))

const (
	// walrusPadFlesh is how far the face of the muzzle pads lies in front
	// of the skull's snout, in metres. Picked.
	walrusPadFlesh = .06
	// walrusOutlineSamples is how many sections the eased outline holds.
	walrusOutlineSamples = 2048
	// walrusOutlineRim is the width over which the edge of the pads' flat
	// face, and of the tail's end, is rounded, in metres. Picked.
	walrusOutlineRim = .03
	// walrusBackReach is how many rims below its crown the outline's back
	// is looked for, and walrusBackSteps how many times that span is
	// halved to find it. Picked: the rim never lowers the back by more
	// than two of itself, and forty halvings leave under a picometre.
	walrusBackReach = 2
	walrusBackSteps = 40
)

// errWalrusOutline is a table of stations the outline cannot be made
// from: out of order, or with a section of no width or no height.
var errWalrusOutline = errors.New("walrus: the outline's stations cannot be eased")

// walrusSection is one section of the outline in rest space: where it
// lies along the body, its half width and half height, the height of its
// middle, and the exponent of its shape.
type walrusSection struct{ x, wide, high, centre, square float64 }

// walrusOutline is the sections eased onto an even spacing in x, so that
// a lookup is one step and one blend.
type walrusOutline struct {
	first, last, step float64
	sections          []walrusSection
}

var (
	walrusOutlineOnce   sync.Once
	walrusOutlineShared *walrusOutline
	walrusOutlineFault  error
)

// walrusTheOutline is the outline, built on first use.
func walrusTheOutline() (*walrusOutline, error) {
	walrusOutlineOnce.Do(func() {
		walrusOutlineShared, walrusOutlineFault = walrusOutlineOf(
			walrusSections(walrusStations, walrusNoseAt(walrusFrame)))
	})
	return walrusOutlineShared, walrusOutlineFault
}

// outline belongs to the same skull frame as the rest rig.
func (rig *walrusRig) outline() (*walrusOutline, error) {
	if rig.body == walrusRest {
		return walrusTheOutline()
	}
	rig.outlineOnce.Do(func() {
		rig.cachedOutline, rig.outlineFault = walrusOutlineOf(
			walrusSections(walrusStationsOf(rig.body.profile), walrusNoseAt(rig.frame)))
	})
	return rig.cachedOutline, rig.outlineFault
}

// walrusNoseAt is where along the body the face of the muzzle pads lies:
// walrusPadFlesh in front of the snout of the skull whose frame is given.
func walrusNoseAt(frame walrusSkullFrame) float64 {
	return frame.at(1, 0, 0).X + walrusPadFlesh
}

// walrusSections is the stations as sections in rest space, tail first,
// for a nose at the given place along the body.
func walrusSections(stations []walrusStation, nose float64) []walrusSection {
	sections := make([]walrusSection, 0, len(stations))
	for i := len(stations) - 1; i >= 0; i-- {
		row := stations[i]
		sections = append(sections, walrusSection{
			x:      nose - row.behind,
			wide:   row.wide,
			high:   (row.top - row.bottom) / 2,
			centre: walrusFloor + (row.top+row.bottom)/2,
			square: row.square,
		})
	}
	return sections
}

// walrusOutlineOf is the sections eased by cubics through them onto an
// even spacing. It refuses fewer than two sections, sections that do not
// run forward, and a section with no width, no height or an exponent
// under one.
func walrusOutlineOf(sections []walrusSection) (*walrusOutline, error) {
	if len(sections) < 2 {
		return nil, fmt.Errorf("%w: %d sections", errWalrusOutline, len(sections))
	}
	for i, section := range sections {
		if !(section.wide > 0) || !(section.high > 0) || !(section.square >= 1) {
			return nil, fmt.Errorf("%w: section %d is %.3f wide, %.3f high, "+
				"exponent %.2f", errWalrusOutline, i, section.wide, section.high,
				section.square)
		}
		if i > 0 && !(section.x > sections[i-1].x) {
			return nil, fmt.Errorf("%w: section %d at %.3f, after %.3f",
				errWalrusOutline, i, section.x, sections[i-1].x)
		}
	}
	first, last := sections[0].x, sections[len(sections)-1].x
	outline := &walrusOutline{first: first, last: last,
		step: (last - first) / (walrusOutlineSamples - 1)}
	for i := 0; i < walrusOutlineSamples; i++ {
		outline.sections = append(outline.sections,
			walrusSectionAt(sections, first+float64(i)*outline.step))
	}
	return outline, nil
}

// walrusSectionAt is the section at x, from the cubics through the
// sections. A cubic may dip a little past the least of its two ends, so
// the width and the height are held to half the smaller end.
func walrusSectionAt(sections []walrusSection, x float64) walrusSection {
	i := 0
	for i < len(sections)-2 && x > sections[i+1].x {
		i++
	}
	a, b := sections[i], sections[i+1]
	blend := func(pick func(walrusSection) float64) float64 {
		return mammals.Hermite(a.x, b.x, pick(a), pick(b),
			walrusSlope(sections, i, pick), walrusSlope(sections, i+1, pick), x)
	}
	wide := blend(func(s walrusSection) float64 { return s.wide })
	high := blend(func(s walrusSection) float64 { return s.high })
	return walrusSection{x: x,
		wide:   math.Max(wide, math.Min(a.wide, b.wide)/2),
		high:   math.Max(high, math.Min(a.high, b.high)/2),
		centre: blend(func(s walrusSection) float64 { return s.centre }),
		square: a.square + (b.square-a.square)*maths.Smooth(a.x, b.x, x)}
}

// walrusSlope is the slope of one quantity at section i, across its
// neighbours.
func walrusSlope(sections []walrusSection, i int,
	pick func(walrusSection) float64) float64 {
	low, high := max(i-1, 0), min(i+1, len(sections)-1)
	return (pick(sections[high]) - pick(sections[low])) /
		(sections[high].x - sections[low].x)
}

// at is the section at x, held at the ends.
func (outline *walrusOutline) at(x float64) walrusSection {
	place := (x - outline.first) / outline.step
	i := max(0, min(int(math.Floor(place)), len(outline.sections)-2))
	t := math.Min(math.Max(place-float64(i), 0), 1)
	a, b := outline.sections[i], outline.sections[i+1]
	return walrusSection{x: x,
		wide:   a.wide + t*(b.wide-a.wide),
		high:   a.high + t*(b.high-a.high),
		centre: a.centre + t*(b.centre-a.centre),
		square: a.square + t*(b.square-a.square)}
}

// ring is how far out of its section a point lies, one on the section's
// edge: up is its height over the section's middle and out its distance
// from the middle plane, each as a share of the section's half size.
func (section walrusSection) ring(up, out float64) float64 {
	n := section.square
	return math.Pow(math.Pow(math.Abs(up), n)+math.Pow(math.Abs(out), n), 1/n)
}

// distance is about how far p lies outside the outline, negative inside:
// its section's ring less one, over the length of the ring's gradient,
// and past either end, how far past it, the corner between rounded.
func (outline *walrusOutline) distance(p maths.Vec) float64 {
	section := outline.at(p.X)
	up := math.Abs(p.Y-section.centre) / section.high
	out := math.Abs(p.Z) / section.wide
	ring := section.ring(up, out)
	gap := -math.Min(section.high, section.wide)
	if ring > 0 {
		n := section.square
		slope := math.Pow(ring, 1-n) * math.Hypot(
			math.Pow(up, n-1)/section.high, math.Pow(out, n-1)/section.wide)
		gap = (ring - 1) / slope
	}
	past := math.Max(p.X-outline.last, outline.first-p.X)
	return walrusSoftMax(gap, past, walrusOutlineRim)
}

// flank is how far from the middle plane the outline's surface lies at x
// along the body and height y, or false where the outline has no surface
// at that height.
func (outline *walrusOutline) flank(x, y float64) (float64, bool) {
	if x < outline.first || x > outline.last {
		return 0, false
	}
	section := outline.at(x)
	up := math.Abs(y-section.centre) / section.high
	if up >= 1 {
		return 0, false
	}
	n := section.square
	return section.wide * math.Pow(1-math.Pow(up, n), 1/n), true
}

// crown is the height of the outline's back at x along the body and z
// from the middle plane, or false where the outline is not that wide.
func (outline *walrusOutline) crown(x, z float64) (float64, bool) {
	if x < outline.first || x > outline.last {
		return 0, false
	}
	section := outline.at(x)
	out := math.Abs(z) / section.wide
	if out >= 1 {
		return 0, false
	}
	n := section.square
	return section.centre + section.high*math.Pow(1-math.Pow(out, n), 1/n), true
}

// back is the height at x along the body and z from the middle plane where
// distance reads nought: the crown, lowered where the rim rounds the end of
// the body, so that a landmark set there lies on the outline as it is drawn
// and not on its sharp section. It is false where crown is, and where the
// rim leaves no back within walrusBackReach rims below the crown.
func (outline *walrusOutline) back(x, z float64) (float64, bool) {
	high, ok := outline.crown(x, z)
	if !ok {
		return 0, false
	}
	low := high - walrusBackReach*walrusOutlineRim
	if outline.distance(maths.V(x, low, z)) >= 0 {
		return 0, false
	}
	low, high = maths.Bisect(low, high, walrusBackSteps, func(y float64) bool { return outline.distance(maths.V(x, y, z)) < 0 })
	return (low + high) / 2, true
}

// girth is the distance round the outline at x, by summing short chords
// of its section.
func (outline *walrusOutline) girth(x float64) float64 {
	const chords = 720
	section := outline.at(x)
	at := func(i int) (float64, float64) {
		angle := 2 * math.Pi * float64(i) / chords
		c, s := math.Cos(angle), math.Sin(angle)
		reach := 1 / section.ring(s, c)
		return section.wide * c * reach, section.high * s * reach
	}
	total := 0.
	lastZ, lastY := at(0)
	for i := 1; i <= chords; i++ {
		z, y := at(i)
		total += math.Hypot(z-lastZ, y-lastY)
		lastZ, lastY = z, y
	}
	return total
}

// volume is the space the outline encloses over a ground at the height
// floor, in cubic metres, by the trapezoid rule over its sections.
func (outline *walrusOutline) volume(floor float64) float64 {
	total := 0.
	for i := 1; i < len(outline.sections); i++ {
		a, b := outline.sections[i-1], outline.sections[i]
		total += (a.area(floor) + b.area(floor)) / 2 * outline.step
	}
	return total
}

// area is the area of the section that lies over the height floor,
// summed in thin upright strips across its half width and doubled.
func (section walrusSection) area(floor float64) float64 {
	const strips = 256
	n := section.square
	half := 0.
	for i := 0; i < strips; i++ {
		out := (float64(i) + .5) / strips
		reach := section.high * math.Pow(1-math.Pow(out, n), 1/n)
		top := section.centre + reach
		bottom := math.Max(section.centre-reach, floor)
		half += math.Max(top-bottom, 0) * section.wide / strips
	}
	return 2 * half
}

// walrusStations copies the authored outline into the scaffold's units.
func walrusStationsOf(profile Profile) []walrusStation {
	stations := make([]walrusStation, len(profile.WalrusOutline))
	for i, row := range profile.WalrusOutline {
		stations[i] = walrusStation{behind: row[0], wide: row[1], top: row[2], bottom: row[3], square: row[4]}
	}
	return stations
}

// checkWalrusOutline permits a small change to trunk dimensions on the
// existing rig; its head and flipper attachments still use fixed anatomy.
func (rig *walrusRig) checkWalrusOutline() error {
	stations := walrusStationsOf(rig.body.profile)
	if len(stations) != len(walrusStations) {
		return fmt.Errorf("walrus: outline must retain scaffold stations")
	}
	changed := false
	for i, row := range stations {
		base := walrusStations[i]
		if row == base {
			continue
		}
		changed = true
		height := base.top - base.bottom
		if i < 7 || i >= 16 || row.behind != base.behind || row.square != base.square ||
			math.Abs(row.wide-base.wide) > .05*base.wide ||
			math.Abs(row.top-base.top) > .05*height || math.Abs(row.bottom-base.bottom) > .05*height {
			return fmt.Errorf("walrus: outline station %d exceeds the shared rig's range", i)
		}
	}
	if !changed {
		return nil
	}
	outline, err := rig.outline()
	if err != nil {
		return err
	}
	for name, form := range rig.body.forms {
		inside := false
		for _, prefix := range []string{"cervical", "thoracic", "lumbar", "sacral", "caudal", "skull", "mandible", "sternebra", "scapula", "humerus", "innominate", "femur"} {
			if strings.HasPrefix(name, prefix) {
				inside = true
				break
			}
		}
		if inside {
			for _, p := range form.Middle() {
				if !(outline.distance(p) < 0) {
					return fmt.Errorf("walrus: %s lies outside the profile outline", name)
				}
			}
		}
	}
	return nil
}
