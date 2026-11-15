# designing a sound for the studio machine

this page is for a model that will write a studio voicing: an instrument
or a set of them. you cannot hear what you write. so this page says what
each number does to the sound in plain words, and gives seven instruments
that are known to work, which you change a little at a time.

`SCORING.md` has every field's range and every error. read its section
"the studio machine" first. the short of it: a voicing is JSON, its
`machine` is `"studio"`, and each channel has a `kind` that picks how its
sound is made.

## choosing a kind

| you want | kind | why |
| --- | --- | --- |
| a bell, a chime, glass, an electric piano, a metallic clang | `fm` | one sine bending another makes many overtones from very little, and the overtones can be out of tune with the note, which is what a bell is |
| a bass, a pad, strings, a brassy or buzzy lead | `analog` | a bright wave with a filter taking the top off is the sound of every classic synthesiser |
| an acid bass line: squelching, sliding, distorted | `analog` | one saw or square, the four-pole filter with high resonance, a short filter envelope, `overdrive`. see "an acid line" |
| a harp, a guitar, a koto, a plucked bass | `pluck` | it is a model of a string, so it starts bright and mellows as it dies, as a string does |
| a glockenspiel, a marimba, a bowl, a gong, a tubular bell | `modal` | it is a model of a struck object: a few overtones, each dying at its own speed |
| drums | `drum` | a falling tone plus hiss covers kicks, snares, hats and toms |

## what makes a sound feel a certain way

- **how it starts.** a sound that begins at once (attack 0 to 0.005) is
  struck or plucked: clear, rhythmic. a sound that swells in (attack 0.2
  to 1) is bowed or blown: soft, slow, behind the beat.
- **how it ends.** a sound that stops with the note is dry and tight. a
  sound that rings on (a long `release`, or modal with no `release`)
  blurs one note into the next and fills the space.
- **how bright it is.** more high overtones is brighter, harder, closer.
  fewer is darker, softer, further away.
- **whether its overtones are in tune.** whole-number overtones (2, 3, 4
  times the note) sound like a voice or a string: one clear pitch.
  overtones off the whole numbers (2.76, 5.4) sound like metal or glass:
  a clang, and with enough of them, no clear pitch at all.
- **how much room.** `send` to the reverb puts the sound in a space. a
  lot of reverb with a long `decay` is far away and cold. none is close
  and dry.

## fm, number by number

an fm sound is two to four operators. each is a sine wave. an operator
that is heard is a carrier. an operator that bends another is a
modulator: you do not hear it, you hear what it does to the carrier.

- **a modulator's `ratio`.** at 1, 2 or 3 the result has whole-number
  overtones: at 1 it is full like a saw, at 2 hollow like a clarinet. at
  a ratio that is not a whole number (1.41, 2.76, 3.5) the tone clangs
  like a bell, because its overtones are no longer in tune with the note.
- **a modulator's `level`.** how many overtones. 0.03 is a hint of
  colour on a pure tone. 0.1 is clearly bright. 0.3 is harsh and buzzy.
  above that it turns to noise on high notes.
- **a modulator's `envelope`.** when the overtones are there. a
  modulator that dies faster than its carrier makes a sound that starts
  bright and goes pure, which is every struck and plucked thing. this is
  the single most useful trick in fm.
- **a carrier's `ratio`.** nearly always 1. a second carrier at 2 or 3
  adds an organ-like overtone.
- **a carrier's `envelope`.** the loudness over time. for a bell:
  attack 0.002, sustain 0, decay 2 to 5, release nearly as long.
- **`feedback`.** makes the last operator buzzy without spending another
  operator. 0.3 is a growl. 0.8 is close to noise.

## analog, number by number

- **`shape`.** `saw` has every overtone: bright, the usual start. `square`
  has every other one: hollow, like a clarinet or an old game console.
  `triangle` is nearly a pure tone: for sub basses and soft flutes.
