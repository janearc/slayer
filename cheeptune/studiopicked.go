package cheeptune

import "math"

// The studio's numbers. Each says whether it is defined, derived or
// picked; the picked ones were picked by the writer of the studio on
// 2026-11-12, with nobody listening, and are the first things to change
// once somebody has.

// studioRate is the rate the studio renders at: the output rate. Defined
// by the piece, which is 44100 samples a second on either machine.
const studioRate = outRate

// studioChannels is the most channels a studio voicing has. Picked:
// sixteen is four layers on each of the four roles.
const studioChannels = 16

// voicesDefault and voicesMost are how many notes a channel sounds at
// once. Picked: eight holds two four-note chords across a change; sixteen
// is where a ringing chord part stops gaining from more.
const (
	voicesDefault = 8
	voicesMost    = 16
)

// stealFade is how long a taken voice takes to fall silent, in seconds.
// Picked: five milliseconds is short enough to free the voice and long
// enough not to click.
const stealFade = 0.005

// studioLowest and studioHighest are the notes a pitched studio channel
// plays, as midi numbers: c0 to c8. Picked: c0 is 16 Hz, under which
// nothing is a pitch, and c8 is the top of a piano.
const (
	studioLowest  = 12
	studioHighest = 108
)

// detuneMost is the most a channel or an oscillator is moved, in cents.
// Defined by the Game Boy machine's rule, kept: a semitone.
const detuneMost = 100

// envelopeMost is the longest any envelope time is, in seconds. Picked: a
// struck glass rings for well under half a minute.
const envelopeMost = 30

// vibratoRateLeast and vibratoRateMost bound a vibrato, in hertz. The top
// is the Game Boy machine's, kept; the bottom is picked.
const (
	vibratoRateLeast = 0.05
	vibratoRateMost  = 20
)

// glideDefault is how long a slide takes to reach the next note when the
// channel does not say, in seconds, and glideLeast and glideMost its
// range. Picked: sixty milliseconds is about what the bass machines this
// is modelled on take; a slide much longer is a siren.
const (
	glideDefault = 0.06
	glideLeast   = 0.001
	glideMost    = 2
)

// masterDefault is the mix's scale when the voicing gives none, and
// masterLeast and masterMost its range. Picked: a half leaves two full
// channels landing together at full scale.
const (
	masterDefault = 0.5
	masterLeast   = 0.01
	masterMost    = 4
)

// sixtyDecibels is the natural log of a thousand: the exponent that takes
// an amplitude down sixty decibels. Defined.
var sixtyDecibels = math.Log(1000)

// envelopeBlock is how many samples an envelope or a bend holds between
// the moments it is worked out exactly; between them it moves in a
// straight line. Picked: thirty-two samples is under a millisecond.
const envelopeBlock = 32

// gateAttack and gateRelease are the envelope of a channel that gives
// none: full while held, with edges just long enough not to click.
// Picked.
const (
	gateAttack  = 0.003
	gateRelease = 0.02
)

// tailMost is the longest a piece rings past its last bar, in seconds.
// Picked: a long reverb is allowed to finish; nothing is allowed to make
// a piece a minute longer than its score.
const tailMost = 40

// reverbDecayLeast and reverbDecayMost bound a reverb's decay, in
// seconds. Picked: a tenth is a box and twenty is a cistern.
const (
	reverbDecayLeast = 0.1
	reverbDecayMost  = 20
)

// repeatsFeedbackMost is the most of a repeat that is fed back. Picked:
// at 0.95 a repeat takes about 135 turns to fall sixty decibels; at one it
// never does.
const repeatsFeedbackMost = 0.95
