// Package cuesheet plays a scene's channels from music: a cue sheet, read
// against a clock that knows the music's bars and notes, gives the value of
// every channel at a moment.
//
// It is the small engine zappy needs, written in the vocabulary moonlake's
// cue sheet settled on (improv/moonlake/cues.yaml), so the two can become
// one engine later. It imports nothing of moonlake's.
//
// # The clock
//
// A Clock says how many beats a bar holds and how many bars the music has,
// how many beats from the first a moment is and the moment of a count of
// beats, and the notes each voice plays. Two are given: a cheeptune piece
// (Piece), whose voices are its score's roles, and a plain fixed tempo
// (Fixed), for tests and for a supplied song whose tempo is known. The
// clock is an interface so moonlake's measured score can be a third.
//
// # The sheet
//
// A sheet is YAML: a list of cues. Every cue says when, by at, "bar.beat"
// counted from one (or "bar" for its first beat), or by at_sec, seconds
// from the music's start. A cue changes only what it names.
//
//   - A channel's name with a number is a level, held from the cue's
//     moment. With fade, seconds, it glides there from what it held, as
//     moonlake's fade does; with until, "bar.beat", it ramps there
//     linearly, reaching it at until and holding after.
//   - flash, a map of channel to peak, makes those channels flash: on each
//     x of pulse, sixteen steps to a bar as moonlake writes it, or on each
//     note of the voice on names (only hit's notes, when hit is given),
//     the channel jumps to its peak and falls back toward its level,
//     to a third (1/e) of the way in decay beats.
//   - follow, a voice, and up, a map of channel to value: those channels
//     stand at their up value while a note of that voice sounds (only
//     hit's, when hit is given) and at their level between. With attack
//     and release, beats, a note swells them up over its attack and lets
//     them fall over its release after it ends, so they breathe.
//   - step, a map of channel to amount, moves those channels on by the
//     amount at each strike, struck as a flash is (pulse, or on and hit),
//     and they stay where the strikes leave them: a tumble a notch a tick.
//   - quiet, a list of channels, ends their flash, follow and step.
//
// Wherever a cue names a voice's notes, hit keeps only a drum's named
// hit, hot only notes written at least so loud and soft only notes
// written quieter than so, as gains: 1 is mf, 1.58 ff, 0.4 p.
//
// A channel the sheet names that the scene has not is refused by name,
// listing those it has; so is a value outside a channel's range, with
// the range; so is a bar past the music's end, and a beat past the bar's.
//
// # Reading a moment
//
// At gives every channel's value at a moment: its level from the cues up
// to then, raised to its follow value while its voice sounds, and the
// latest flash's remainder over that. A channel no cue names stays at
// its rest.
package cuesheet