- **two oscillators with `detune`.** set one at minus 8 and one at 8 and
  the two drift against each other, which sounds wide and alive. past 20
  it sounds out of tune.
- **`octave`.** a second oscillator at minus 1 adds weight under the
  note; at 1 it adds an edge above.
- **the filter's `cutoff`.** the most important number. 300 is dark and
  muffled, 1000 is warm, 3000 is bright, 10000 is wide open.
- **`resonance`.** 0 just darkens. 0.5 adds a nasal peak. 0.9 whistles
  at the cutoff.
- **`follow`.** at 0 high notes are as dark as low ones and get lost. at
  0.5 to 1 the filter opens as the notes go up, and every note has the
  same character.
- **the filter's `envelope` and `amount`.** a filter that opens at the
  start of each note and closes again (attack 0, decay 0.2, sustain 0.2,
  amount 2 to 3) is a pluck or a "wow". one that opens slowly (attack
  0.5) is a swell.
- **`drive`.** 0.3 thickens. 0.6 is distorted and aggressive.
- **the channel's `envelope`.** loudness over time, as for an fm carrier.
- **`poles`.** 2 is smooth and polite. 4 is the acid filter: with
  `resonance` 0.8 to 0.9 each note says "wow" as the filter closes, and
  near 1 it whistles.
- **`accent` and `snap`.** what a loud mark in the score does besides
  loudness. with `accent` 1.5 a note marked `@ff` opens the filter an
  octave and a half further, and with `snap` 0.5 it closes twice as fast:
  the accented notes bark. this is half of what makes an acid line move.
- **`glide`.** how long a slide (`>` in the score) takes. 0.06 is the
  classic slither. 0.2 is a swoop.
- **`overdrive`.** distortion after the filter. 0.3 is warm, 0.6 is
  properly distorted, 1 is a wall. the note still swells and dies as the
  channel's envelope says.
- **`fold`.** a harsher distortion: 0.1 adds an edge, 0.5 is a metallic
  buzz that changes as the filter moves.

## pluck, number by number

- **`brightness`.** 0.3 is a finger on a nylon string. 0.9 is a pick on
  steel.
- **`damping`.** 0.1 rings bright for a long time, like a harpsichord or
  a steel string. 0.5 goes dull quickly, like a muted or a gut string.
- **`position`.** 0.1 is plucked near the bridge: thin, nasal, twangy.
  0.3 is a normal guitar. 0.5 is the middle of the string: round and
  hollow, like a harp.
- **`decay` and `release`.** low strings ring longer than this number at
  the bottom and die sooner at the top if `damping` is high. a `release`
  of 0.2 damps the string when the note ends; 2 lets it ring.

## modal, number by number

- **`set`.** `bar` is bright and glassy with overtones far apart: a
  glockenspiel, a music box. `marimba` is woody and plain. `bowl` is a
  hum with a shimmer over it. `plate` is dense and clangorous, a small
  gong. `tube` is a church bell: its overtones do not agree with its
  note, and it sounds grave and a little sour.
- **`hardness`.** 0.2 is felt: only the hum. 0.5 is a rubber mallet. 0.9
  is a metal beater: every overtone, a bright ping at the front.
- **`decay`.** 0.5 is wood. 2 is a metal bar. 6 is a bowl or a bell.
- **`release`.** leave it out and the note rings however short it was
  written, which is what a struck bar does. give it 0.2 and each note is
  stopped by hand, which is tighter and drier.

## the room and the repeats

- reverb `decay` 0.8 and `size` 0.3 is a small room: it just takes the
  dryness off. `decay` 2.5 and `size` 0.7 is a hall. `decay` 6 and `size`
  1 is a cathedral, and fast music turns to mud in it.
- `damping` 0.2 is a cold stone room. 0.6 is a warm wooden one.
- send the lead and the chords to the room (0.3 to 0.6). send the bass
  very little (0 to 0.15) or the low end blurs. send drums a little.
