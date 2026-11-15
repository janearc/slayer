package pan

import "time"

// The counts the package is made of, each defined.
const (
	nothing = 0 // defined: no frames, no samples
	one     = 1 // defined: the first bar, a single frame
	stereo  = 2 // defined: a sound's channels, left then right
	ends    = 2 // defined: a span's two ends, from and to
	half    = 0.5
	whole   = 1.0 // defined: all of a share
)

// A picture's bytes: red, green, blue and alpha, alpha last. Defined.
const (
	rgba  = 4
	alpha = 3
)

// smoothThree and smoothTwo are smoothstep's coefficients, 3s^2 - 2s^3,
// which eases a window in and out. Defined.
const (
	smoothThree = 3
	smoothTwo   = 2
)

// The numbers the package picked, each by the pan slice's author,
// 2026-11-12.
const (
	// tail is how long the sound fades at the film's end, so a cut
	// inside the piece does not click; half a beat at 66 a minute.
	tail = 450 * time.Millisecond

	// titleAcross and titleDown are where a title stands: centred,
	// high, in the sky, which in a night scene is the darkest part and
	// so where light words read; near the foot they sat on pale mist.
	titleAcross = 0.5
	titleDown   = 0.15

	// titleInk is the theme's role the title is written in.
	titleInk = "dim"

	// defaultRows is the rows a title is laid out in when the script
	// says none: many rows, small type.
	defaultRows = 40
)
