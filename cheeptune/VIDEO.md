# writing a tune for a video

this page is for a small language model that has read `SCORING.md` and can
write a valid score. it adds what a tune for a video needs: parts a cut can
land on, and a sheet that says where they are. read `SCORING.md` first.

## what a tune for a video is

a short video has a shape: something to hook the first second, a build, a
moment where something is revealed, and an end that lands. the music has the
same shape, with its changes on bar lines where a cut can go.

- **length:** 30 to 90 seconds. most are 30 to 60.
- **parts:** each starts on a bar line and sounds different from the one
  before it: a change of density, register, rhythm or which voices play. a
  tune has at least two parts, one obvious change, and one obvious break or
  stop where a cut or a reveal can land.
- **a clear tempo.** a steady beat is easiest to cut against. the synth also
  accepts tempo changes at bar boundaries through `tempos`; it does not
  interpolate gradual ramps. a change of energy can instead use shorter
  note lengths over the same beat.
- **an ending that stops.** end on a held chord or a clean stop, with
  `"loop": false`. a fade into a loop does not land.
- **a mood,** in a few plain words, and the sentences a person reads: see the
  cut sheet below.

a tune of 17 to 30 bars is about right: at 120 a minute in 4/4 a bar is two
seconds, and the length is bars times meter times 60 divided by tempo.

## the kinds of part

| part | what it does | how to write it |
| --- | --- | --- |
| opening | hooks and sets the mood | one or two voices; the drum rests or is a single kick a bar |
| build | the energy rises | add a voice each four bars; shorten the notes: whole, half, quarter, eighth |
| break | near silence or one voice alone | rests in `pad` and `perc`, a held bass, or a lone lead |
| reveal, or drop | everything arrives | all four voices at once, on a bar's first beat |
| ending | lands | a held chord, one kick, then nothing |

## how to make each change with four voices

- **drop the drums for a break.** the `perc` part is `-:4` for as many bars
  as the break lasts. the bass and the lead go on, so the break is felt as a
  gap and not as a stop.
- **bring the bass in an octave lower for a reveal.** the same notes written
  one octave down: `a2` becomes `a1`. the wave channel plays `c1` to `c7`,
  and a low bass under a lead that has stayed in place sounds like the floor
  opening.
- **double the lead's rhythm for a lift.** if the lead was four quarters, it
  is now eight eighths over the same chords. the tempo is unchanged and the
  tune sounds twice as busy.
- **a snare roll and a climbing arpeggio into a drop.** in the last bar
  before the drop, `perc` is sixteen quarter-of-a-beat snares
  (`s:1/4` sixteen times) and the lead climbs in sixteenths up the scale.
- **a bar of silence before a hit.** all four parts are `-:4` for one bar,
  then the next bar begins with a kick and the lead. a bar that is nearly
  silent works too: one beat of rest in the lead and the drum still rolling.
- **a register change.** the lead plays the same shape an octave higher in
  the last part, and a viewer hears it as the tune lifting.
- **a stop.** the last bar is a held chord in `pad`, the tonic held in the
  lead, the root held in the bass and one `k` in `perc`: `k -:3`.

## the cut sheet

a cut sheet is a second file beside the score, `NAME.cuts.json`. it is read
by `cheeptune.ReadCuts` and checked against the score. an error names the
part and what to change; the checker tells you every problem at once.

```
{
	"group": "sombre and slow",
	"mood": "sombre and slow",
	"feeling": "A quiet room after bad news.",
	"intent": "A slow, wordless moment: an empty house, a letter.",
	"listen_for": "The drum that starts like a heartbeat at 0:11.",
	"voicing": "@tape-chords",
	"tempo": 64,
	"meter": 4,
	"seconds": 41.25,
	"parts": [
		{"name": "embers", "bar": 1, "second": 0.0,
		 "change": "one voice and a held chord, no drums",
		 "feels": "still and alone: nobody speaks"}
	]
}
```

