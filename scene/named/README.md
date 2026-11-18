# named

Scenes built in Go, by name. Four of slayer's scenes were built inside their
commands, in `package main`, from flags, so nothing but the command could
draw them. This package holds their construction so a job can say
`scene: {named: orbital, with: {gas: 0.25}}` and get the same picture the
command draws.

This package is a bridge, agreed as one. Each named scene is owed a set
file; when the last of them has one, `named` goes.

## What it holds

| Name | What it is | One unit is |
|---|---|---|
| `swarm` | the lantern: a star inside a Dyson swarm, in a haze | the star's radius |
| `zappy` | the lantern grown up: the beamed star in its swarm, alone, in a thin haze | a thousand kilometres |
| `orbital` | two counter-turning orbitals in a nebula, round the lantern from far | a thousand kilometres |
| `ring` | a mirror ring round a flaring magnetar | a thousand kilometres |
| `moonrise` | the full moon rising behind a tree | a metre |
| `magnetar` | a real magnetar at its true size: its field drawn, how far the danger reaches, a giant flare, and zappy beside it | a kilometre |

| `weather` | a planet's air stirred by storms, drawn as three maps | none: no world |

`All` lists every scene of both kinds, the traced first, each once.
`Traced` lists the traced ones. `Get` is a traced scene by name and
`GetMaps` a scene drawn as maps; each refuses the other kind's name by
saying which kind it is, and a name no scene has by listing the names.

Zappy itself is defined in package `lantern` (`lantern.Zappy()`); the
orbital's far view and the `zappy` scene both place it from there.

## Two kinds

A traced scene (`Scene`) builds a world and an eye, and a renderer makes
its one picture a frame. A scene drawn as maps (`Maps`) has neither: built
from its with block and a `MapsAsk` (renderer, cache, spacing, a writer),
it is a `Mapper`, carried to each moment of scene time to make several
pictures there, each with the file name its command gives it, and it says
which of them a film shows.

Both kinds are `Listed`: a name, one line, the
parameters and `Check`.

## weather

The barotropic weather of package `weather`, as `cmd/weather` runs it. Its
parameters are the command's flags: `across`, `rows`, `limit`, `radius`,
`day`, `drag`, `relax`, `rate`, `storm-width` (the flag `-wide`),
`strength`, `bands`, `seed`; and `show`, the map a film shows (`spun`,
`bands` or `aged`, default `bands`), and `pace`.

A job's times stop at a day of scene time and the weather runs for weeks,
so a second of scene time is `pace` of the weather's: an hour by default.
`scene-duration: {value: 840, unit: s}` is 840 hours, the command's default
run of three million seconds, a little over. The command asks for a pace
of a second.

At each moment it makes `spun-CLOCK`, `bands-CLOCK` and `aged-CLOCK`,
CLOCK the weather's seconds, eight wide, as the command always named them.
It runs on the gpu, in float32, where there is a metal device and no
renderer is said, and says so; `renderer: cpu` runs it in float64. Its
moments are kept in the weather cache, `weather/` in the cache home
(`SLAYER_CACHE`), which hillside shares.

## A scene

A `Scene` has a name, one line saying what it is, the unit its lengths are
in, the renderers that can draw it (`cpu`, `gpu`), its parameters, and its
defaults: film size and rays, lens width, time and tone, as its command's
flags default to them.

`Build` takes the parameter values as a job's `with:` block decodes them, a
`map[string]any`, and a writer for what the scene says as it builds. It
checks every key and value first and refuses by name:

- a key the scene has not, listing the ones it has;
- a value of the wrong kind, saying the kind it wants;
- a value outside its bounds, saying the bounds.

A key left out takes its default. `Check` is the same checking without
building, for an editor.

What `Build` gives back holds the world and the eye, and is told the time
of each frame before it is drawn (`Set`). Zappy, alone or as the
orbital's far view's middle, has lamps that keep their own time, its
beams; for the rest `Set` does nothing, and `Timed` says which. The
package holds no mutable state: the beams' clock is the built scene's
own.

## Parameters

A parameter has a name, a kind, a default, bounds where the old command
enforced them, and one line of help. They are data, so a lister or an
editor can show them.

