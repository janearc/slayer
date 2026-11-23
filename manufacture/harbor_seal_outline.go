package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
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
	name          string
	behind, round float64
}

// sealSource is one measured animal: its sites from the nose back, its
// straight length nose to tail and the radius at its axillary site, in
// centimetres.
type sealSource struct {
	sites            []sealSourceSite
	length, axillary float64
}

// sealWeddellSource is the one phocid measured site by site that was
// reached: Weddell seal WS12-22, tables 2 and 3 of Shero, Pearson,
// Costa and Burns 2014, PLoS ONE 9(3) e91233, read through Europe PMC
// (PMC3948782). Straight lengths summed from the nose; outer radii.
var sealWeddellSource = sealSource{length: 262.3, axillary: 32.2,
	sites: []sealSourceSite{
		{"ears", 16.1, 11.9},
		{"neck", 41.2, 21.8},
		{"axillary", 76.7, 32.2},
		{"sternum", 107.7, 33.7},
		{"middle", 145.7, 32.8},
		{"umbilicus", 182.5, 28.8},
		{"pelvis", 223.6, 20.2},
		{"ankles", 245.1, 12.1},
	}}

// place is where a site lies on the rest space's X axis: the same share
// of the length behind the nose.
func (source sealSource) place(site sealSourceSite) float64 {
	return sealNoseX - sealLength*site.behind/source.length
}

// radius is the mean radius a site gives a seal whose own axillary
// radius is axillary, in metres: the source's radius there as a share
// of its axillary radius, times the seal's.
func (source sealSource) radius(site sealSourceSite,
	axillary float64,
) float64 {
	return axillary * site.round / source.axillary
}

// harborSealAxillary is the harbor seal's mean radius behind the fore
// flippers, in metres: a girth of 126.7 cm for a 175 cm seal, by girth =
// 10.858 + 0.662 x length in cm. The regression is from a search
// summary of the thesis "Morphology of the Pacific harbor seal"
// (California State University); the thesis itself was not opened.
const harborSealAxillary = .2017

// harborSealOutline is the harbor seal's outline: one spindle, with no
// waist between head and body. The eight named sites are the source's,
// each with its mean radius, the neck and the ears a little over; how
// much wider than high each is, the centre heights, and the muzzle and
// tail stations between the sites and the ends are picked.
var harborSealOutline = sealOutline{
	{x: -.9200, high: .0060, wide: .0080, centre: -.015}, // tail tip
	{x: -.8600, high: .0260, wide: .0300, centre: -.012}, // tail base
	{x: -.8052, high: .0640, wide: .0876, centre: -.008}, // ankles
	{x: -.6619, high: .1227, wide: .1303, centre: 0},     // pelvis
	{x: -.3877, high: .1750, wide: .1858, centre: .012},  // umbilicus
	{x: -.1421, high: .1992, wide: .2116, centre: .025},  // middle
	{x: .1115, high: .2048, wide: .2174, centre: .035},   // sternum
	{x: .3183, high: .1956, wide: .2077, centre: .045},   // axillary
	{x: .5551, high: .1400, wide: .1550, centre: .060},   // neck
	{x: .6650, high: .0960, wide: .1080, centre: .066},   // back of skull
	{x: .7226, high: .0760, wide: .0870, centre: .068},   // ears
	{x: .7500, high: .0690, wide: .0790, centre: .067},   // eyes
	{x: .7750, high: .0590, wide: .0700, centre: .064},   // whisker pads
	{x: .8000, high: .0480, wide: .0590, centre: .060},   // muzzle
	{x: .8220, high: .0330, wide: .0400, centre: .057},   // nose pad
	{x: .8300, high: .0190, wide: .0230, centre: .056},   // nose tip
}

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
		return sealHermite(low.x, high.x, pick(low), pick(high),
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

// sealHermite is the cubic through (x0, y0) and (x1, y1) with slopes m0
// and m1 there, at x between them.
func sealHermite(x0, x1, y0, y1, m0, m1, x float64) float64 {
	span := x1 - x0
	t := (x - x0) / span
	square, cube := t*t, t*t*t
	return (2*cube-3*square+1)*y0 + (cube-2*square+t)*span*m0 +
		(3*square-2*cube)*y1 + (cube-square)*span*m1
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