| field | rule |
| --- | --- |
| `group` | the family of tunes it is listed with, one to five words |
| `mood` | two to five words |
| `feeling` | one sentence: what a viewer should feel |
| `intent` | one sentence: what kind of video or moment it is for |
| `listen_for` | one sentence naming the one thing to notice, with its time |
| `voicing` | a built-in voicing, written with the `@` |
| `tempo`, `meter` | the same as the score's |
| `seconds` | the score's length, to two decimals: bars times meter times 60 divided by tempo |
| `parts` | at least two (the sample above shows one to be short). each has `name`, `bar`, `second`, `change` and `feels` |

in a part, `bar` counts from one and the first part is bar 1. bars go up
from one part to the next, and `name` is used once. `second` is the bar's
start, `(bar - 1) times meter times 60 divided by tempo`, to two decimals
(the reader accepts a hundredth of a second either way).

`change` is one line
on what changes there. `feels` is a short phrase in feeling terms, not in
theory, at most ten words: "the floor drops out: tension", "everything
arrives: release", "one voice left alone: waiting".

write the sheet before the notes. say what the tune is for, then write it,
then render it and look at the piano roll (`Piece.Roll()`): if the roll does
not show the change the sheet claims, change the tune or change the words.

`ReadCuts(path, score)` returns the sheet; `Spans()` gives each part's name,
bar, start, end, change and feels, with times as durations.

## the library

`cheeptune/library/` holds twenty-three tunes written to this page, each with a
sheet. they are compiled in. `@NAME` names one as a score, and
`cheeptune.OpenTune("NAME")` renders it with the voicing its sheet names and
returns the sheet as well. `cheeptune.Library()` lists them.

## two worked examples

the files are checked by tests: each is rendered with the voicing in its
sheet, and its sheet is read against it.

### slow and somber, then a boss fight: ember-rising

the file is `examples/ember-rising.json`. this began as a first draft that
was valid and not tuned by ear. what was changed: the sheet was written, and
the last bar before the drop now has a beat of rest in the lead, so the drop
lands out of a gap and not into a wall; the opening comment says the plan.
the notes are otherwise the draft's. 132 a minute, a minor.

