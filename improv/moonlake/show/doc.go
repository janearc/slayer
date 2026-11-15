// Package show is moonlake's reading of its music and its cue sheet,
// shared by the commands that need it: cueshow, which says the lights'
// state at a moment for lightshow; metronome, which flashes the beat;
// and explain, which draws the beat, the cues and the chant over the
// finished film.
//
// It holds what used to sit in cueshow's and metronome's package main:
//
//   - ReadScore reads score.yaml: the measured beat grid, which beats
//     were measured and which carried through a break, and the
//     transcribed tracks (bass, lead, chant).
//   - ReadSheet reads cues.yaml against a score. Every cue's moment is
//     placed on the grid when the sheet is read, so a cue at a bar the
//     score lacks, or naming a figure lightshow does not play, is
//     refused by name then and not half way through a render.
//   - Sheet.At folds every cue up to a moment into a Moment: the rig,
//     beams, palette, strobes, floods, searchlights, mist, master, moon,
//     ship's beam, exposure and fires. Moment.Lines is cueshow's output,
//     key=value, unchanged.
//   - Sheet.Fired is each cue at its moment with the words that say
//     what it sets, as the metronome's band shows them.
//   - Score.BeatAt is the beat a moment is in: its bar, its place in the
//     bar, whether it was measured, and how long since it struck.
//
// Bar 1 beat 1 is the first beat of the grid. A bar is four beats.
package show
