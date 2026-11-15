# writing a score for cheeptune

this page is for a small language model that has never seen the format. read
all of it once, then write one score file, then check it. the checker tells
you every mistake at once; fix them all and check again.

you write a score. a voicing says what plays it, and you can choose one that
already exists instead of writing one. a score never names a sound: it names
roles, and notes in those roles.

## the shortest way to a tune

1. choose a built-in voicing from the list below.
2. write a score with all four roles: lead, pad, bass, perc. every built-in
   voicing needs all four parts, even a quiet one, because it is refused if
   a part it plays or drops is missing. a part that has nothing to play is
   all rests: `-:4` for a bar of four beats.
3. save it as a file, for example `mine.json`.
4. check it: `cheeptune.Check(w, "mine.json", "@dmg-pad")` writes what the
   documents hold and every problem, one a line.
5. render it: `cheeptune.Open("mine.json", "@dmg-pad")`. a pair can be
   refused at this step for clipping even when the check is clean; the
   error names the peak. pick another voicing in the list.

## the document format

both documents are json with two changes: a `//` comment runs to the end of
the line, and an unknown field is an error, not skipped. there are no
trailing commas. a file with a `parts` key is a score and a file with a
`channels` key is a voicing; a file with both, or neither, is refused.

## the score

```
{
  "name": "ember",
  "tempo": 66,
  "meter": 4,
  "loop": true,
  "parts": [
    {"role": "lead", "bars": ["e5:2 c5 a4", "c5:2 b4:2"]},
    {"role": "perc", "bars": ["k -:1 s -", "k -:1 s -"]}
  ]
}
```

| field | type | rule |
| --- | --- | --- |
| `name` | text | required, not empty |
| `tempo` | number | beats a minute, 40 to 300. the tempo the piece starts at, and its only tempo unless `tempos` says otherwise |
| `meter` | whole number | beats in a bar, 1 to 16 |
| `loop` | true or false | optional, false when left out |
| `parts` | list | at least one part |
| `tempos` | list | optional. each entry is `{"bar": 9, "tempo": 132}`: from the start of that bar the piece is at that tempo. see "a tempo that changes" |

a part is `{"role": ..., "bars": [...]}`. the roles are exactly `lead`,
`bass`, `pad` and `perc`. a role appears once. every part has the same
number of bars, and at least one. each bar is one text string of tokens
separated by spaces.

a bar has to add up to the meter exactly. in 4/4 a bar holds four beats,
in 3/4 three. the sum is exact: `1/3` three times is one beat.

### tokens

| token | meaning |
| --- | --- |
| `a4` | the note a in octave 4, which is 440 hertz. a letter a to g, lowercase, an optional `#` (sharp) or `b` (flat), then an octave digit 0 to 9. `c#5`, `bb3`, `e2`. middle c is `c4` |
| `-` | a rest |
| `(c4 e4 g4)` | a chord of two or more notes, in parentheses, with spaces between. no chord inside a chord. the channel plays the notes one after another very fast, which sounds like a chord |
| `k`, `s`, `h` | a hit: a lowercase word, letters only, not shaped like a note. the voicing decides what it sounds like. only a `perc` part uses hits |
| `(k h)` | hits struck together: two or more hit names in parentheses, in a `perc` part. it takes a length and a loudness mark as a chord of notes does: `(k h):1/2@f`. a chord is all notes or all hits. the studio sounds every hit. the Game Boy has one noise channel and plays the first hit written, so write the one that matters first |
| `:n` or `:n/d` | after any token, its length in beats. no suffix means one beat. `:2` two beats, `:1/2` half a beat, `:3/2` a beat and a half, `:1/3` a third. the numbers are whole and above nought |
| `~` | after a note, a tie: it ends the bar, and the first token of the next bar must be the same pitch, and the two sound as one longer note. only a single note ties, only as the last token of a bar, and a rest or a hit does not tie |

| `c4@p` | a loudness mark: an `@` and a word after a note, a chord or a hit, and after its length. one of `pp`, `p`, `mp`, `mf`, `f`, `ff`, from very soft to very loud. no mark means `mf`. `c5:1/2@p`, `(c4 e4 g4):2@ff`, `k@f`. a tie goes last: `c4:4@p~`. see "loud and soft" |
| `>` | after a single note, last of all: a slide. the note runs into the next one, which does not start again; the pitch glides to it. `a2:1/4> a3:1/4`, `c3:1/4@ff>`. the next token must be a single note. a note slides or ties, not both. see "slides" |

write the length after the closing parenthesis for a chord: `(c4 e4 g4):2`.
a silent bar is `-:4` (or whatever the meter is), never an empty string.

only `perc` parts hold hits and only the other three hold notes and chords,
because the noise channel plays hits and the other channels play pitch.

### the roles and the machine

the machine is a Game Boy and has four channels. each role is played by one
of them in the built-in voicings.

| role | channel | what it can do |
| --- | --- | --- |
| `lead` | pulse | one note at a time, `c2` to `c8`. a thin buzzing square wave. use `c4` to `c6` for a melody |
| `pad` | pulse | the second pulse. one note at a time, `c2` to `c8`. chords are played as a fast cycle through their notes. use `c3` to `c5` |
| `bass` | wave | one note at a time, `c1` to `c7`, a soft rounded tone. use `c2` to `c3`. its volume has only four steps, so it is blunt |
| `perc` | noise | hits only, no pitch. `k` a kick, `s` a snare, `h` a hat in every built-in voicing |

a note outside the channel's range is refused. one channel plays one thing at
a time: two notes that overlap in one part cannot exist, because a bar lays
tokens end to end.

## the voicing

you rarely write one. use a built-in. this is here so that an error about a
voicing makes sense.

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

fields of the voicing: `name` (required), `machine` (`gameboy` or
`studio`), `channels` (at least one), `drop` (a list of roles nothing
plays). what follows here is the `gameboy` machine. the `studio` machine
has its own section, "the studio machine".

the Game Boy has two `pulse` channels, one `wave` and one `noise`. a fifth
channel, or a third pulse, is refused.

fields of a channel:

