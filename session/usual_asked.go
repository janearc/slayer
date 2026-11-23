package session

import (
	"github.com/janearc/slayer/fermat"
)

// the usual request's own numbers, beside usualWide and the rest
const (
	// usualSeed is the seed of the hillside when nobody names one.
	usualSeed = 1
	// usualTilt is how far below level the eye at -from looks, in degrees:
	// straight down.
	usualTilt = 90.0
	// usualScale is a scene scale (the moon's disc, its glow, the close
	// spruces, the rim's named trees, the stage rigs) left at its true
	// size.
	usualScale = 1.0
	// usualOut is where the picture goes when nobody says.
	usualOut = "out/hillside.png"
	// usualSky is the sky when nobody says: the painted one.
	usualSky = "painted"
)

// usualAsked is what was asked of a hillside when nobody has asked
// anything: the usual picture, the usual count of each kind, every
// switch off and none of them said. The command's flags take their
// defaults from it, and a job's request starts from it.
func usualAsked() asked {
	given := usualScene(asked{counts: usualCounts()})
	given.seed = usualSeed
	given.apart = usualApart
	given.exposure = brightness
	given.width, given.height, given.samples = usualWide, usualTall, usualRays
	given.wide = lensWide
	given.tilt = usualTilt
	given.sky = usualSky
	given.out = usualOut
	given.many = fermat.Many
	given.switches = map[string]bool{}
	return given
}

// usualScene is what was asked with the scenes' scales at their usual,
// true size.
func usualScene(given asked) asked {
	given.moonScale, given.moonGlow = usualScale, usualScale
	given.closeScale, given.giants = usualScale, usualScale
	given.rig = usualScale
	return given
}

// usualCounts is how many of each kind the command's count flags give
// when nobody says: the usual stand's, and none of a kind it has not.
func usualCounts() map[string]int {
	counts := map[string]int{}
	for _, name := range flagKinds() {
		counts[name] = usual[name]
	}
	return counts
}

// sequenceAsked is what a sequence draws its frames from: the usual
// request of the set file at scene, as the set's own film and world stand
// over the picture's size and rays. It is checked as the command is.
func sequenceAsked(scene string) (asked, error) {
	given := usualAsked()
	given.set = scene
	return given, given.sound()
}
