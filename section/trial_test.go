package section_test

import (
	"time"
)

// What the trials draw with, each picked by the test author.
const (
	// trialWide and trialHigh are a frame of 96 by 64 pixels: small
	// enough to draw in a second, and six sections across by four down.
	trialWide = 96
	trialHigh = 64
	// trialSide is the side of a section in pixels: sixteen, which
	// divides both sides of the frame.
	trialSide = 16
	// oneRay is one ray a pixel: the fastest draw, with nothing
	// averaged down.
	oneRay = 1
	// later is the moment of the second frame: a second after the first.
	later = time.Second
	// starRadius is how big a self-lit ball is, in scene units: a little
	// under the width of a section at its distance.
	starRadius = 1.2
	// starGlow is how bright a self-lit ball shows, in linear light:
	// bright enough to stand out from the empty sections.
	starGlow = 0.8
	// sunLight is the strength of the sun in the trials, as the existing
	// section tests have it.
	sunLight = 3.0
)
