# tune

tune turns two documents into sound for a terminal game. The score says
what is played, and the voicing says what plays it.

It renders the pair to a WAV before the game starts, and draws the result
as a picture, so a person or an agent can check a tune without hearing it.
While the game runs it answers one question: where in the music a moment
falls.

The name is provisional.

## The seam

A score is a stencil and a voicing is a colourway. This is the same move
miami makes with sprites.

A score holds roles: `lead`, `bass`, `pad`, `perc`. Each role has bars of
notes, and a score never names a sound. A voicing binds each role to a
channel on one machine and says what that channel sounds like.

The same score played through two voicings is the same tune on two
machines. Because the score never changes, a test on the score still holds
when the sound changes.

Echo is not a role in the score. It is something a voicing does: a channel
that plays another role's notes later and quieter. That is how chip music
fakes reverb, and it decides nothing about the tune.

## The machine

The first machine is the Game Boy. The limit is the point, so tune holds it
and refuses anything past it:

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

A second machine is a new `machine` value with its own channel set. It
needs no change to the score. In the code a machine is one type with five
methods (its kinds, its rules on a voicing, whether a channel can sound an
event, which kinds play hits, and how a plan becomes samples) and one line
in the table in `machines.go`.

## The second machine: the studio

The Game Boy is thin for four reasons: one family of timbres, one voice a
channel so a chord is one voice flicking between notes, no room, and every
note as loud as the next. The studio takes each of those away and keeps
the score.

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

The instrument does not go on the score's part. The voicing already is
that split, and it is why one score can be heard through many sounds.

Nobody who wrote the studio could hear it. Every claim about it is a
measurement in the tests, and every number in it says whether it was
defined, derived or picked. `SOUNDS.md` teaches designing an instrument.

A score may mark a note's loudness (`c5@p`) and may change tempo at a bar
(`tempos`). The studio plays the marks; the Game Boy plays a marked score
exactly as an unmarked one. Both machines follow the tempo.

## The score

```
{
  // one comment style, stripped before decoding
  "name": "dusk",
  "tempo": 92,        // beats a minute; one tempo for the whole piece
  "meter": 4,         // beats in a bar
  "loop": true,
  "parts": [
    {"role": "lead", "bars": ["e5:2 d5 b4", "c5:4"]},
    {"role": "perc", "bars": ["k h s h", "k h s h"]}
  ]
}
```

A bar is a string of tokens separated by spaces:

- `a4` is a note in scientific pitch, where A4 is 440 Hz. The letter is
  lowercase, then an optional `#` or `b`, then the octave: `c#5`, `bb3`.
- `-` is a rest.
- `(c4 e4 g4 b4)` is a chord. One channel can only play one note at a
  time, so the channel cycles through the chord at its `arp` rate. That is
  how chip music plays sevenths and ninths.
- a bare lowercase word such as `k` or `hat` is a percussion hit. The
  voicing says what it sounds like.
- `:n` or `:n/d` after any token is its length in beats. With no suffix it
  is one beat.
- `~` at the end of a bar's last note ties it to the first token of the
  next bar, which has to be the same pitch.

Lengths are exact fractions. Every bar has to add up to `meter` exactly,
and every part has to have the same number of bars. A bar that does not is
refused, and the error names the part, the bar and what the bar held.

Everything is checked, and nothing is quietly ignored. An unknown field is
an error. So is an unknown role, a note outside the machine's range, or a
tie to a different pitch. Every problem in a document is reported at once.

## The voicing

```
{
  "name": "dmg-echo",
  "machine": "gameboy",
  "channels": [
    {"kind": "pulse", "role": "lead", "duty": 0.25, "volume": 0.9,
     "envelope": {"attack": 0.01, "decay": 0.4, "sustain": 0.55,
                  "release": 0.3},
     "vibrato": {"rate": 5, "depth": 10, "delay": 0.3}},
    {"kind": "pulse", "echo": "lead", "delay": "3/4", "gain": 0.35,
     "detune": -7, "duty": 0.5, "pan": "right"},
    {"kind": "wave", "role": "bass", "table": "triangle"},
    {"kind": "noise", "role": "perc",
     "hits": {"k": {"shift": 7, "divisor": 3, "decay": 0.12}}}
  ],
  "drop": ["pad"]
}
```