```
{
	// slow and somber, then a boss fight. a minor, 132 a minute. four bars
	// of long notes (the village), four of falling chords (the doubt), two
	// of rising low notes and a snare roll (the build, with a beat of
	// silence before the drop), eight of eighth notes (the fight), four
	// more with the lead higher (the last stand) and a held chord.
	// twenty-three bars, about 42 seconds.
	"name": "ember-rising",
	"tempo": 132,
	"meter": 4,
	"loop": false,
	"parts": [
		{"role": "lead", "bars": [
			"e5:4",
			"c5:2 a4:2",
			"c5:4",
			"b4:4",
			"a4:4",
			"c5:2 f5:2",
			"e5:4~",
			"e5:2 d5:2",
			"e4:1/2 g#4:1/2 b4:1/2 e5:1/2 g#4:1/2 b4:1/2 e5:1/2 g#5:1/2",
			"b4:1/2 e5:1/2 g#5:1/2 b5:1/2 e5:1/4 g#5:1/4 b5:1/4 g#5:1/4 -:1",
			"a4:1/2 c5:1/2 e5:1/2 a5:1/2 g5:1/2 e5:1/2 c5:1/2 e5:1/2",
			"a5:1 g5:1/2 e5:1/2 a5:1 g5:1/2 e5:1/2",
			"f5:1/2 a5:1/2 f5:1/2 c5:1/2 f5:1/2 a5:1/2 f5:1/2 c5:1/2",
			"g5:1 b5:1/2 g5:1/2 d5:1/2 g5:1/2 b5:1",
			"a4:1/4 c5:1/4 e5:1/4 a5:1/4 e5:1/4 c5:1/4 e5:1/4 a5:1/4 g5:1/2 e5:1/2 g5:1/2 a5:1/2",
			"a5:1/2 a5:1/2 g5:1/2 e5:1/2 d5:1/2 e5:1/2 g5:1/2 a5:1/2",
			"c5:1/2 f5:1/2 a5:1/2 f5:1/2 a5:1 f5:1",
			"g#5:1/2 b5:1/2 g#5:1/2 e5:1/2 b4:1/2 e5:1/2 g#5:1/2 b5:1/2",
			"d5:1/2 f5:1/2 a5:1/2 f5:1/2 d5:1/2 f5:1/2 a5:1/2 f5:1/2",
			"f5:1/2 a5:1/2 f5:1/2 a5:1/2 g5:1/2 a5:1/2 f5:1/2 c5:1/2",
			"g5:1/2 b5:1/2 d5:1/2 g5:1/2 b5:1/2 g5:1/2 d5:1/2 b4:1/2",
			"e5:1/2 g#5:1/2 b5:1/2 g#5:1/2 b5:1/4 g#5:1/4 b5:1/4 g#5:1/4 b5:1",
			"a5:4"
		]},
		{"role": "pad", "bars": [
			"(a3 c4 e4):4",
			"(a3 c4 e4):4",
			"(a3 c4 e4):4",
			"(a3 c4 e4):4",
			"(f3 a3 c4):4",
			"(f3 a3 c4):4",
			"(f3 a3 c4):4",
			"(f3 a3 c4):4",
			"(e3 g#3 b3):4",
			"(e4 g#4 b4):4",
			"(a3 c4 e4):2 (a3 c4 e4):2",
			"(a3 c4 e4):2 (a3 c4 e4):2",
			"(f3 a3 c4):2 (f3 a3 c4):2",
			"(g3 b3 d4):2 (g3 b3 d4):2",
			"(a3 c4 e4):2 (a3 c4 e4):2",
			"(a3 c4 e4):2 (a3 c4 e4):2",
			"(f3 a3 c4):2 (f3 a3 c4):2",
			"(e3 g#3 b3):2 (e3 g#3 b3):2",
			"(d4 f4 a4):2 (d4 f4 a4):2",
			"(f3 a3 c4):2 (f3 a3 c4):2",
			"(g3 b3 d4):2 (g3 b3 d4):2",
			"(e3 g#3 b3):2 (e3 g#3 b3):2",
			"(a3 c4 e4):4"
		]},
		{"role": "bass", "bars": [
			"a2:4~",
			"a2:2 e3:2",
			"a2:4~",
			"a2:2 e3:2",
			"f2:4~",
			"f2:2 c3:2",
			"f2:4~",
			"f2:2 c3:2",
			"e2:1 e2:1 e2:1 e2:1",
			"e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2",
			"a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 e3:1/2 a2:1/2",
			"a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 e3:1/2 a2:1/2",
			"f2:1/2 f2:1/2 f2:1/2 f2:1/2 f2:1/2 f2:1/2 c3:1/2 f2:1/2",
			"g2:1/2 g2:1/2 g2:1/2 g2:1/2 g2:1/2 g2:1/2 d3:1/2 g2:1/2",
			"a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 e3:1/2 a2:1/2",
			"a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 a2:1/2 e3:1/2 a2:1/2",
			"f2:1/2 f2:1/2 f2:1/2 f2:1/2 f2:1/2 f2:1/2 c3:1/2 f2:1/2",
			"e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 b2:1/2 e2:1/2",
			"d3:1/2 d3:1/2 d3:1/2 d3:1/2 d3:1/2 d3:1/2 a2:1/2 d3:1/2",
			"f2:1/2 f2:1/2 f2:1/2 f2:1/2 f2:1/2 f2:1/2 c3:1/2 f2:1/2",
			"g2:1/2 g2:1/2 g2:1/2 g2:1/2 g2:1/2 g2:1/2 d3:1/2 g2:1/2",
			"e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 e2:1/2 b2:1/2 e2:1/2",
			"a2:4"
		]},
		{"role": "perc", "bars": [
			"k:2 -:2",
			"s:2 -:2",
			"k:2 -:2",
			"s:2 -:2",
			"k:2 -:2",
			"s:2 -:2",
			"k:2 -:2",
			"s:2 -:2",
			"k:1 s:1 k:1 s:1",
			"s:1/2 s:1/2 s:1/2 s:1/2 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 s:1/4 s:1/4 s:1/4 s:1/4 s:1/2 s:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 s:1/4 s:1/4 s:1/4 s:1/4 s:1/2 s:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 s:1/4 s:1/4 s:1/4 s:1/4 s:1/2 s:1/2",
			"k:4"
		]}
	]
}
```

its sheet, `examples/ember-rising.cuts.json`:

