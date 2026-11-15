package explain

// The numbers the package works in. Each says whether it is defined,
// measured (and where), or picked (by whom, and when). "Picked here"
// means by the explainer's author, fable-team, 2026-11-12.

// readingRate is how many characters a second a stranger is given to
// read a sentence over a moving picture. Picked here: under the 17 a
// second subtitle guides give for adult viewers, since the eye is also
// on the film and the panel.
const readingRate = 15.0

// beatFlash is how long a beat's square takes to fall to a third, in
// seconds. Picked here, a little longer than the metronome's 0.12 so a
// phone viewer sees it.
const beatFlash = 0.18

// mostBeats is the most beats a bar may hold and still be drawn as
// squares across the panel. Picked here: past eight they are too narrow
// to read on a phone.
const mostBeats = 8

// mostLights is the most lamps the panel holds: two rows of three.
// Picked here, for the layout's room.
const mostLights = 6

// legibility's contrast band for text: ink on ground at 10 to 1 up to
// 12.5 to 1. Defined by the house legibility notes (contrast.md).
const (
	bandLeast = 10.0
	bandMost  = 12.5
)

// hueApart is how far apart, in degrees, ink and ground must be in hue
// unless the ground's saturation is under groundGrey. Defined by the
// house legibility notes (contrast.md): 40 degrees, or 0.35.
const (
	hueApart   = 40.0
	groundGrey = 0.35
)

// srgb's transfer and relative luminance. Defined by IEC 61966-2-1 and
// WCAG 2.1; flare is WCAG's 0.05 added to each side of a contrast ratio.
const (
	srgbKnee   = 0.04045
	srgbOut    = 0.0031308
	srgbLinear = 12.92
	srgbOffset = 0.055
	srgbScale  = 1.055
	srgbGamma  = 2.4
	lumaRed    = 0.2126
	lumaGreen  = 0.7152
	lumaBlue   = 0.0722
	flare      = 0.05
	byteMost   = 255.0
)

// the hue circle, for the hue rule. Defined.
const (
	fullCircle = 360.0
	sextant    = 60.0
	greenTurn  = 2.0
	blueTurn   = 4.0
)

// tintTries is how many times a ground is pulled back toward the theme's
// ground to keep it legible. Picked here: each try halves the tint, and
// after twelve it is the theme's ground to within a step.
const tintTries = 12