- a delay of `"3/4"` with `feedback` 0.4 fills the gaps in a sparse
  melody with its own echoes. on a busy melody it is clutter.

## how loud each channel should be

you cannot hear the balance, so start from this and let `check` and the
peak tell you the rest: lead `volume` 0.6 to 0.9, chords 0.4 to 0.6, bass
0.6 to 0.8, drums 0.5 to 0.9, `master` 0.35 to 0.45 for four channels. a
ringing sound (long release, lots of reverb) adds up: lower the master.
if the pair is refused for clipping, the error gives the peak; divide the
master by it.

## seven instruments that work

each is a whole voicing with one channel, so it can be checked on its
own. to use one, copy its channel into your voicing. a test in this
package renders every one of these and measures what this page says
about it.

### a glass bell

an fm bell. two carriers and two modulators in `pairs`. the modulators
are at ratios that are not whole numbers, so it clangs; they die faster
than the carriers, so it starts bright and settles to a pure ring.

```json
{"name": "glass-bell", "machine": "studio", "channels": [
  {"kind": "fm", "role": "lead",
   "fm": {"arrangement": "pairs", "operators": [
     {"ratio": 1, "level": 0.8,
      "envelope": {"attack": 0.002, "decay": 3.5, "sustain": 0,
                   "release": 2.5}},
     {"ratio": 3.5, "level": 0.14,
      "envelope": {"attack": 0, "decay": 1.4, "sustain": 0,
                   "release": 1.2}},
     {"ratio": 2.76, "level": 0.22,
      "envelope": {"attack": 0.002, "decay": 0.9, "sustain": 0,
                   "release": 0.7}},
     {"ratio": 5.4, "level": 0.1,
      "envelope": {"attack": 0, "decay": 0.5, "sustain": 0,
                   "release": 0.4}}]}}]}
```

to change it: raise operator 2's `level` to 0.25 for a harder, more
metallic bell. shorten operator 1's `decay` to 1 for a small chime.

### soft keys

an fm electric piano. one carrier, one modulator at the same ratio, so
the overtones are whole and it has one clear pitch. the modulator fades
to a low level: a bright touch at the start, then mellow.

```json
{"name": "soft-keys", "machine": "studio", "channels": [
  {"kind": "fm", "role": "lead",
   "vibrato": {"rate": 4.5, "depth": 5, "delay": 0.3},
   "fm": {"arrangement": "chain", "operators": [
     {"ratio": 1, "level": 0.8,
      "envelope": {"attack": 0.003, "decay": 1.8, "sustain": 0.3,
                   "release": 0.5}},
     {"ratio": 1, "level": 0.12,
      "envelope": {"attack": 0, "decay": 0.5, "sustain": 0.15,
                   "release": 0.4}}]}}]}
```

to change it: a modulator `ratio` of 2 makes it hollow, like a clarinet.
a modulator `level` of 0.25 makes it bark.

### a warm pad

an analog pad for chords. two saws tuned a little apart, a filter that
keeps them dark, and a slow attack so chords swell in. the long release
carries each chord into the next.

```json
{"name": "warm-pad", "machine": "studio", "channels": [
  {"kind": "analog", "role": "pad",
   "envelope": {"attack": 0.4, "decay": 1.5, "sustain": 0.8,
                "release": 2},
   "analog": {"oscillators": [{"shape": "saw", "detune": -9},
                              {"shape": "saw", "detune": 9}],
              "filter": {"cutoff": 800, "resonance": 0.2,
                         "follow": 0.4}}}]}
```

to change it: `cutoff` 2500 makes it bright strings. an `attack` of 0.01
makes it an organ.

### a harp

a plucked string, plucked at the middle, bright, ringing on after the
note.

