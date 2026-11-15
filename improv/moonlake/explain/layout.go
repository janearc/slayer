package explain

import "image"

// The explainer frame's layout, in pixels of the 1080 by 1920 frame.
// Every place here was picked by the explainer's author, fable-team,
// 2026-11-12, by looking at the frame full size and at 360 across, the
// size of a phone; the README says what each part is.

// frameSize is the explainer frame. Defined: a vertical short, 9 by 16.
var frameSize = image.Pt(1080, 1920)

// margin is the panel's left and right gutter.
const margin = 54

// the beat row: the squares across beatRow, the bar beside them
const (
	beatTop     = 850
	beatRow     = 4*150 + 3*28
	beatHigh    = 150
	beatGap     = 28
	beatHollow  = 16
	barTextSize = 66
	barTextLeft = margin + beatRow + 60
	barBaseline = beatTop + beatHigh/2 + barTextSize*3/8
	// beatLeast is how lit the current beat's square stays between
	// flashes, so the eye can find the place in the bar
	beatLeast = 0.3
	// offShare is how much of its colour a dark square or lamp keeps
	offShare = 0.12
)

// the lamps: two rows of three, each a disc over its label, as large as
// the band between the beat and the chant allows
const (
	lampTop       = 1040
	lampRow       = 235
	lampColumns   = 3
	lampRadius    = 72
	ringRadius    = 90
	ringWidth     = 10
	lampLabelSize = 42
	lampLabelDrop = 138
	// ringShown is the least a ring's flash is drawn at
	ringShown = 0.02
)

// the chant and the sentence
const (
	chantBaseline    = 1585
	chantSize        = 60
	sentenceTop      = 1625
	sentenceSize     = 54
	sentenceLeading  = 74
	sentenceWidth    = 1080 - 2*margin
	sentenceMostRows = 3
)
