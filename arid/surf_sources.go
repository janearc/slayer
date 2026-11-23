// This file holds the numbers the surf is worked from. Each says where it
// comes from: a publication and the page it was read on, a definition, or
// a choice, said to be one.
//
// [Airy] linear wave theory, https://en.wikipedia.org/wiki/Airy_wave_theory
// [CW-SW] https://www.coastalwiki.org/wiki/Shallow-water_wave_theory
// [CW-RU] https://www.coastalwiki.org/wiki/Wave_run-up
// [SWAN] https://swanmodel.sourceforge.io/online_doc/swantech/node16.html
// [IRI] https://en.wikipedia.org/wiki/Iribarren_number

package arid

import "math"

// gravity is the standard acceleration of gravity, in metres a second
// squared: the 3rd CGPM's (1901) definition, the value oceantex's
// sources.yaml holds as gravity.standard.
const gravity = 9.80665

// twice is the two of linear theory's group speed, c_g = c / 2 (1 + 2 k h
// / sinh 2 k h) [CW-SW eq. 3.23], where it halves the phase speed and
// doubles k h, and of Godunov's update, which halves a sum.
const twice = 2

// significantDeviations is how many standard deviations of the surface's
// height make its significant height: Hs = 4 sqrt(m0), by definition.
const significantDeviations = 4

// rootMeanSquareShare is the root-mean-square height of the waves over
// their significant height: Hrms = sqrt(8 m0) and Hs = 4 sqrt(m0), so it
// is one over the root of two, by definition.
const rootMeanSquareShare = 1 / math.Sqrt2

// surfSaturated is the most the root-mean-square height of the waves can
// be of the depth inside the surf: Hrms = 0.42 h, Thornton and Guza 1982,
// J. Geophys. Res. 87, 9499-9508, as oceantex's sources.yaml holds it
// (breaking.saturated), read there.
const surfSaturated = 0.42

// breakerIndex is gamma in Battjes and Janssen's (1978) greatest height,
// Hmax = gamma d: the average 0.73 of Battjes and Stive (1985), as [SWAN]
// quotes it.
const breakerIndex = 0.73

// spillingBelow and surgingAbove are the surf similarity numbers, offshore,
// that part the breakers' kinds: spilling under 0.5, plunging between,
// surging or collapsing over 3.3. Battjes 1974, Coastal Engineering
// Proceedings 1(14), 466-480, as [IRI] tabulates it.
const (
	spillingBelow = 0.5
	surgingAbove  = 3.3
)

// whitecapScale and whitecapPower are Monahan and O'Muircheartaigh's
// (1980, J. Phys. Oceanogr. 10, 2094-2099) whitecap cover of the open sea,
// W = 3.84e-6 U^3.41, U the wind at 10 m in metres a second, as oceantex's
// sources.yaml holds them (monahan.scale, monahan.power).
const (
	whitecapScale = 3.84e-6
	whitecapPower = 3.41
)

// surfFoamReflectance is the reflectance in the visible of fresh, dense
// foam, Whitlock, Bartlett and Gurganus 1982, Geophys. Res. Lett. 9,
// 719-722, as oceantex's sources.yaml holds it (foam.fresh). It is the
// white water a breaking wave leaves.
const surfFoamReflectance = 0.55

// whitecapReflectance is the effective reflectance in the visible of
// whitecaps of every age at sea, Koepke 1984, Appl. Opt. 23, 1816-1824,
// as oceantex's sources.yaml holds it (koepke.foam.reflectance).
const whitecapReflectance = 0.22

// The run-up of Stockdon, Holman, Howd and Sallenger 2006, Coastal
// Engineering 53, 573-588, eq. 3, as [CW-RU] gives it: R2 = 1.1 (setup +
// 0.5 sqrt(Sw^2 + Sig^2)) with setup = 0.35 H xi, Sw = 0.75 H xi and Sig =
// 0.06 sqrt(H L), for xi at least 0.3; R2 = 0.043 sqrt(H L) under it.
const (
	runupExceedance   = 1.1
	setupScale        = 0.35
	swashIncident     = 0.75
	swashInfragravity = 0.06
	dissipativeRunup  = 0.043
	dissipativeBelow  = 0.3
)

// swashDeviations is how many standard deviations of the swash make its
// significant height, S, by the convention of Hs = 4 sqrt(m0). [CW-RU]
// does not say it; it is taken from the significant height's definition.
const swashDeviations = 4

// exceedanceDeviations is how far over the mean run-up, in deviations, the
// run-up that two in a hundred exceed stands: Stockdon's 1.1 (setup + S /
// 2) is 1.1 times the setup plus two of S's quarters.
const exceedanceDeviations = 2

// bendFloor is the depth, in root-mean-square heights of the arriving
// waves, under which the waves' shortening is held at that depth's. It is
// picked: shallower, the waves are broken bores held to the depth, and a
// shortening without end at the waterline would make the search's step
// bound without end.
const bendFloor = 1

// foamEdge is the width, in standard deviations of the open surface's
// height, over which a crest goes from water to foam. It is picked, to
// keep the foam's edge from being sharper than the waves' own heights.
const foamEdge = 0.25

// dispersionRounds and dispersionSettled are the most rounds of Newton's
// method the dispersion relation takes, and the share of the wave number
// it stops at, as oceantex's own solution of it takes.
const (
	dispersionRounds  = 30
	dispersionSettled = 1e-12
)

// breakingRounds is how many halvings the fraction of breaking waves is
// searched by: sixty halve the interval from nought to one below any
// float's spacing there. It is a tolerance of the search.
const breakingRounds = 60

// sweepRounds and sweepSettled are the most rounds of the four sweeps the
// waves' bending is solved by, and the change, in grid steps, under which
// a round counts as settled. Both are picked.
const (
	sweepRounds  = 16
	sweepSettled = 1e-9
)