| field | applies to | rule |
| --- | --- | --- |
| `kind` | all | `pulse`, `wave` or `noise` |
| `role` | all | the role it plays. a channel plays a role or echoes one, never both, never neither. a role is played by one channel only |
| `echo` | all | the role it copies, later and quieter. needs `delay` and `gain` |
| `delay` | echo only | beats, written `"3/4"` or `"1"`. a text string |
| `gain` | echo only | 0 to 1, how loud the copy is |
| `volume` | all | 0 to 1 |
| `detune` | pitched | cents, minus 100 to 100 |
| `pan` | all | `left`, `right` or `both` |
| `duty` | pulse, required | exactly 0.125, 0.25, 0.5 or 0.75. smaller is thinner and brighter |
| `table` | wave, required | `triangle`, `saw`, `square`, or 32 hex digits, one a step, 0 to f |
| `envelope` | pitched | `attack`, `decay`, `release` in seconds, not below nought; `sustain` 0 to 1. without one a note is full while it lasts and silent after |
| `vibrato` | pitched | `rate` above 0 to 20 hertz, `depth` 0 to 100 cents, `delay` seconds before it starts |
| `arp` | pitched | how a chord cycles: a whole number 1 to 60 is driver ticks a step (shimmer), a text such as `"1/8"` is beats a step (broken chord). default 3 ticks |
| `hits` | noise, required | a map of hit name to `{shift 0 to 13, divisor 0 to 7, short true or false, volume 0 to 1, decay above 0 to 4 seconds}`. `short` makes it tonal |

a `drop` role must be a role the score has, and a role a score has must be
played or dropped. a hit in the score must be a name the noise channel's
`hits` holds. a note sent to the noise channel, or a hit sent to a pitched
channel, is refused.

## built-in voicings

all of them use the lead on a pulse, the pad on a pulse, the bass on the
wave and the perc on noise, except where it says it drops the pad.

| name | how it sounds |
| --- | --- |
| `@crypt-dmg` | short, dry and loud. notes are over before the next beat. chord stabs shimmer. saw bass. lead left, pad right |
| `@dmg-echo` | a singing lead with a quiet late copy of it on the right. smooth triangle bass. drops the pad, so the chords are not heard |
| `@dmg-pad` | a round square lead. the pad plays the chords as a slow soft broken arpeggio under it. smooth |
| `@leap-dmg` | a round lead with a little vibrato. the chords broken into quick sixteenths. short, hopping drums |
| `@overdrive-dmg` | thin and bright. narrow lead, shimmering stabs, a square bass. has a fourth hit, `t`, a tonal tom, which the others lack |
| `@tape-chords` | soft and slow, lo-fi. a lead on the left with slow vibrato, the chords as a swung arpeggio in thirds of a beat |
| `@tape-echo` | lo-fi with no chords. the lead and a flat quiet copy of it two thirds of a beat late. drops the pad |

the voicings above are the Game Boy machine. the ones below are the studio
machine (see "the studio machine" further down): every chord sounds all its
notes at once, notes ring on, and there is a room around them. all four
roles are played, and each has the six hits `k`, `s`, `h`, `o`, `t` and
`b` (kick, snare, closed hat, open hat, tom, bell).

| name | how it sounds |
| --- | --- |
| `@acid` | acid. the bass and the lead are each one bright saw through a steep filter that snaps open on every note and squelches, with overdrive. slides (`>`) and accents (`@f`, `@ff`) act on both. square and saw chord stabs, a hard kick, a clap-like snare, tight hats. write short notes for it, at 130 to 150 |
| `@acid-clockwork` | the acid bass and lead of `@acid` over struck metal that ticks: `k` a thud on a plate, `s` a ratchet, `h` a fine tick, `o` a longer tick, `t` an anvil, `b` a chime. the chords are short strikes on a metal bar |
| `@glass-bells` | bells. an fm bell lead that rings on, the chords struck on a bowl, a round bass, quiet drums, a long bright room. made for glass-hour |
| `@glass-struck` | struck and plucked. the lead on a struck bowl, the chords on plucked strings, a deep plain bass, drums barely there, a very large room |
| `@night-pad` | dark and wide. two detuned saws under a low filter swell on the chords, left and right; a soft lead that repeats three quarters of a beat later; a low bass |
| `@analog-drive` | bright and hard. a driven saw lead through a ringing filter, a square bass, drums with a hard front, almost no room |
| `@wire-harp` | all plucked strings. a bright lead answered a quarter beat later by a duller string on the right, plucked chords, a low string bass, knocks for drums, a small room |
| `@iron-works` | hard metal. the lead is a struck tube with a clanging fm tone on top, the chords a struck plate, a growling bass, heavy drums that repeat off the beat |

## built-in scores

use these as they are or as a model for the shape of one.

| name | how it sounds |
| --- | --- |
| `@crypt` | 130 a minute, a minor, eight bars. a dungeon floor, a drum on every half beat, off-beat chord stabs, bass bouncing an octave |
| `@dusk` | 92 a minute, slow, major sevenths and ninths. long chords, a lead that sings. eight bars |
| `@leap` | 150 a minute, c major, eight bars. a platformer. the lead leaps in thirds and octaves, the bass bounces root to fifth |
| `@overdrive` | 172 a minute, e minor, eight bars. high energy, sixteenth notes in the bass, a lead that climbs. needs `@overdrive-dmg` because it uses the hit `t` |
| `@tape` | 76 a minute, swung eighths written as `2/3` and `1/3`, sevenths and ninths. lo-fi. eight bars |

all five loop. `@overdrive` works only with `@overdrive-dmg`.

some pairs are refused because the sum of the four channels passes full scale
and a clipped sample is wrong, so the render stops and names the peak.
the refused pairs are:

| score | voicing | why |
| --- | --- | --- |
| `@crypt` | `@dmg-echo` | the mix clips |
| `@crypt` | `@overdrive-dmg` | the mix clips |
| `@leap` | `@overdrive-dmg` | the mix clips |
| `@overdrive` | any but `@overdrive-dmg` | the hit `t` is not defined |

the examples in this page have their own list of voicings that work.

## the studio machine

the studio is the second machine. the score is the same: any score plays
through a studio voicing unchanged. what changes is what the voicing can
ask for.

- a chord sounds all its notes at once, each with its own envelope.
- there are five kinds of channel, each a different way of making a
  sound, where the Game Boy has three.
- up to sixteen channels, and more than one may play the same role, which
  layers their sounds on the same notes.
