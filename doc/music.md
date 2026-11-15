# Music

Slayer has its own synth, `cheeptune`: a score and a voicing in, samples
and a WAV out, with a playhead that says the beat and bar at any moment.
The score also provides an exact musical clock for scene cues. Supplied
recordings can instead use an editable measured beat grid. Soundtrack
playback and the scene's simulated time remain separate.

The full formats are in [scoring](../cheeptune/SCORING.md),
[writing for video](../cheeptune/VIDEO.md) and
[designing sounds](../cheeptune/SOUNDS.md). This guide connects them to
jobs, auditioning, synthesis and music-driven pictures.

## Put music under a film

Use the [first-film job](../jobs/first-film.yaml) for a small complete
example. A synthesized soundtrack needs a score and a voicing:

```yaml
sound:
  tune: ../cheeptune/examples/sunrun.json
  voicing: "@leap-dmg"
```

This fragment assumes a job in `jobs/`. File paths are relative to that
job; `@name` selects an embedded item. Discover current names through:

```sh
game run slayer -- list tunes
game run slayer -- list voicings
game run slayer -- list library
game run slayer -- draw jobs/first-film.yaml --dry-run
```

Then draw the job normally. `ffmpeg` must be available for movie output.
A tune marked as looping repeats to fill the film. A nonlooping tune ends
and leaves silence; a longer soundtrack is cut at the film's end. A still
cannot have sound. Choosing a voicing changes synthesis, not the written
notes; machine-specific expressive features can differ as described below.

A supplied recording uses a different sound form:

```yaml
sound:
  file: audio/song.opus
  from: {value: 12, unit: s}
  score: audio/song-score.yaml
```

`from` trims playback from twelve seconds into the original recording.
The score is optional for a plain soundtrack; a played scene needs a
musical clock, estimated if no score is supplied. A kept, checked score
is preferable for repeatable editing. Never put both `tune` and `file`
in the same sound block.

## Choose or write a score

Audition an existing library tune before changing its instruments. Its
cut sheet identifies opening, build, break and reveal moments so picture
edits can land on musical structure. A cut sheet is descriptive metadata;
a scene cue sheet actively changes scene channels. They are separate files.

For a new composition, start from a short score in `cheeptune/examples/`.
Keep every bar's durations equal to its meter and every part's bar count
equal. Change notes first, then voicing, then effects; this makes mistakes
easier to locate. A four-beat bar can be `c4 e4 g4 c5`, or
`(c4 e4 g4):4`, or a rest `-:4`. Chords and percussion stacks are tokens,
not additional parts.

A constant-tempo piece lasts `bars * meter * 60 / tempo` seconds. For
tempo changes, add the duration of each tempo section. The score supports
bar-boundary tempo changes, not gradual ramps. For a video, give the opening
a hook, leave a real break and make an ending that lands. Use `loop: false`
when the ending should stop rather than restart.

Create a cut sheet beside the score as `NAME.cuts.json`; follow
[VIDEO.md](../cheeptune/VIDEO.md) for its exact schema and bar positions.
Use the library checker and piano-roll film to catch malformed bars,
overlong notes, clipping and mismatched voicings, then listen to the result.
Waveform/peak checks cannot decide whether music sounds good.

## A tune

Two JSON documents, with `//` comments allowed and an unknown field an
error. A file with a `parts` key is a score and one with a `channels` key
is a voicing.

A score says `name`, `tempo` (40 to 300), `meter` (1 to 16), optionally
`loop` and `tempos`, and `parts`: roles `lead`, `bass`, `pad` and `perc`,
each once, each with the same number of bars. A bar is a string of tokens
that adds up to the meter exactly.

| Mark | Means |
|---|---|
| `a4`, `c#5`, `bb3` | a note: letter, optional sharp or flat, octave 0 to 9 |
| `-` | a rest |
| `(c4 e4 g4)` | a chord of notes |
| `k`, `s`, `h` | a hit, in a `perc` part; the voicing says what it sounds like |
| `(k h)` | hits struck together, in a `perc` part |
| `:n`, `:n/d` | length in beats; none is one beat |
| `~` | a tie, last token of a bar, into the same pitch |
| `@pp @p @mp @mf @f @ff` | loudness after a note, chord or hit and its length; none is `mf` |
| `>` | after a single note, last: a slide into the next note |
| `tempos: [{"bar": 9, "tempo": 132}]` | the tempo changes at the start of a bar; no gradual change, write one a bar |

The studio plays loudness marks and slides. The Game Boy machine reads
them and plays the score as if they were not there, so every score works
with every voicing. Slides apply only to `fm` and `analog` channels. On the
studio the drum letters `k s h o t b` mean one thing in every voicing.

A library tune also has a cut sheet, `NAME.cuts.json`: group, mood,
feeling, intent, what to listen for, the voicing, tempo, meter, length in
seconds, and its parts with the bar and second each begins at. It is read
and checked against its score.

## Two machines

- The Game Boy machine: four channels (two pulse, one wave, one noise)
  simulated at the register, byte for byte as it was.
- The studio: up to sixteen channels of five kinds (`fm`, `analog`,
  `pluck`, `modal`, `drum`); real chords; a place and a send to a shared
  reverb and a shared delay on each channel; a four-pole ladder filter, an
  accent that opens it, fold and overdrive on the analog voice; struck
  and synthesised drums, stacked hits and metal drums.

## Voicings

