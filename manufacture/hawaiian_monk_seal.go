package manufacture

import "github.com/janearc/slayer/maths"

// The Hawaiian monk seal, Neomonachus schauinslandi, as a second form
// for the lofted maker: its own stations, its own length and a plain
// coat. It is drawn over the harbor seal's skeleton, which its outline
// holds.

// monkSealScale is the monk seal's length over the rest space's: 2.13
// m, seven feet, the top of "6 to 7 feet" (NOAA Fisheries, Hawaiian
// monk seal species page), over 1.75 m.
const monkSealScale = 2.13 / sealLength

// monkSealAxillary is the monk seal's mean radius behind the fore
// flippers in the rest space, in metres. No girth of a monk seal was
// found on a page that opens, so it is worked from weight: with this
// radius the outline, scaled to 2.13 m, holds about 0.22 cubic metres,
// 226 kg of seawater, the middle of "400 to 600 pounds" (the same
// page). The volume was worked by hand as cones between the sites.
const monkSealAxillary = .189

// monkSealOutline is the monk seal's outline in the rest space. The
// eight named sites are the Weddell seal's, a seal of the monk seal's
// own subfamily, each with its mean radius, the neck and the ears a
// little over; the head is flatter and broader than the harbor seal's.
// Widths over heights, centre heights and the muzzle and tail stations
// are picked.
var monkSealOutline = sealOutline{
	{x: -.9200, high: .0060, wide: .0080, centre: -.015}, // tail tip
	{x: -.8600, high: .0240, wide: .0280, centre: -.012}, // tail base
	{x: -.8052, high: .0600, wide: .0820, centre: -.008}, // ankles
	{x: -.6619, high: .1150, wide: .1220, centre: 0},     // pelvis
	{x: -.3877, high: .1640, wide: .1740, centre: .012},  // umbilicus
	{x: -.1421, high: .1870, wide: .1980, centre: .025},  // middle
	{x: .1115, high: .1920, wide: .2036, centre: .035},   // sternum
	{x: .3183, high: .1835, wide: .1945, centre: .045},   // axillary
	{x: .5551, high: .1280, wide: .1434, centre: .060},   // neck
	{x: .6650, high: .0880, wide: .1040, centre: .066},   // back of skull
	{x: .7226, high: .0660, wide: .0848, centre: .068},   // ears
	{x: .7500, high: .0600, wide: .0770, centre: .067},   // eyes
	{x: .7750, high: .0520, wide: .0680, centre: .064},   // whisker pads
	{x: .8000, high: .0430, wide: .0580, centre: .060},   // muzzle
	{x: .8220, high: .0300, wide: .0400, centre: .057},   // nose pad
	{x: .8300, high: .0180, wide: .0230, centre: .056},   // nose tip
}

// monkSealPelt is the monk seal's coat, with no marks: "dark gray to
// brown fur on their back, and light gray to yellowish brown fur on
// their belly" (NOAA Fisheries, Hawaiian monk seal species page). The
// colours themselves are picked.
var monkSealPelt = sealPelt{
	back:    maths.RGB(.080, .074, .066),
	belly:   maths.RGB(.380, .340, .260),
	flipper: maths.RGB(.070, .064, .058),
	pad:     maths.RGB(.022, .020, .019),
}

// monkSealForm is the Hawaiian monk seal.
var monkSealForm = sealForm{outline: monkSealOutline,
	source: sealWeddellSource, axillary: monkSealAxillary,
	scale: monkSealScale, pelt: monkSealPelt}

// sealBasking is a monk seal asleep on a beach: nearly flat, the head
// a little raised, the fore flippers out on the sand. Angles picked.
var sealBasking = sealLoftedKind{form: monkSealForm, rests: true,
	wet: .2, lean: .05,
	pose: SealRigPose{Yaw: -.08, Pitch: -.22, HeadPitch: .22,
		Fore: [2]float64{.35, .35}, Hind: [2]float64{.6, .6}}}