- every channel has a place between left and right, and can be sent to a
  shared reverb (a room) and a shared delay (repeats in time with the
  beat).
- a note can be marked loud or soft in the score.

`SOUNDS.md` teaches how to design an instrument. this section is the
reference: every field, its range, and every error.

```
{
  "name": "small-bells",
  "machine": "studio",
  "master": 0.4,
  "reverb": {"size": 0.8, "decay": 3, "damping": 0.3, "level": 0.5},
  "delay": {"beats": "3/4", "feedback": 0.4, "tone": 0.5, "level": 0.5},
  "channels": [
    {"kind": "modal", "role": "lead", "volume": 0.8, "place": -0.2,
     "send": {"reverb": 0.5, "delay": 0.2},
     "modal": {"set": "bar", "hardness": 0.7, "decay": 3}},
    {"kind": "pluck", "role": "pad", "volume": 0.5, "place": 0.3,
     "send": {"reverb": 0.4},
     "pluck": {"brightness": 0.6, "damping": 0.3, "position": 0.2,
               "decay": 4, "release": 1}},
    {"kind": "analog", "role": "bass",
     "envelope": {"attack": 0.005, "decay": 2, "sustain": 0.5,
                  "release": 0.3},
     "analog": {"oscillators": [{"shape": "triangle"}],
                "filter": {"cutoff": 600, "resonance": 0.1}}},
    {"kind": "drum", "role": "perc", "volume": 0.7,
     "drums": {"k": {"tone": 50, "sweep": 30, "fall": 0.05, "decay": 0.3},
               "s": {"tone": 190, "decay": 0.1, "noise": 0.4,
                     "noise_decay": 0.15, "colour": 1800},
               "h": {"noise": 0.3, "noise_decay": 0.04, "colour": 7000}}}
  ]
}
```

### the mix

fields of a studio voicing, beside `name`, `machine`, `channels` and
`drop`:

| field | rule |
| --- | --- |
| `master` | 0.01 to 4, 0.5 when left out. the whole mix is multiplied by it. it is the first thing to lower when a pair clips |
| `reverb` | optional. the one room. `size` 0 to 1, how big; `decay` 0.1 to 20, the seconds a sound takes to die away in it; `damping` 0 to 1, how much faster the high end dies (0 is a bright cold room, 1 a dull warm one); `level` 0 to 1, how loud the room is |
| `delay` | optional. the one delay. `beats`, a text string like `"3/4"`, how long until a sound comes back; `feedback` 0 to 0.95, how much of each repeat is repeated again (0 is one repeat); `tone` 0 to 1, how bright the repeats are; `level` 0 to 1 |

a mix that goes over full scale is refused, as on the Game Boy. the error
says the peak and the master: divide the master by the peak and a little
more.

### a studio channel

| field | applies to | rule |
| --- | --- | --- |
| `kind` | all | `fm`, `analog`, `pluck`, `modal` or `drum` |
| `role` | all | the role it plays. several channels may play the same role; all of them sound |
| `echo`, `delay`, `gain` | echo only | as on the Game Boy: a later, quieter copy of a role, here through this channel's own sound |
| `volume` | all | 0 to 1 |
| `place` | all | minus 1 (left) to 1 (right), 0 when left out. `pan` is the Game Boy's and is refused here |
| `send` | all | `{"reverb": 0.5, "delay": 0.2}`, each 0 to 1: how much of the channel goes to the room and to the delay. the voicing must have the effect it sends to |
| `voices` | all but drum | 1 to 16, 8 when left out. how many notes the channel sounds at once. when a new note needs a voice and all are in use, the oldest note that has already been let go is cut off, and if none has, the oldest note is |
| `detune` | all but drum | cents, minus 100 to 100 |
| `glide` | fm, analog | seconds a slide takes to reach the next note, 0.001 to 2, 0.06 when left out |
| `arp` | all but drum | as on the Game Boy, ticks or beats. with an arp a chord is not sounded whole: its notes are struck one after another. leave it out for a real chord |
| `envelope` | analog | attack, decay, sustain, release. see below: the studio reads the four numbers as a curve |
| `vibrato` | fm, analog | rate 0.05 to 20 hertz, depth 0 to 100 cents, delay in seconds |
| `fm`, `analog`, `pluck`, `modal`, `drums` | its own kind | the sound itself. each kind needs its own and refuses the others |

the notes a pitched studio channel plays are `c0` to `c8`.

the studio's envelope takes the same four numbers as the Game Boy's and
reads them as a curve. `attack` is a straight rise, in seconds. `decay` is
the seconds the level takes to fall from full to `sustain`. `release` is
the seconds it takes to die away after the note ends. times are 0 to 30.
a struck or plucked sound has `sustain` 0 and a long `decay` and
`release`, so a short note still rings.

### fm

sine waves that bend each other. good for bells, glass, electric pianos
and metal.

| field | rule |
| --- | --- |
| `arrangement` | `chain`, `fan`, `pairs` or `sum`. how the operators are joined |
| `operators` | two to four (`pairs` takes exactly four). each is `{"ratio": 1, "level": 0.8, "envelope": {...}}` |
| `ratio` | 0.125 to 32. the operator's frequency as a multiple of the note |
| `level` | 0 to 1. for an operator that is heard, how loud. for one that bends another, how hard it bends: 0.05 is a little colour, 0.3 is harsh |
| `envelope` | optional, the operator's own. without one it is full while the note is held |
| `feedback` | 0 to 1. the last operator bends itself, which roughens it toward a buzz |

in `chain`, operator 4 bends 3, 3 bends 2, 2 bends 1, and 1 is heard. in
`fan`, operators 2, 3 and 4 each bend 1, and 1 is heard. in `pairs`, 2
bends 1 and 4 bends 3, and 1 and 3 are heard. in `sum`, nothing bends
anything and all are heard.

### analog

one or two plain waves through a filter. good for basses, pads, strings
and brass-like leads.