A channel plays a role or echoes one, and never both. An echo needs
`delay`, in beats, and `gain`, which scales the echoed notes' volume.
`volume` is for any channel, 0 to 1. `detune` is in cents.

A score role that no channel plays has to be listed in `drop`. Otherwise
the pair is refused, because a silent part is almost always a mistake. A
pitched note sent to the noise channel is refused, and so is a hit sent to
a pitched channel.

A field on the wrong kind of channel is refused rather than ignored: a
duty on a wave, hits on a pulse, an envelope on noise.

- `envelope` takes attack, decay and release in seconds, and sustain from
  0 to 1. With no envelope a note is full while it lasts and silent after.
- `vibrato` takes rate in hertz, depth in cents, and a delay in seconds
  before it starts.
- `arp` is a number or a string. A number is driver ticks a step, where a
  chord shimmers. A string is beats a step, `"1/8"`, where a chord is
  heard as broken. The default is 3 ticks.
- a hit takes `shift` (0 to 13), `divisor` (0 to 7), `short` for the
  7-bit register, `volume`, and `decay` in seconds to silence.

The envelope is built on a scalar ramp that belongs to this package:
position in, number out. libdaffy's ramp maps a number to a colour.

Giving a colour library a scalar type is a contract decision, and
it has not been made. If a second user of a scalar ramp turns up, that is
the time to move it.

## Rendering

Rendering happens ahead of time and never in the frame loop. The frame loop
runs at 12 to 80 frames a second, and audio needs 44,100 samples a second
with no missed deadline.

The machine runs at four times the output rate with plain, unfiltered
waveforms, as the hardware does. The result is low-pass filtered at 18 kHz
and decimated to 44,100 Hz. What folds back lands above 18 kHz. Output is
16-bit stereo WAV.

A looping piece is rendered for two passes and the second pass is kept. An
echo or release that runs past the end of the first pass is already
sounding at the start of the second, cut off where the hardware would cut
it. The envelopes and echoes at the seam are the ones the middle has.

A looping piece is not periodic in its own length. The oscillators and the
driver's ticks run on in absolute time, so the last sample of a pass does not
lead into its first. Both ends of a looping piece therefore fade over 5 ms.

The fade turns that jump into a 10 ms dip, which nobody has listened for
yet. It also covers afplay, where each pass is its own process and passes
meet within a latency's jitter. A player that loops to the sample would
want a crossfade into a third pass instead.

The fade is in the samples, so `level` dips with it: about a fifth, for
one level window, once a loop. A scene that pulses on `level` pulses down
there for a frame or two. That is what plays, and it is left that way.

The kept pass also opens with the tails of the loop's own last notes, so
the very first play begins with a faint echo of an ending it never played.
Every later pass is right. A piece that minds can get an intro pass later.

`LoopPeriod` is the length of one pass, read from the samples, and nought
for a piece that does not loop. A player can hand it on as a restart period
without looping a piece written to play once.

The mix has fixed headroom: four channels at full volume sum to full scale.
A rendered piece reports its peak. If any sample clips, rendering fails and
names the peak. It never clips quietly.

## The playhead

```
type Playhead interface {
    At(offset time.Duration) (beat, bar, beats, level float64, ok bool)
}
```

A rendered piece implements this. The offset is measured from the first
sample. tune owns no clock: whoever started the audio subtracts start time
and latency. miami's app declares the same interface, and neither package
imports the other.

- `beat` is the position inside the current beat, from 0 up to but not
  including 1.
- `bar` is the position inside the current bar, on the same scale.
- `beats` counts beats since the first sample, fraction included, and only
  grows. In a looping piece it keeps growing across the loop seam, and `ok`
  stays true.
- `level` is the loudness of the mix: the RMS of both output channels over
  a 50 ms window centred on the offset, divided by the loudest such window
  in the piece. It reads 25 ms ahead, which is fine: the audio exists.
- `ok` is false before offset 0, and past the end of a piece that does not
  loop. When `ok` is false, all four numbers are 0.

A piece has one tempo, which is why `beats` is the offset divided by the
beat length. A score with tempo changes would have to integrate over them,
and scores cannot express those yet.

The clock is an estimate. macOS `afplay` cannot say where its playhead is,
so the caller times from when it started the process, and the startup
latency has not been measured.

That is accurate enough for a ball bouncing on the beat, and not for
anything that has to line up to the sample.

