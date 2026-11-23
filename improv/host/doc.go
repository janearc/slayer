// Package host lays a short's host over its frames, in time with its
// music: small at the foot of the picture, drifting and flickering,
// jumping on the kicks, showing the clip that fits the moment, and never
// outside the box the caller gives it, so it never covers what is being
// explained.
//
// It lifts the picture for a frame or two on the kicks, puts
// a full-frame intro clip with a line of words before the short, and
// writes the film with the tune's own sound.
//
// It knows nothing of what the short is about. A caller hands it:
//
//   - the frames, as any function from a frame number to a picture (the
//     walk package's Short.Frame is one);
//   - the tune, as a Tune: its score's beat, its cut sheet's parts, and
//     its own sound, from which the loudness is measured;
//   - the host's clips by name (resting, speaking, angry, flash, quiet),
//     looping films on black, read with interfaces/movie;
//   - when the host is speaking, as spans of seconds;
//   - the box the host may move in, in the frame's pixels.
//
// The pure parts take a moment, what is happening at an instant, and say
// which clip shows, where the host is, how bright it flickers, and how
// much the picture lifts. The motion is seeded, so a frame drawn twice is
// the same frame.
package host