| field | rule |
| --- | --- |
| `oscillators` | one or two. each is `{"shape": "saw"}` with optional `width`, `detune`, `octave`, `level` |
| `shape` | `saw` (bright, buzzy), `square` (hollow) or `triangle` (soft, nearly pure) |
| `width` | square only, 0.05 to 0.95, 0.5 when left out. away from 0.5 the square gets thinner and more nasal |
| `detune` | cents, minus 100 to 100. two oscillators a few cents apart beat slowly and sound wide |
| `octave` | minus 3 to 3 |
| `level` | 0 to 1. the balance between the oscillators; together they are as loud as one |
| `noise` | 0 to 1. hiss mixed in before the filter |
| `drive` | 0 to 1. pushes the sound into a soft clipper before the filter: louder, harder, more overtones |
| `filter` | optional. `cutoff` 20 to 20000 hertz, written for middle c: above it the sound is taken away. `resonance` 0 to 1: a peak at the cutoff, a ringing whistle near 1. `follow` 0 to 1: how much the cutoff moves up and down with the note. `amount` minus 8 to 8: how many octaves the filter's own `envelope` opens the cutoff at its peak. an `amount` needs an `envelope` |
| `filter` `poles` | 2 or 4, 2 when left out. 2 is a gentle filter. 4 is twice as steep and its resonance squelches: the acid filter |
| `filter` `accent` | 0 to 8. how many more octaves the envelope opens the filter on a note marked `ff`. `f` gets half of it; `p` and `pp` close the filter instead. needs an `envelope` |
| `filter` `snap` | 0 to 0.95. how much shorter the envelope's decay is on a note marked `ff`: 0.5 halves it. needs an `envelope` |
| `fold` | 0 to 1. after the filter and before the channel's envelope: bends the wave back on itself. harsh, buzzing, metallic |
| `overdrive` | 0 to 1. after the filter and the fold, before the channel's envelope: pushes the sound into a soft clipper and brings it back to full level. thick, loud, distorted, and the note still dies away as its envelope says. unlike `drive` it cannot make the channel louder than full scale |

### pluck

a plucked string. good for harps, guitars, kotos and plucked basses.

| field | rule |
| --- | --- |
| `brightness` | 0 to 1. how bright the pluck is: 0 a soft thumb, 1 a hard pick |
| `damping` | 0 to 1. how fast the high overtones die against the low |
| `position` | 0 to 0.5. where the string is plucked. near 0 is thin and twangy; 0.5 is the middle, round and hollow |
| `decay` | 0.01 to 30. seconds the note takes to die away while held |
| `release` | 0.01 to 30. seconds it takes to die away after the note ends |

### modal

a struck object. good for bells, bars, bowls, plates and anything hit.

| field | rule |
| --- | --- |
| `set` | a named object: `bar` (a glockenspiel bar), `marimba`, `plate` (a metal disc), `bowl` (a singing bowl or a glass), `tube` (a tubular bell) |
| `modes` | instead of a set, up to twelve of your own: `{"ratio": 2.76, "decay": 1, "level": 0.5}`. ratio 0.1 to 40, decay 0.01 to 30 seconds, level 0 to 1 |
| `hardness` | 0 to 1. the blow: 0 a soft mallet, which rings only the low overtones; 1 metal on metal, which rings them all |
| `decay` | with a set, 0.01 to 30: the seconds the lowest overtone rings. the higher ones die sooner. not used with your own modes |
| `release` | optional, 0 to 30. seconds the sound takes to die after the note ends, as if a hand were laid on it. left out, the object rings on however short the note |

### drum

hits made from a tone that falls in pitch, a click, and filtered hiss.
`drums` is a map from the hit's letter in the score to its sound.

on the studio a letter means the same kind of hit in every voicing, so a
score's drums move between voicings without being rewritten. there are
six, and no others are allowed:

| letter | is |
| --- | --- |
| `k` | the kick, or what stands in for it: the low, heavy hit |
| `s` | the snare or clap: the hit on the back beat |
| `h` | the closed hat or a tick: short and high |
| `o` | the open hat: high, and longer than `h` |
| `t` | the tom: a pitched drum between the kick and the snare |
| `b` | the bell: a pitched or metallic accent that rings a little |

every built-in studio voicing has all six. a voicing of your own may
leave some out; a score that uses a letter the voicing lacks is refused.
the Game Boy's built-in voicings have `k`, `s` and `h`, and one has `t`.

| field | rule |
| --- | --- |
| `tone` | 20 to 2000 hertz. the pitch the hit lands on. leave out for a hit that is all hiss |
| `decay` | 0.001 to 8. seconds the tone rings. needed with a tone |
| `sweep` | 0 to 60. how many semitones above its tone the hit starts |
| `fall` | 0.001 to 8. seconds the pitch takes to fall. needed with a sweep |
| `click` | 0 to 1. a three millisecond tick at the front |
| `noise` | 0 to 1. how much hiss |
| `noise_decay` | 0.001 to 8. seconds the hiss lasts. needed with noise |
| `colour` | 20 to 16000 hertz. the hiss is cut away below this. needed with noise |
| `volume` | 0 to 1 |
| `struck` | optional: `bar`, `bowl`, `marimba`, `plate` or `tube`. the tone is then not a pure tone but that metal object struck at the tone's pitch, ringing for `decay`. with a short `decay` it ticks. needs a `tone`, takes no `sweep` |
| `hardness` | 0 to 1, with `struck` only. how hard the blow is |

a kick is a low tone with a big sweep and a click. a snare is a tone
near 200 and hiss above 1500. a hat is hiss above 7000 with no tone, 0.04
seconds for closed and 0.3 for open. a tom is a tone with a small sweep.

### loud and soft

a mark on a note says how loud it is: `c5@p`, `c5:1/2@ff`. the marks are
`pp`, `p`, `mp`, `mf`, `f`, `ff`. `mf` is what an unmarked note is. `pp`
is about a quarter as loud and `ff` about one and a half times.

the studio plays the marks. the Game Boy machine reads them and plays the
score exactly as if they were not there. so a score with marks still
works with every voicing.

use them the way a player would: the first beat of a bar a little louder
than the rest, a phrase that grows to its top note and falls away, a
quiet answer after a loud statement. a line where every note is the same
loudness sounds like a machine. `f` and `ff` can make a loud pair clip.

### slides

`a2:1/4> a3:1/4` slides from the first note into the second. the second
note does not start again: there is no new attack, the filter's envelope
does not fire again, and the pitch glides from the first note to the
second over the channel's `glide` time. a chain is allowed, `c3:1/4>
d#3:1/4> c4:1/2`, and a slide may cross a bar line. a loudness mark on a
note inside a slide still changes how loud that note is.