```
{
	"group": "slow start, fast finish",
	"mood": "village, then boss fight",
	"feeling": "Quiet grief in a small place, then a hard turn into a fight you can win.",
	"intent": "A video that starts somber and slow, then cuts to action: a story told in two halves.",
	"listen_for": "The snare roll, the beat where the lead stops just before 0:18, and the fight crashing in after it.",
	"voicing": "@leap-dmg",
	"tempo": 132,
	"meter": 4,
	"seconds": 41.82,
	"parts": [
		{"name": "embers", "bar": 1, "second": 0.0,
		 "change": "long single notes over held chords, a kick every other bar",
		 "feels": "still and alone: a village at dusk"},
		{"name": "doubt", "bar": 5, "second": 7.27,
		 "change": "the chords fall to f; the lead is still slow",
		 "feels": "heavier: something is wrong"},
		{"name": "build", "bar": 9, "second": 14.55,
		 "change": "the bass rises in eighths, a snare roll, then one beat of silence",
		 "feels": "the floor drops out: tension"},
		{"name": "fight", "bar": 11, "second": 18.18,
		 "change": "everything arrives: eighth-note lead and bass, a full drum",
		 "feels": "everything arrives: release"},
		{"name": "last-stand", "bar": 19, "second": 32.73,
		 "change": "the lead climbs higher; the drum fills the last bar of each four",
		 "feels": "all or nothing: the last push"},
		{"name": "end", "bar": 23, "second": 40.0,
		 "change": "one held chord and a kick",
		 "feels": "it is over: stillness"}
	]
}
```

### eerie and brittle, in four: glass-hour

the file is `library/glass-hour.json`. four beats a bar, 66 a minute,
fourteen bars. its parts differ by how much is played: thin glassy stabs
over a held bass; a ticking hat and a nervous lead as a line spreads; a
break, a flurry of notes and a snare roll; and then sparse again, one low
bass and a lead, cold. the voicing is `@crypt-dmg`.

```
{
	// eerie, stiff and brittle. f sharp phrygian, 66 a minute, @crypt-dmg. short glassy stabs on odd beats, a bass that holds a note then drops a half step, hats that skip; a sudden big wash then sparse again.
	"name": "glass-hour",
	"tempo": 66,
	"meter": 4,
	"loop": false,
	"parts": [
		{"role": "lead", "bars": [
			"-:2 c#5:1/2 f#5:1/2 c#5",
			"- d5:1/2 g5:1/2 d5:2",
			"-:2 c#5:1/2 f#5:1/2 c#5",
			"- b4:1/2 e5:1/2 b4:2",
			"c#5:1/2 e5 g5 c#5:1/2 e5",
			"- d5:1/2 f#5:1/2 a5:2",
			"b4:1/2 d5 f#5 b4:1/2 d5",
			"-:4",
			"a5:1/2 c#6:1/2 b5:1/2 a5:1/2 f#5:1/2 e5:1/2 f#5:1/2 g5:1/2",
			"g5:1/4 a5:1/4 f#5:1/4 a5:1/4 f#5:1/4 d5:1/4 b4:1/4 c#5:1/4 c#5:1/4 d5:1/4 c#5:1/4 e5:1/4 d5:1/4 e5:1/4 f#5:1/4 a5:1/4",
			"- f#4:1/2 f#4:1/2 f#4:2",
			"- f#4:1/2 f#4:1/2 f#4:2",
			"- g4:1/2 g4:1/2 g4:2",
			"- f#4:1/2 f#4:1/2 f#4:2"
		]},
		{"role": "pad", "bars": [
			"-:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2",
			"-:1/2 (g2 d3):1/2 -:1/2 (g2 d3):1/2 -:1/2 (g2 d3):1/2 -:1/2 (g2 d3):1/2",
			"-:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2",
			"-:1/2 (c3 g3):1/2 -:1/2 (c3 g3):1/2 -:1/2 (c3 g3):1/2 -:1/2 (c3 g3):1/2",
			"-:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2 -:1/2 (f#3 c#4):1/2",
			"-:1/2 (g2 d3):1/2 -:1/2 (g2 d3):1/2 -:1/2 (g2 d3):1/2 -:1/2 (g2 d3):1/2",
			"-:1/2 (c3 g3):1/2 -:1/2 (c3 g3):1/2 -:1/2 (c3 g3):1/2 -:1/2 (c3 g3):1/2",
			"-:4",
			"-:1/2 (f#3 a3 c#4):1/2 -:1/2 (f#3 a3 c#4):1/2 -:1/2 (f#3 a3 c#4):1/2 -:1/2 (f#3 a3 c#4):1/2",
			"-:1/2 (g2 b2 d3):1/2 -:1/2 (g2 b2 d3):1/2 -:1/2 (g2 b2 d3):1/2 -:1/2 (g2 b2 d3):1/2",
			"-:4",
			"-:4",
			"-:4",
			"-:4"
		]},
		{"role": "bass", "bars": [
			"f#2:2 -:2",
			"g2:2 -:2",
			"f#2:2 -:2",
			"c2:2 -:2",
			"f#2 f#2 f#2 f#2",
			"g2 g2 g2 g2",
			"c2 c2 c2 c2",
			"-:4",
			"f#2:1/4 f#2:1/4 f#3:1/4 f#2:1/4 f#2:1/4 f#2:1/4 f#3:1/4 f#2:1/4 f#2:1/4 f#2:1/4 f#3:1/4 f#2:1/4 f#2:1/4 f#2:1/4 f#3:1/4 f#2:1/4",
			"g2:1/4 g2:1/4 g3:1/4 g2:1/4 g2:1/4 g2:1/4 g3:1/4 g2:1/4 g2:1/4 g2:1/4 g3:1/4 g2:1/4 g2:1/4 g2:1/4 g3:1/4 g2:1/4",
			"f#1:4",
			"f#1:4",
			"g1:4",
			"f#1:4"
		]},
		{"role": "perc", "bars": [
			"-:4",
			"-:4",
			"-:4",
			"-:4",
			"h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2",
			"h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2",
			"h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2 h:1/2",
			"s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4",
			"s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4",
			"s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4 s:1/4",
			"-:4",
			"-:4",
			"-:4",
			"-:4"
		]}
	]
}
```

