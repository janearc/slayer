// Package cheeptune renders a score through a voicing to sound, ahead of
// time, and answers where in that sound a moment falls.
//
// A score holds roles and never names a sound. A voicing binds each role to
// a channel on one machine and says what the channel sounds like. The same
// score through two voicings is the same tune on two machines. DESIGN.md is
// the contract; this is the working summary.
//
//	p, _ := cheeptune.Open("@crypt", "@crypt-dmg")
//	err := p.WriteWAV("crypt.wav")
//	beat, bar, beats, level, ok := p.At(offset)
//
// Open reads a path, or a built-in named with "@", and runs the machine.
// The piece holds the samples (Rate, Left and Right, in minus one to one),
// Duration and LoopPeriod.
//
// The piece is also the playhead: At says which beat and bar an offset from
// the first sample falls in, and how loud it is. Check reports every
// problem in a document at once, and Builtin lists what is compiled in.
// SCORING.md is the page a small model reads to write a score.
//
// There are two machines. The Game Boy (gameboy.go, machine.go, ramp.go
// and the sample-level part of render.go) is four channels simulated at
// the register. The studio (studio.go and the files it names) has five
// engines, real chords, a reverb and a delay. machines.go is the seam
// between them, and SOUNDS.md teaches designing a studio instrument.
//
// Both are private to this package and know nothing of documents; the
// piece renderer is their only caller. A separate tone library, called cheep,
// is reserved for a later split and nothing here fills that name.
//
// This came from the unpublished library tune, main at commit 5a0d353,
// brought in on 2026-11-12. Left behind: cmd/tune, which played through
// afplay and which Slayer has no use for, and tune's go.mod and go.sum.
// WAV playback is the caller's: the clock is whoever started the audio.
package cheeptune