only `fm` and `analog` channels slide. on `pluck` and `modal` channels,
on a channel with an `arp`, and on the Game Boy machine, a note after a
`>` is played as an ordinary new note. so a score with slides still
works with every voicing.

a slide is what makes an acid bass line slither: write it between two
notes an octave apart, or into an accented note.

### a tempo that changes

```
"tempo": 66,
"tempos": [{"bar": 9, "tempo": 132}, {"bar": 13, "tempo": 66}]
```

the piece starts at `tempo`. at the start of each listed bar it takes the
new tempo at once and keeps it until the next change. bars are counted
from one, each change is at a later bar than the one before, and the
first is at bar 2 or later. both machines follow it.

there is no gradual change. to speed up over four bars, write four
changes, one a bar. a cut sheet for a score with changes gives each
part's `second` as it really falls, and `seconds` as the real length;
`check` prints the length.

### studio errors

| the error says | change |
| --- | --- |
| `the studio has 16 channels and the voicing has 17` | remove channels |
| `is only for the gameboy machine` | `duty`, `table`, `hits` and `pan` are the Game Boy's. on the studio use `place` for `pan` and `drums` for `hits` |
| `pluck is only for a channel of kind pluck` | the sound's object does not match the channel's `kind`. the same for `fm`, `analog`, `modal` and `drums` |
| `a channel of kind fm needs fm` | the channel has no sound. add the object named after its kind |
| `envelope is not for a fm channel` | only `analog` takes the channel's `envelope`. an fm operator has its own, and pluck, modal and drum have `decay` |
| `vibrato is not for a pluck channel` | only `fm` and `analog` take vibrato |
| `arp, detune and voices are for pitched channels` | remove them from the drum channel |
| `sends to a reverb and the voicing has none` | add a `reverb` to the voicing, or remove the send. the same for the delay |
| `place 2 is outside -1 to 1` | every number says its range when it is outside it. bring it inside |
| `fm arrangement "tree"; arrangements are chain, fan, pairs, sum` | use one of the four |
| `fm arrangement pairs takes [4] operators and this has 2` | give that arrangement the number of operators it takes |
| `analog has 0 oscillators` | one or two |
| `shape "sine"; shapes are saw, square, triangle` | use one of the three. for a sine use `fm` with `sum` |
| `width is only for a square` | remove it from the saw or triangle |
| `filter amount needs a filter envelope to move by` | add `"envelope"` inside the filter, or remove `amount` |
| `filter accent and snap need a filter envelope to act on` | the same: add the filter's `envelope` |
| `filter poles 3; a filter has 2 or 4` | use 2 or 4 |
| `glide is not for a pluck channel; fm and analog slide` | remove `glide` |
| `a struck drum needs a tone and takes no sweep` | give the drum a `tone` and remove `sweep` and `fall` |
| `hardness is for a struck drum` | add `struck`, or remove `hardness` |
| `modal set "anvil"; sets are bar, bowl, marimba, plate, tube` | use one of the five, or give your own `modes` |
| `modal has a set and its own modes; give one` | keep `set` or `modes` |
| `modal decay is for a named set` | with your own `modes`, each mode has its own `decay`; remove the outer one |
| `a drum channel with no drums plays nothing` | add a `drums` map |
| `a studio drum is one of k, s, h, o, t, b` | the drum's name is not one of the six letters. rename it to the letter for what it is |
| `has neither a tone nor noise and is silent` | give the drum a `tone` or some `noise` |
| `a note on a drum channel; it plays hits` | a drum channel is playing a role that holds notes. bind it to `perc` |
| `is outside the studio's range c0 to c8` | move the note into range |
| `and clips; lower the voicing's master` | the mix is over full scale. divide the master by the peak the error gives, and a little more |

## errors

the checker returns every error at once, one to a line, most beginning with
the part and bar it is in. fix all of them and check again. an error you did
not fix is shown again. a table of what each says and what to change:

### the file

| the error says | change |
| --- | --- |
| `not a json object` | the file is not valid json. look for a missing comma, a missing quote, a trailing comma or a stray bracket |
| `unknown field` | a key is misspelt or does not exist. the message names it. use only the fields in this page |
| `there is more after the document ends` | text follows the closing brace. delete it |
| `holds both parts and channels` | the file is two documents. keep only a score |
| `neither a score (parts) nor a voicing (channels)` | the top-level keys are wrong. a score has `parts` |
| `is a voicing, and a score belongs here` | the score and voicing were handed over in the wrong order, or one is the wrong file |
| `no built-in` | an `@name` that does not exist. the message lists the ones that do |

### the score

| the error says | change |
| --- | --- |
| `score has no name` | add `"name"` |
| `is outside 40 to 300` (tempo) | set `tempo` between 40 and 300 |
| `meter 0 is outside 1 to 16` | set `meter` to a whole number from 1 to 16 |
| `score has no parts` | add the four parts |
| `unknown role` | a role is not `lead`, `bass`, `pad` or `perc` |
| `the role appears twice` | merge the two parts of that role into one |
| `every part has the same` | give each part the same number of bars; add rest bars such as `-:4` |
| `has no bars` | a part needs at least one bar |
| `holds 3 beats and the meter is 4` | the bar's lengths add up to the wrong number. add or lengthen tokens, or shorten them, until the sum is the meter |
| `the bar is empty` | a bar is `""`. write `-:4`, or the meter's number |
| `not a note (a4, c#5, bb3), chord, rest (-) or hit (k)` | the token is none of them. notes are a letter a to g, an optional sharp or flat, an octave. hits are letters only |
| `is not a note` | a note inside a chord is misspelt |
| `octave 10 is past 9` | use an octave 0 to 9 |
| `a chord has two notes or more` | a chord of one note is just the note. remove the parentheses |
| `a chord inside a chord` | remove the inner parentheses |
| `a ) with no (` or `a ( with no )` | balance the parentheses |
| `is not a number or n/d` | a length is not `2` or `1/2`. write whole numbers |
| `is not above nought` | a length is zero or negative. use a length above nought |
| `a rest does not tie` / `a hit does not tie` | remove the `~` |
| `only a single note ties` | remove the `~` from the chord |
| `a tie is only on a bar's last token` | move the `~` to the last token of its bar |
| `lands on ... not the same pitch` | the first token of the next bar must be the same note as the tied one. change it or remove the `~` |
| `ties into a bar that is not there` | the last bar ties on. remove the `~` or add a bar |
| `a chord is notes or hits, not both` | a chord mixes a note and a hit. split it |
| `the hit k is in the chord twice` | name each hit once in a chord of hits |
| `only a single note slides` | a `>` is on a chord, a rest or a hit. remove it |
| `the slide before it lands on a single note` | the token after a `>` is a rest, a chord or a hit. make it a single note or remove the `>` |
| `slides into a bar that is not there` | the last note of the part has a `>`. remove it |
| `a note slides (>) or ties (~), not both` | keep one |
| `loudness mark "x"; marks are pp, p, mp, mf, f, ff` | what follows the `@` is not one of the six. use one of them |
| `a rest has no loudness to mark` | remove the `@` and its mark from the rest |
| `a tied note keeps the loudness it began with` | the note a tie lands on has a mark. remove it; put the mark on the first of the tied notes |
| `tempo change 1: bar 1 is not after bar 1` | a change is at bar 2 or later, and each is at a later bar than the one before. the first bar's tempo is `tempo` |
| `tempo change 1: bar 9 is not in the score` | the change names a bar past the last one |
| `tempo change 1: tempo 900 is outside 40 to 300` | a changed tempo keeps the same range as the first |

