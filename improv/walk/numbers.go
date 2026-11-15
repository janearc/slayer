package walk

import "image"

// The numbers the package works in. Each says whether it is defined,
// measured (and where), or picked (by whom, and when). "Picked here"
// means by the walkthrough's author, fable-team, 2026-11-12.

// readingRate is how many characters a second a stranger is given to
// read a sentence. Picked by the explainer's author, the same day, under
// the 17 a second subtitle guides give, as the eye is also on a picture.
const readingRate = 15.0

// filmRate is a short's frames a second. Picked for shorts, 2026-11-12.
const filmRate = 24

// frameSize is a short's frame, a vertical 9 by 16. Defined by the brief.
var frameSize = image.Pt(1080, 1920)

// pictureSize is the picture at the top of the frame, square. Picked
// here: room for a picture a phone shows large, and a panel of three
// parts under it.
var pictureSize = image.Pt(1080, 1080)

// The panel, in pixels of the frame. Picked here, by looking at frames
// full size and at 360 across, the size of a phone, as the explainer's
// layout was: nothing under 34 pixels high, about 11 on a phone.
const (
	margin         = 54
	nameBaseline   = pictureTop + 1080 + 110
	nameSize       = 64.0
	figureTop      = nameBaseline + 60
	figureSize     = 104.0
	unitSize       = 48.0
	saysSize       = 36.0
	figureSaysDrop = 64
	sentenceTop    = figureTop + figureSize + figureSaysDrop + 40
	sentenceSize   = 46.0
	sentenceLead   = 64
	sentenceRows   = 4
	pictureTop     = 0
)

// hostBox is the corner of the panel kept for a host, a looping clip of
// a character laid in by the compositor: the panel's foot on the right.
// Picked here: a third of the frame across, and as tall, under the
// figures. The sentence is narrowed by its width and a margin when a
// script names a host; with none, nothing is kept.
var hostBox = image.Rect(frameSize.X-margin-360, frameSize.Y-margin-360,
	frameSize.X-margin, frameSize.Y-margin)

// The labels on the picture. Picked here.
const (
	labelSize    = 38.0
	labelPad     = 14
	leaderReach  = 120
	leaderWidth  = 4
	dotRadius    = 9
	dotRing      = 4
	tagSize      = 34.0
	tagInset     = 24
	barSize      = 34.0
	barHeight    = 12
	barInset     = 40
	barMostShare = 0.4
)

// dipSeconds is how long the picture dips through the ground at a cut,
// in seconds. Picked here: a fifth of a second.
const dipSeconds = 0.2

// soundLevel is the song's gain under a short, and soundFade the seconds
// it fades in and out over. Picked here: low, under the reading.
const (
	soundLevel = 0.35
	soundFade  = 1.5
)

// stereo is the channels a movie.Sound interleaves. Defined by the
// movie package.
const stereo = 2

// millimetre is a millimetre in metres. Defined.
const millimetre = 0.001

// half is a half. Defined: a pixel's middle, the middle of a frame.
const half = 0.5

// scaleSteps are the round lengths a scale bar takes, times a power of
// ten. Picked here: the usual 1, 2 and 5.
var scaleSteps = []float64{1, 2, 5}

// The contrast band the house legibility notes hold prose to, and the
// least a state chip may have. From the bookshelf's
// legibility/contrast.md.
const (
	proseLeast = 10.0
	proseMost  = 12.5
	chipLeast  = 7.0
)

// roundingSlack is the share a length may be over a round one and still
// take it, for lengths worked out to the round one in floating point.
// Picked here: well over float64's rounding, well under any bar.
const roundingSlack = 1e-9
