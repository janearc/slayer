# Cheeptune design

a score contains timed notes and hits for `lead`, `bass`, `pad`
and `perc`. a voicing binds those roles to instruments. the same
score can run on either supported machine when its notes and hits fit
the voicing. [the scoring guide](SCORING.md) describes both formats.

## The machine

the Game Boy model exposes the hardware’s four channels:

- two pulse channels. The duty is 12.5, 25, 50 or 75 per cent, and no
  other value.
- one wave channel. It plays a table of 32 steps, and each step holds 0 to
  15. A triangle is one such table and ships as the built-in `triangle`,
  beside `saw` and `square`. (Two pulses, a triangle and noise is the NES.
  The Game Boy's third channel is a wavetable.)
- one noise channel: a 15-bit or 7-bit shift register clocked at a rate
  set by `shift` and `divisor`, the two numbers the hardware takes.

Pitch is quantised to the machine's 11-bit frequency register, so a high
note sits where the hardware can put it rather than where the maths says.

Pulse and noise volume have 16 levels. The wave channel's volume has four
settings: off, a quarter, a half and full.

Envelopes, vibrato and arps advance once per driver tick, 59.73 times a
second, the rate a Game Boy music driver runs at. A smooth ramp becomes a
staircase, which is how the hardware sounds.

Each channel pans left, right or both, as the hardware allows. The output
passes through a one-pole high-pass filter, as the hardware's output
capacitor does, so a pulse wave carries no DC offset.

## The second machine: the studio

the studio plays the same score with continuous envelopes,
polyphony, effects and dynamics.

- five engines, one a channel, chosen by `kind`: `fm` (two to four sine
  operators), `analog` (band-limited oscillators into a resonant low-pass
  filter), `pluck` (a string by Karplus and Strong's method), `modal` (a
  struck object as a bank of resonators) and `drum`.
- a chord sounds all its notes, each its own voice. A channel has a limit
  on voices; past it the oldest voice already let go is taken.
- up to sixteen channels, and several may play one role, which layers
  them.
- each channel has a place between left and right and a send to one
  shared reverb (a feedback delay network) and one shared delay in beats.
- nothing is simulated a tick at a time: each note is rendered whole at
  the output rate. A loop has its tail folded onto its start.
- a mix that clips is refused, with its peak and the voicing's master.

both machines follow bar-boundary tempo changes. the studio also
uses score dynamics and supports slides on compatible instruments.

## Rendering

`Open(score, voicing)` checks and renders the pair to stereo audio.
a mix that would clip fails with its peak. `Piece.WriteWAV` writes
16-bit stereo WAV. `Piece.Cook(directory)` writes or reuses a named
file keyed by the score, voicing and renderer version. `Open`
still synthesizes the piece in memory.

render ahead of playback. a non-looping piece can have a release tail.
a looping piece carries its release and echo tails across the loop
boundary. `LoopPeriod()` returns the audio period for a loop and
zero for a piece that plays once.

## The playhead

```go
type Playhead interface {
    At(offset time.Duration) (beat, bar, beats, level float64, ok bool)
}
```

`Piece.At` reads time since the first audio sample. the caller
supplies that clock, including any playback latency correction.

- `beat` is the fractional position within the current beat.
- `bar` is the fractional position within the current bar.
- `beats` is the cumulative beat count using the score’s tempo
  changes. it continues across a loop seam.
- `level` is stereo RMS over a centred 50 ms window, normalized
  to the loudest measured window in the piece.
- `ok` is false before the first sample and after non-looping audio
  ends. the other values are zero in those cases.

Slayer uses this clock for [music-driven scenes](../doc/music.md).
a separate audio player supplies its own start time.

## Judging a keypress

for a rhythm game, compare a key press with the nearest beat on
the audio clock.

**Subtract the latency first.** `At` measures from the first sample; a
player hears the sound later and presses later still. That delay is the
speaker, the buffer, the room and the person, and none of it can be worked
out from here.

Measure it once with a click the player taps along to -- compose's `tap`
verb writes it to a file -- and subtract it from the offset before asking
where the beat is. Without that, a game judges its own machine.

**Distance is to the nearest beat, not the last one.** `beat` runs from
nought up to one, so a press a hair early reads as 0.98 rather than as
-0.02. A press is `beat` late if `beat` is below a half, and `1 - beat`
early otherwise:

```go
_, _, _, _, ok := p.At(pressedAt - latency)
off := beat
if off > 0.5 {
	off = off - 1 // nearer the next beat than the last
}
missedBy := math.Abs(off) * p.Score.BeatSeconds()
```

Taking the remainder without that fold makes every early press look like the
latest possible press, which is the most forgiving judgement turned into the
harshest one.

**A window is in seconds, not in beats.** A tenth of a beat is 52 ms at 115
and 35 ms at 172, and a player's hand does not get quicker because the tune
did. Pick the window in milliseconds and divide.

## Inspect a piece

`cheeptune.Check` reports the documents’ roles, length and compatible
channel assignments. `Piece.WriteRoll(path)` writes a PNG with
pitch, hits, waveform and level on the same time axis.

run `slayer tunes DIR --check` to load and synthesize the named tune
pairs in a library folder. use `slayer tunes DIR --plates` for their
piano-roll images.