```json
{"name": "harp", "machine": "studio", "channels": [
  {"kind": "pluck", "role": "lead",
   "pluck": {"brightness": 0.8, "damping": 0.2, "position": 0.5,
             "decay": 5, "release": 2}}]}
```

to change it: `position` 0.12 and `damping` 0.1 is a harpsichord.
`brightness` 0.4, `damping` 0.5 and `release` 0.2 is a muted bass.

### a singing bowl

a struck bowl with a soft mallet. a long hum with two or three
overtones over it that are not in tune with it and fade first.

```json
{"name": "singing-bowl", "machine": "studio", "channels": [
  {"kind": "modal", "role": "lead",
   "modal": {"set": "bowl", "hardness": 0.4, "decay": 6}}]}
```

to change it: `set` `bar` and `decay` 2 is a glockenspiel. `set` `tube`
and `hardness` 0.9 is a church bell.

### an acid line

the bass line of acid house and goa trance. one saw, the four-pole
filter just short of whistling, an envelope that snaps the filter open
on each note and lets it fall shut in a fifth of a second, overdrive
after. it needs the score to do its part: short notes, slides between
some of them, accents on a few.

```json
{"name": "acid-line", "machine": "studio", "master": 0.3, "channels": [
  {"kind": "analog", "role": "bass", "glide": 0.06,
   "envelope": {"attack": 0.002, "decay": 0.5, "sustain": 0.55,
                "release": 0.04},
   "analog": {"oscillators": [{"shape": "saw"}],
              "overdrive": 0.55,
              "filter": {"poles": 4, "cutoff": 500, "resonance": 0.86,
                         "follow": 0.3, "amount": 3.2, "accent": 1.6,
                         "snap": 0.5,
                         "envelope": {"attack": 0, "decay": 0.22,
                                      "sustain": 0, "release": 0.1}}}}]}
```

to change it: `"shape": "square"` is hollower and more nasal. a lower
`cutoff` (250) with a bigger `amount` (4.5) is a deeper squelch. `decay`
0.1 in the filter's envelope is a tight blip; 0.5 is a long "wow".
`resonance` 0.95 screams. `overdrive` 0.9 and `fold` 0.2 is the
distorted, aggressive version.

a bar for it, sixteenths in 4/4 at 138 beats a minute:
`a1:1/4 a1:1/4 a2:1/4@ff> a1:1/4 c2:1/4 a1:1/4> a2:1/4 g1:1/4@f
a1:1/4 a1:1/4 e2:1/4@ff> a2:1/4 a1:1/4 c2:1/4> d2:1/4 a1:1/4@f`

### a soft kit

four drums: a low kick with a quick fall and a little click, a snare of
tone and hiss, a short closed hat, a tom.

```json
{"name": "soft-kit", "machine": "studio", "channels": [
  {"kind": "drum", "role": "perc",
   "drums": {
     "k": {"tone": 50, "sweep": 30, "fall": 0.05, "decay": 0.3,
           "click": 0.2},
     "s": {"tone": 190, "decay": 0.09, "noise": 0.35,
           "noise_decay": 0.14, "colour": 1800},
     "h": {"noise": 0.3, "noise_decay": 0.045, "colour": 7000},
     "t": {"tone": 110, "sweep": 7, "fall": 0.08, "decay": 0.25}}}]}
```

to change it: for a harder kick, `click` 0.6 and `fall` 0.03. for an
open hat add a drum with `noise_decay` 0.3.

## what this machine does not do

- no wavetables, no samples, no recorded sounds.
- no pianos, no bowed strings, no wind instruments: those need models
  this machine does not have. `fm` gets near an electric piano; `analog`
  with a slow attack gets near a string section, in the way a 1970s
  synthesiser did.
- a loudness mark changes how loud a note is, and not yet how bright.
- one reverb and one delay for the whole voicing.
- the named `modal` sets are simple physics of ideal objects, not
  measurements of real instruments. `modes.go` says where each number
  comes from.