A note sounds from the first driver tick at or after its beat, up to 16.7 ms
late. The playhead counts beats exactly and does not follow the ticks, so a
visual on the beat leads the note by that fixed, repeating amount.

## Music in a game

A game needs four things from this package, and they happen at four
different times. Keeping them apart is what stops music being a thing that
has to be rewritten when the game gets bigger.

**Compose.** A score and a voicing, as documents, in the game's own
repository beside its scenes. Or a built-in: the scores and voicings that
travel with this package are named with "@", and a game that wants music
tonight starts with `@crypt` and `@crypt-dmg` and writes its own later.

**Check.** `tune check` says what the pair holds and lists every problem,
and `tune roll` draws what actually played. Neither needs ears, which is the
point: a person who cannot hear the machine, and an agent that cannot hear
at all, both work from the same picture.

`tune pairs` renders every built-in pair and prints each one's peak, or why
it was refused. A voicing is not a hat that fits any tune, and that is worth
reading rather than finding one peak at a time.

**Ship.** `tune cook` renders the pair to a WAV named for its own contents
and the machine's version. A game either cooks at build time and ships the
file, or calls Open at startup and Cook into a cache directory it keeps
between runs.

Opening costs one render: about four tenths of a second for twenty seconds
of music on this machine.

**Play.** The piece that made the sound is the playhead that says where the
sound is, so there is nothing to keep in step:

```go
p, err := tune.Open("@crypt", "@crypt-dmg")
wav, err := p.Cook(cacheDir)
stop, err := stage.Score(wav, p, p.LoopPeriod())
defer stop()
```

`stage.Score` is miami's. It starts the sound, takes the start time, and
supplies `@beat`, `@bar`, `@beats` and `@level` to the scene document, which
binds them like any other number. Nothing in the scene knows what made them,
and nothing here knows what they draw.

A looping piece needs `LoopPeriod`, because afplay plays a file once and
miami restarts it on that period. A piece that plays once reports nought,
which is what "do not repeat it" looks like.

**What a game must not do.** Render in the frame loop. The loop is
build-bound at 12 to 80 frames a second and audio wants 44,100 samples a
second with no missed deadline; those are different clocks and only one of
them forgives a late answer.

## Judging a keypress

A rhythm game asks whether a press landed on the beat. The playhead answers
it, and there are exactly two subtleties, both of which bite silently.

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

## Seeing it

An agent cannot hear a WAV. So every tune can be inspected in two ways.

`tune check SCORE [VOICING]` says what the documents hold: name, tempo,
meter, bars, length in seconds, and each part's notes, chords, range and
hits. Given a voicing it prints each channel and what it does, and whether
the pair fits.

Every problem is listed, one to a line. The exit status is 0 when the
documents are clean and 1 when they are not.

`tune roll SCORE VOICING` writes a PNG with pitch against time, drawn from
what the driver wrote on every tick. Each role has its own colour, an echo
is its role's colour faint, and hits have a lane each.

Underneath are the rendered waveform and the level the playhead reports,
on the same time axis, with bar lines through all of them. The picture
shows what plays, not what the score intended.

`check` identifies a document by its top-level keys: `parts` is a score and
`channels` is a voicing. It never searches the text for a keyword.

## Reading a green run

Two tests here said more than their bodies checked, and both were green the
whole time they did it: one read one kind of document where its name
promised both, and one was named for the opposite of its own arithmetic.

The rule that came out of that is written once, in compose's DESIGN.md,
under "A mutation that does not bite". It says what it means when you break
the code on purpose and the test stays green, which is a question this
package raises as often as that bench does.

It is not repeated here. This package does not know compose exists and does
not need to, but a rule kept in two places drifts in one of them, and that
is the fault the pointer avoids.

## Not decided

- Whether the music reacts to play or only plays under it. This package
  renders pieces either way. Reacting would add a scheduler that joins
  short loops at bar lines. That scheduler would be a second Playhead, and
  nothing above would change.
- Where the scalar ramp lives. See the voicing section.
- The name.
- What the pointer above resolves to if this package is ever published.
  "Reading a green run" aims at compose's DESIGN.md, and compose has a road
  and no dist. Publishing tune without settling this ships a reference to a
  document the reader cannot reach.

  This shape turned up in miami, which is published and does point out
  of itself. The same fault is latent here rather than absent. It wants
  deciding when the name is, not at the moment of the tag.