### the voicing

| the error says | change |
| --- | --- |
| `voicing has no name` | add `"name"` |
| `unknown machine` | `"machine"` is `"gameboy"` or `"studio"` |
| `voicing has no channels` | add at least one channel |
| `unknown kind` | on the gameboy a channel's kind is `pulse`, `wave` or `noise`. on the studio it is `fm`, `analog`, `pluck`, `modal` or `drum` |
| `is only for the studio machine` | the field belongs to the studio. remove it, or make the voicing's machine `studio` |
| `the gameboy has 2 pulse channels and this is one more` | remove a channel of that kind; the machine has two pulse, one wave, one noise |
| `is already played by another channel` | one channel per role. make the second one an echo, or remove it |
| `drop: unknown role` | a dropped role must be one of the four |
| `is dropped and also played` | remove it from `drop` or from the channels |
| `plays ... and echoes ...; a channel does one` | keep either `role` or `echo` |
| `neither plays a role nor echoes one` | give the channel a `role` or an `echo` |
| `an echo needs a delay in beats` / `an echo needs a gain` | add `"delay": "3/4"` and `"gain": 0.35` |
| `delay is only for an echo` / `gain is only for an echo` | remove it. use `volume` on an ordinary channel |
| `is outside 0 to 1` | volume, gain, sustain and hit volume are 0 to 1 |
| `pans are left, right, both` | use one of the three |
| `is past a semitone` | detune is minus 100 to 100 cents |
| `duty is only for a pulse` / `table is only for a wave` / `hits are only for noise` | the field is on the wrong kind of channel. move or remove it |
| `envelope, vibrato, arp and detune are for pitched channels` | remove them from the noise channel. a hit has its own `decay` |
| `a pulse needs a duty` / `the hardware has 0.125, 0.25, 0.5, 0.75` | set duty to one of those four |
| `a wave needs a table` / `is not triangle, saw, square or 32 hex digits` | name a table or give 32 hex digits |
| `noise with no hits plays nothing` | add a `hits` map |
| `a hit is a lowercase word` | rename the hit with letters only, not shaped like a note |
| `shift 14 is outside 0 to 13` / `divisor 9 is outside 0 to 7` | keep within those |
| `decay 9 s is outside 0 to 4` | a hit's decay is above 0 and up to 4 seconds |
| `envelope times are not below nought` | no negative times |
| `vibrato rate` / `vibrato depth` | rate above 0 to 20 hertz, depth 0 to 100 cents |
| `ticks are a whole number, 1 to 60` / `arp is ticks (3) or beats ("1/8")` | write arp as `3` or `"1/8"` |

### the pair

| the error says | change |
| --- | --- |
| `drop: the score has no pad part` | the voicing drops a role the score lacks. add that part, all rests |
| `no channel plays it and it is not dropped` | the score has a part that the voicing neither plays nor drops. remove the part or use another voicing |
| `the score has no lead part` (and so on) | a channel plays a role the score lacks. add that part, all rests |
| `echoes lead, which is dropped` | remove the role from `drop` or the echo |
| `a note on the noise channel; it plays hits` | put hits in `perc` and notes in the other roles |
| `hit "t" is not one of h, k, s` | use only the hit names the voicing lists |
| `hit "k" on a pitched channel` | a hit is in a part that a pitched channel plays. keep hits in `perc` |
| `is outside the pulse range c2 to c8` | move the note into range; the wave range is `c1` to `c7` |
| `the mix peaks at 1.027 and clips` | choose another voicing from the list, or write a voicing with a lower `volume` on a channel |

## three worked scores

each is a complete valid score. the voicings that work with each are listed.
a test in this package parses and renders each file, so they stay valid.

### slow and dark: ember

the file is `examples/ember.json`. a minor, 66 a minute, eight bars, about
29 seconds. works with `@crypt-dmg`, `@dmg-echo`, `@dmg-pad`, `@leap-dmg`,
`@tape-chords` and `@tape-echo`. `@overdrive-dmg` clips it.

```
{
	// slow and dark. a minor, 66 a minute, am f dm e with the major e
	// pulling back to am. the lead is long notes and falls; the drum is
	// two beats apart. eight bars, about 29 seconds.
	"name": "ember",
	"tempo": 66,
	"meter": 4,
	"loop": true,
	"parts": [
		{"role": "lead", "bars": [
			"e5:2 c5 a4",
			"c5:2 b4:2",
			"a4:2 c5 f5",
			"e5:3 d5",
			"d5:2 f5 a5",
			"g5:2 f5:2",
			"b4:2 g#4 e5",
			"e5:3 -"
		]},
		{"role": "pad", "bars": [
			"(a3 c4 e4):4",
			"(a3 c4 e4):4",
			"(f3 a3 c4):4",
			"(f3 a3 c4):4",
			"(d4 f4 a4):4",
			"(d4 f4 a4):4",
			"(e4 g#4 b4):4",
			"(e4 g#4 b4):4"
		]},
		{"role": "bass", "bars": [
			"a2:3 e3",
			"a2:3 e3",
			"f2:3 c3",
			"f2:3 c3",
			"d2:3 a2",
			"d2:3 a2",
			"e2:3 b2",
			"e2:3 b2"
		]},
		{"role": "perc", "bars": [
			"k -:1 s -",
			"k -:1 s -",
			"k -:1 s -",
			"k -:1 s -",
			"k -:1 s -",
			"k -:1 s -",
			"k -:1 s -",
			"k -:1 s s:1/2 s:1/2"
		]}
	]
}
```