| Kind | Takes |
|---|---|
| whole number | an integer, or a number with no fraction |
| number | any number |
| switch | true or false |
| duration | Go's duration text (`24h`, `1.5s`), or a `time.Duration` |
| one of a list | one of the names listed |

## Commands and parameters

Each command parses its flags as before and calls `Build`. Where a flag's
name would collide with a job's own key, the parameter has a plain name of
its own.

| Command | Flag | Parameter |
|---|---|---|
| swarm | `-wide` | `band-width` |
| swarm | `-from` | `distance` |
| orbital | `-from` | `distance` |

The rest keep their flag's name. Flags that are film, time or output
(`-w`, `-h`, `-n`, `-at`, `-out`, `-gpu`, the orbital's `-eye`, `-look`,
`-up` and `-wide`, the ring's false-colour flags, moonrise's `-wide`,
`-preview` and `-stops`) stay in the command.

## Playing zappy

Zappy has channels: each thing in it that music can move or pulse,
declared as data in `lantern/zappy-channels.yaml` with what it does, its
unit, its range and its rest. `Built.Channels` lists them; `Built.Play`
holds their values for the moment about to be drawn, after `Set` has told
the scene the time.

Channels left out stand at their rest, and with every channel at rest the
world holds exactly what was placed, so the picture is zappy unplayed,
byte for byte.

A phase channel (the beams' spin and precession, a shell's tumble) is
turns added to where the clock has the motion. The clock keeps its rate;
to place a motion outright, hold the rate at nought and the phase is all
of it. A shell's own channel, off its rest, wins over the all-shells one.

What a frame costs: the star, the lamps and a shell's tumble and breath
are swapped in place and cost nothing to speak of. A shell's band and
opening make its panels again, about half a second a shell, seven for all
twelve; what is made is kept, so a value held costs it once and a ramp
costs it every frame.

Not offered: each beam on or off by itself, since
the gpu kernel draws both poles of a polar emitter alike, and changing
that is `gpu/trace.metal`'s; the star's glow does not follow its
temperature, only its light does.

Package `cuesheet` plays these channels from music; a job says
`scene: {named: zappy, play: SHEET}` with a tune as its `sound`.

## The magnetar

A real magnetar at its true size, to stand where zappy stands when the
music stops. Zappy is a model of one made hundreds of times too large;
this is the real one, and every number it draws by is read from the
research data (`research.Value`), each with its source and page, or
computed here from those by a stated model.

- The star: SGR 1806-20's, a sphere of the review's lower radius, 10 km,
  at the middle, pale and very hot. A neutron star glows in X-rays, not
  in light we see; the colour is a rendering choice and the sentences
  say so. It has no surface features.
- The field, drawn: a dipole's lines, `r = L sin^2(theta)` about the
  magnetic axis, as thin glowing tubes, the axis tilted off the spin
  axis, the whole turning once in the star's period, so the lines sweep.
  It is a drawing of something no camera sees.
- The danger: a hoop in the magnetic equator at each distance where the
  field has fallen to a named strength, the dipole's
  `B(r) = B_surface (R / r)^3` there; the quantum field where the vacuum
  changes how light travels, the field that squashes atoms, the
  strongest steady laboratory magnet, a fridge magnet, the Earth's own.
  Along the poles each distance is a quarter further (2^(1/3)).
- The flare: the 2004 giant flare's light curve on the star's glow: a
  spike that rises in a millisecond and decays in 65 ms, over in its
  true 0.2 s, then a tail that pulses at the spin period for 380 s,
  shown sped up by `flare-speed` and said to be. How bright each part is
  is picked for the picture; the times are measured.
- Beside zappy: zappy, its beams off, at its own 9,000 km, beside the
  real star, so a camera that sees zappy fill the frame sees the real
  one as the dot it is; the camera must close in by the ratio of their
  radii to see the real one as large.

`Set` turns nothing (the turning is the things' own motion) and runs the
flare: the star's glow is told the light curve at that moment. The
sentences zappy says over it are in `magnetar-script.yaml`; each names
the value behind every figure in it, and a figure with no registered or
computed value behind it is refused.