`slayer list voicings` says the 15 built in: 8 for the studio (`@acid`,
`@acid-clockwork`, `@analog-drive`, `@glass-bells`, `@glass-struck`,
`@iron-works`, `@night-pad`, `@wire-harp`) and 7 for the Game Boy
(`@crypt-dmg`, `@dmg-echo`, `@dmg-pad`, `@leap-dmg`, `@overdrive-dmg`,
`@tape-chords`, `@tape-echo`).

`slayer list tunes` says the 5 built-in scores. A voicing a library tune
was written for lives in the library beside it. `slayer tunes DIR
--sounds` with voicing names writes a sheet with one note of each instrument.

Eleven candidate voicings for the studio are in `cheeptune/candidates`
with `WANTED.md` files saying what the studio could not yet do. They are
not built in and not listed.

## The library

33 tunes written for videos are compiled in, each named `@NAME`
(`slayer list library`). A library tune may not take a built-in's name.
In code, `OpenTune` and `Library` read it. A job names one:

```yaml
sound: {tune: "@glass-hour", voicing: "@crypt-dmg"}
```

The voicing is the one its cut sheet names. A tune that loops is played
again to the film's end and cut there; one that does not is followed by
silence.

## Auditioning and voting

`slayer tunes DIR` serves, on this machine only (`127.0.0.1:8417`, or the
loopback address `--address` says), a page to hear each tune in DIR and
keep or toss it, with a film of its piano roll with its parts marked
and named as they begin. It stops on control-c.

- A vote is `keep`, `toss` or `clear`, written to `votes.json` in the
  folder. A vote moves nothing on disk.
- A vote needs JSON and this machine's host and port, which shuts out a
  web page posting from a browser. It does not shut out another program on
  the machine: there is no login.
- A tune added while serving can be voted on; a vote for a tune gone from
  the folder can be cleared.
- Instead of serving: `--check` (length, peak, voicing; fails if any will
  not load), `--films` (`NAME.mp4`), `--plates` (`NAME.png`), `--page`
  (`index.html`), `--sounds` with voicing names. They write beside the tunes, or
  to `--out`.

## Play scene channels from the song

Adding a soundtrack alone does not animate lights. A named scene that
exposes channels can also name a `scene.play` cue sheet. The `cuesheet`
engine reads it against the soundtrack's clock. Channel names and valid
ranges belong to the scene; an unknown channel is refused rather than
silently ignored.

A cue can set a held level, fade/ramp it, flash on a rhythm or note,
follow a voice's envelope, or step a rotation/value on each strike. Bar
positions are counted from one; explicit seconds are also supported.
Drum-hit and velocity filters can select which notes trigger a cue.

Read [the cue format](../cuesheet/doc.go) and the
[Moonlake worked show](../improv/moonlake/README.md) before authoring one.
Moonlake's original show tooling and the general cuesheet engine remain
distinct interfaces; use the invocation belonging to the example.

A synthesized piece knows its bars and notes directly. A measured recording
supplies an irregular beat grid through `cuesheet.Measured`; it is not
forced into one tempo. The current generic audio analysis supplies a `beat`
voice, not an instrument transcription.

The playback clock advances at the film's rate. Compressing an hour of
scene motion into thirty seconds does not speed up the song. Audio trimming
keeps cues attached to the original song's bars. When music drives a scene,
its audio, voicing, score and cues participate in picture identity; changing
them can require retracing frames. A soundtrack-only change can reuse
pictures while remaking the movie.

## Supplied audio

For supplied audio, [beats](../beats/README.md) describes editable beat
grids, aligned drum-stem input and the visual-metronome check. A named
scene's `play` sheet can use `sound: {file: song.opus, score: score.yaml}`;
the score is checked against the audio's hash. Without one, the file gets
an explicitly provisional estimate when the scene is opened.

Music follows playback time during a hyperlapse. Keep weather timestamps
and flight progress on their separate scene timeline. A faster ascent
must not speed up the song or its cues.

### Estimate and check a beat grid

```sh
game build
game run beats -- analyse song.opus --out score.yaml
game run beats -- analyse song.opus --drums drums.wav --out stem-score.yaml
```

The optional drum stem must keep the original decode's sample rate and
sample count. The analyser does not perform source separation. It uses
onset energy and a tempo/spacing search; it can select half/double tempo
or the wrong downbeat. Its score stays `status: estimated`. Correct the
editable YAML and check a visual metronome with the original audio before
marking it verified.

The metronome emits raw RGB24 frames, not a movie by itself:

```sh
game run metronome -- draw --score score.yaml --cues cues.yaml --fps 24 --width 640 --height 360
```

Mux those frames with the same original song using the worked workflow
in [Moonlake](../improv/moonlake/README.md). An empty `cues: []` sheet
checks beats alone. Verify downbeat phase, quiet breaks and drift by eye
and ear.

A measured mark records local onset support, not human approval.
The metronome currently refuses meters other than four rather than drawing
incorrect bars. Keep the score's audio hash intact; changed audio bytes
are refused even at the same file path.

## Synthesis from Go

`cheeptune.Open(score, voicing)` returns a rendered `Piece` or an error.
`Piece.Cook(directory)` writes/caches its WAV, and the piece's playhead
provides bar, beat and note timing for an application. `LoopPeriod` is
nonzero only for looping music. `OpenTune(name)` opens a library tune with
its cut sheet. These APIs do not need a GPU or a Slayer world. See
[the package](../cheeptune/README.md) and [design](../cheeptune/DESIGN.md).

## Limits

There is no gradual tempo ramp or automatic instrument transcription.
The two synthesizers have intentionally different voice constraints;
neither is a general sampler or a promise of an acoustic performance.
Peak, spectrum and duration checks establish signal properties. Listening
is still required to judge the composition, balance and mix.