### bright and driving: sunrun

the file is `examples/sunrun.json`. c major, 140 a minute, twelve bars, about
20 seconds. works with `@crypt-dmg`, `@dmg-pad`, `@leap-dmg`,
`@overdrive-dmg`, `@tape-chords` and `@tape-echo`. `@dmg-echo` clips it.

```
{
	// bright and driving. c major, 140 a minute, c g am f three times
	// round, the last four bars leaning on f and g to come back to c.
	// eighth notes in the lead and the bass, a drum on every beat.
	// twelve bars, about 20 seconds.
	"name": "sunrun",
	"tempo": 140,
	"meter": 4,
	"loop": true,
	"parts": [
		{"role": "lead", "bars": [
			"e5:1/2 g5:1/2 c6:1/2 g5:1/2 e5:1/2 g5:1/2 c6",
			"d5:1/2 g5:1/2 b5:1/2 g5:1/2 d5:1/2 g5:1/2 b5",
			"c5:1/2 e5:1/2 a5:1/2 e5:1/2 c5:1/2 e5:1/2 a5",
			"c5:1/2 f5:1/2 a5:1/2 f5:1/2 c5:1/2 f5:1/2 a5",
			"g5:1/2 e5:1/2 c5:1/2 e5:1/2 g5 c6",
			"b5:1/2 g5:1/2 d5:1/2 g5:1/2 b5 d6",
			"a5:1/2 e5:1/2 c5:1/2 e5:1/2 a5 e6",
			"a5:1/2 f5:1/2 c5:1/2 f5:1/2 a5 c6",
			"f5 a5 c6:2",
			"g5 b5 d6:2",
			"f5 a5 c6 a5",
			"g5 b5 d6:1/2 b5:1/2 g5"
		]},
		{"role": "pad", "bars": [
			"-:1/2 (c4 e4 g4):1/2 -:1/2 (c4 e4 g4):1/2 -:1/2 (c4 e4 g4):1/2 -:1/2 (c4 e4 g4):1/2",
			"-:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2",
			"-:1/2 (a3 c4 e4):1/2 -:1/2 (a3 c4 e4):1/2 -:1/2 (a3 c4 e4):1/2 -:1/2 (a3 c4 e4):1/2",
			"-:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2",
			"-:1/2 (c4 e4 g4):1/2 -:1/2 (c4 e4 g4):1/2 -:1/2 (c4 e4 g4):1/2 -:1/2 (c4 e4 g4):1/2",
			"-:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2",
			"-:1/2 (a3 c4 e4):1/2 -:1/2 (a3 c4 e4):1/2 -:1/2 (a3 c4 e4):1/2 -:1/2 (a3 c4 e4):1/2",
			"-:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2",
			"-:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2",
			"-:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2",
			"-:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2 -:1/2 (f3 a3 c4):1/2",
			"-:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2 -:1/2 (g3 b3 d4):1/2"
		]},
		{"role": "bass", "bars": [
			"c3:1/2 c3:1/2 c4:1/2 c3:1/2 c3:1/2 c3:1/2 c4:1/2 c3:1/2",
			"g2:1/2 g2:1/2 g3:1/2 g2:1/2 g2:1/2 g2:1/2 g3:1/2 g2:1/2",
			"a2:1/2 a2:1/2 a3:1/2 a2:1/2 a2:1/2 a2:1/2 a3:1/2 a2:1/2",
			"f2:1/2 f2:1/2 f3:1/2 f2:1/2 f2:1/2 f2:1/2 f3:1/2 f2:1/2",
			"c3:1/2 c3:1/2 c4:1/2 c3:1/2 c3:1/2 c3:1/2 c4:1/2 c3:1/2",
			"g2:1/2 g2:1/2 g3:1/2 g2:1/2 g2:1/2 g2:1/2 g3:1/2 g2:1/2",
			"a2:1/2 a2:1/2 a3:1/2 a2:1/2 a2:1/2 a2:1/2 a3:1/2 a2:1/2",
			"f2:1/2 f2:1/2 f3:1/2 f2:1/2 f2:1/2 f2:1/2 f3:1/2 f2:1/2",
			"f2:1/2 f2:1/2 f3:1/2 f2:1/2 f2:1/2 f2:1/2 f3:1/2 f2:1/2",
			"g2:1/2 g2:1/2 g3:1/2 g2:1/2 g2:1/2 g2:1/2 g3:1/2 g2:1/2",
			"f2:1/2 f2:1/2 f3:1/2 f2:1/2 f2:1/2 f2:1/2 f3:1/2 f2:1/2",
			"g2:1/2 g2:1/2 g3:1/2 g2:1/2 g2:1/2 g2:1/2 g3:1/2 g2:1/2"
		]},
		{"role": "perc", "bars": [
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k:1/2 h:1/2 s:1/2 h:1/2 k:1/2 k:1/2 s:1/2 h:1/2",
			"k s k:1/2 s:1/4 s:1/4 s:1/2 s:1/2"
		]}
	]
}
```

### sparse and tense: wire

the file is `examples/wire.json`. three beats a bar, 96 a minute, eight
bars, about 15 seconds. works with every built-in voicing.