its sheet, `library/glass-hour.cuts.json`:

```
{
	"group": "eerie",
	"mood": "brittle and glassy",
	"feeling": "Walking on a frozen pond and hearing it crack.",
	"intent": "A ice or glass scene, a slow-motion shatter, a tense museum or lab shot.",
	"listen_for": "The wash of notes at 0:29 and how thin it goes again after it.",
	"voicing": "@crypt-dmg",
	"tempo": 66,
	"meter": 4,
	"seconds": 50.91,
	"parts": [
		{"name": "ice", "bar": 1, "second": 0.0,
		 "change": "glassy stabs over a held bass",
		 "feels": "Thin, brittle"},
		{"name": "hairline", "bar": 5, "second": 14.55,
		 "change": "a ticking hat and a nervous lead",
		 "feels": "A line spreads under your feet"},
		{"name": "crack", "bar": 9, "second": 29.09,
		 "change": "a flurry of notes and a snare roll",
		 "feels": "It breaks"},
		{"name": "after", "bar": 11, "second": 36.36,
		 "change": "sparse again: one low bass and a lead",
		 "feels": "Cold water, silence"}
	]
}
```

## mistakes that were made, and how to avoid them

- a bar that does not add up to the meter. write the lengths of a bar down
  and add them. in 4/4 `e5:1/2 g5:1/2 c6:1/2 g5:1/2` is two beats, so two
  more are needed. the checker reports the sum it found.
- parts of different lengths. every part has the same number of bars; a
  break is a bar of `-:4`, not a missing bar.
- a pair that clips. it is refused only when rendered, so render every tune
  with its voicing. a dense part (sixteenths in two voices over a drum) can
  clip in `@dmg-echo`, `@leap-dmg` and `@overdrive-dmg`; use `@dmg-pad`,
  `@tape-chords` or `@tape-echo`, or thin the part.
- a sheet that says one thing and a tune that does another. render the roll
  and look.
- a time quoted in `listen_for` that is not where the part is. work it out
  from the bar: `(bar - 1) times meter times 60 divided by tempo`.