```
{
	// sparse and tense. three beats to a bar, 96 a minute, e against a
	// b flat, the interval that never settles. most of the bars are rests
	// on purpose; the bass is a pulse and the drum is a few hits. eight
	// bars, about 15 seconds.
	"name": "wire",
	"tempo": 96,
	"meter": 3,
	"loop": true,
	"parts": [
		{"role": "lead", "bars": [
			"-:2 e5",
			"f5:3",
			"-:3",
			"b4 - bb4",
			"e5:2 -",
			"f5 e5 bb4",
			"-:3",
			"b4:2 -"
		]},
		{"role": "pad", "bars": [
			"-:3",
			"-:3",
			"(e4 bb4):3",
			"-:3",
			"(e4 bb4):3",
			"-:3",
			"(f4 b4):3",
			"-:3"
		]},
		{"role": "bass", "bars": [
			"e2 -:2",
			"e2 -:2",
			"f2 -:2",
			"e2 -:2",
			"e2 -:2",
			"f2 -:2",
			"e2 -:2",
			"b1 -:2"
		]},
		{"role": "perc", "bars": [
			"-:3",
			"-:2 h",
			"k -:1 h",
			"-:3",
			"k -:1 h",
			"k h h",
			"k:1/2 -:1/2 h s",
			"-:2 s"
		]}
	]
}
```

## rules of thumb

### tempo

| mood | beats a minute |
| --- | --- |
| slow, sad, heavy | 60 to 76 |
| calm, drifting | 76 to 92 |
| walking, neutral | 92 to 110 |
| upbeat, cheerful | 110 to 135 |
| driving, urgent | 135 to 160 |
| frantic, a chase | 160 to 200 |

### notes

dark and moody: a minor key. lean on the root, the third a minor third
above it and the fifth. a minor: `a c e`. add the second and seventh for
colour. end phrases on the root. use `c#`, `d#`, `g#` sparingly. a major
chord on the fifth degree (in a minor, `e g# b`) pulls back to the start
and is the best last chord before a loop.

bright and happy: a major key. lean on the root, the third and the fifth:
c major is `c e g`. the first, fifth, sixth and fourth chords in turn
(`c g am f` in c major) is the most common sequence. melodies that climb
up the chord notes and fall back sound bright.

tense: avoid the major third and the fifth. use the tritone, six semitones
apart: `e` and `bb`, or `f` and `b`. long rests. a note that is held and
then not resolved.

stepwise melodies (neighbouring notes) sound calm, and leaps of a third or
an octave sound energetic. repeat a short phrase twice, then change its
last bar.

### a bass line

a bass bar is the chord's root note, and sometimes its fifth or its octave,
in the range `c2` to `c3`. keep one chord to one bar or two. two ways that
always work:

- slow: the root held for three beats then the fifth for one:
  `a2:3 e3`.
- driving: eighths with the octave on the off beats:
  `c3:1/2 c3:1/2 c4:1/2 c3:1/2 c3:1/2 c3:1/2 c4:1/2 c3:1/2`.

the bass must change to the chord that sounds with it. write the chords
first, then the bass from their roots.

### a drum bar

`k` is a kick, `s` a snare, `h` a hat. in 4/4 the kick falls on beats one
and three and the snare on two and four, which reads as `k s k s`, one beat
each. a hat on the half beats makes it faster: `k:1/2 h:1/2 s:1/2 h:1/2
k:1/2 k:1/2 s:1/2 h:1/2`. a sparse bar is `k -:1 s -`. in 3/4 use `k h h`
or `k -:1 h`. every bar adds up to the meter.

once every four or eight bars, make the last bar a fill: more hits,
shorter, such as `k s k:1/2 s:1/4 s:1/4 s:1/2 s:1/2`. the fill is still a whole bar, so it still sums to the meter.

### chords in the pad

a chord is two to four notes. keep them between `c3` and `c5`. one chord a
bar for a slow piece, or four off-beat stabs a bar for a fast one:
`-:1/2 (c4 e4 g4):1/2` four times. with `@crypt-dmg` and `@overdrive-dmg`
the stabs shimmer. with `@dmg-pad`, `@leap-dmg` and `@tape-chords` they are
broken into an arpeggio.

### how long

the length of a piece is bars times meter times 60 divided by tempo, in
seconds. a short video wants a tune that is a little longer than the
video, or a loop. the bars for the length you want, in 4/4 (the second
number is the length the bars make):

| tempo | 10 seconds | 20 seconds | 30 seconds | 60 seconds |
| --- | --- | --- | --- | --- |
| 70 | 3 bars, 10.3 s | 6 bars, 20.6 s | 9 bars, 30.9 s | 18 bars, 61.7 s |
| 90 | 4 bars, 10.7 s | 8 bars, 21.3 s | 11 bars, 29.3 s | 22 bars, 58.7 s |
| 120 | 5 bars, 10.0 s | 10 bars, 20.0 s | 15 bars, 30.0 s | 30 bars, 60.0 s |
| 140 | 6 bars, 10.3 s | 12 bars, 20.6 s | 17 bars, 29.1 s | 35 bars, 60.0 s |
| 160 | 7 bars, 10.5 s | 13 bars, 19.5 s | 20 bars, 30.0 s | 40 bars, 60.0 s |

round to a whole bar. an eight-bar or a four-bar phrase is the shape most
tunes have; keep the number of bars a multiple of four when you can. in
3/4, a bar is three quarters as long as in 4/4.

### a loop that does not jump

set `"loop": true`. the piece is rendered twice and the second pass kept, so
the echoes and releases at the seam are the ones the middle has, and both
ends fade over 5 milliseconds so there is no click.

to loop cleanly:

- the last bar should lead to the first: end on the fifth degree chord (in
  a minor, `e`; in c major, `g`), or on the root and begin on the root.
- do not end on a long held note that the first bar then cuts. use a short
  last note or end the bar on a rest.
- keep the number of bars a multiple of four when you can, and every part the same.
- the piece's length is the loop. a 30 second video with a 15 second loop
  plays it twice; the player restarts it every `LoopPeriod()`.

a piece that does not loop has a tail, the release of its last notes, that
sounds past its last bar.

## checking, before you say you are done

1. `cheeptune.Check(w, "mine.json", "@voicing")` returns an error and
   writes every problem. an empty list of problems means the documents
   are valid and fit each other.
2. `cheeptune.Open("mine.json", "@voicing")` renders it. an error here
   is a clip or a pair that does not fit; the message says which.
3. the piece's `Duration()` should be near what you meant.

## a tune for a video

this page is the format. `VIDEO.md` is how to write a tune with parts a
video can be cut to, and the cut sheet that says where they are.
