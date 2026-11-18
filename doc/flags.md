# Flags: how a picture departs from exact

Use [Rendering modes](modes.md) for complete science and fast-film
workflows. [The optimization catalogue](optimizations.md) also covers
automatic savings that have no flag, and the separate compositor plan.

By default Slayer uses the configured tracer's own answer, with optional
lighting shortcuts off. This is not a promise of equal bytes between
backends or every native process. Named lighting shortcuts are recorded
with the job. Other choices, including native traversal, source grids and
section drift, are explained in [modes](modes.md).

This page covers lighting flags and declared scene fantasy.
Measurements for fast light are in
[fast-light.md](fast-light.md); the keys are in
[job/README.md](../job/README.md), "flags and shortcuts".

A flag is not a quality. `quality` (`exact` or `preview`) says how much of
a frame is drawn. A flag says how its light is found, or what the scene
states.

## The table

| Flag | What it changes in the picture | What it is for | Never use it for |
|---|---|---|---|
| none (exact) | nothing: the reference | anything measured, catalogued or judged; anything not said to be otherwise | |
| `vibes` | turns on every shortcut below | films to watch, going to a video site where they are compressed | a measurement, a smoke, the catalogue, a picture cited as evidence |
| `fastlight` | turns on the light shortcuts, `lamp-store` and `sky-looks` | the same, for scenes with mist or smoke | the same |
| `lamp-store` | mist and smoke are lit from a store of each lamp's light; a shadow cast into mist is as soft as a cell (12.5 m on the cloud sea); about 2 percent of the picture's light differs | a faster frame of mist or smoke under a few lamps | a picture whose mist shadows are the subject; more than sixteen lamps (refused) |
| `sky-looks` | the sky's sixteen looks from a surface point are shared among a pixel's rays; sky-lit shade is a little rougher within a pixel; no change at one ray a pixel | a faster frame at four rays a pixel or more | one ray a pixel (nothing changes); without `lamp-store` (refused) |
| `fantasy` (a set's block) | the scene says values the physics does not; its keys are below | pictures that are made to be looked at | a claim about what a place looks like; anything where the measured value is wanted |
| science workflow (no single switch) | adds independent checks and replay where supported | claiming a picture is right | combining with any shortcut (refused) |

## exact

With nothing said, a picture is the exact tracer's. The caches keep
exact results only. A flag said anywhere changes the job's canonical form
and so its picture key and its frames' key: a picture drawn with a flag
is never taken for one drawn without.

## vibes

`vibes: on` is the flag for every flag. Today it turns on `lamp-store`
and `sky-looks`, through `fastlight`. It exists so that a film to watch is
one word.

It does not make anything else cheaper: `vibes` with `fastlight:
off` disables these light shortcuts unless an individual setting overrides
it. Additional mist-once, early-stopping and dither policies remain
proposals.
The tracer already has its own faint-transmission stopping threshold;
turning vibes off does not remove that model limit.

Why it is allowed: a film going to a video site is compressed there, and
its audience is not measuring it. It is never the default. A picture drawn with it says so in its
record.

## fastlight

`fastlight: on` turns on the light shortcuts. It is GPU only. The CPU
tracer stays exact and refuses it, as it does a rig's lens, a named
scene, `volume-lighting: cached` with `lamp-store`, and a proof. It may be
said in a job, in a set under `scene`, and on the door as `--fastlight
on|off`.

A shortcut is decided by the first that says anything: its own word under
`shortcuts`, then `fastlight`, then `vibes`; each is read from the command
line, then the job, then the set. So `fastlight: on` with `lamp-store:
off` draws exact mist light.

## lamp-store

For each lamp and each boxed medium, a grid of cells (128 along the
longest side) holds the share of the lamp's light that reaches the cell's
middle past everything. Mist and smoke read it in place of a march
toward every lamp at every step. Surfaces do not read it and should not:
a surface point lies between cells inside its own trunk or ground, which
hold no light.

A store is not kept on disk.

Refused by name: more than sixteen lamps (the exact tracer picks among
them at each point), and beside `volume-lighting: cached`.

## sky-looks

The tracer takes sixteen looks at the sky from every surface point a
camera ray meets, with a shadow ray along each, at every ray of a pixel.
That is the larger part of a forest frame. Shared, the
sixteen are dealt round the pixel's rays: at four rays a pixel each takes
four, at sixteen each takes one.

It needs `lamp-store`, since the two are drawn by one mode of the kernel.

## fantasy

A set's `fantasy` block says what the scene states that the physics does
not. Its keys are `lake`, `lakeglow`, `moon`, `moonglow`, `giants`,
`close` and `closeglow`, the names of hillside's improv flags; a flag
given on the command line stands over the set's key.

Hillside prints a line for each key: `fantasy: NAME = VALUE departs from the measured or
derived value`, and says when a flag overrode the set's. `fantasy` is a
statement about the scene, where `vibes` is about how it is drawn; they
are independent. `sets/moonlake.yaml` is the worked example.

## strict: science

There is no switch with this name. It is three things, used together:

- `fermat` selects 64 pixels from a hash of a raw image's bytes and
  replays their computation on the exact tracer; each must match bit for
  bit. Run it with `hillside -prove` and the original rendering
  parameters.
- the harness's checks (`burning-trousers`), which compare the renderer
  with closed-form physics and independent expectations;
- a signed binary. (Nothing in the tree makes or checks one today; it is
  a requirement, not a feature.)

No shortcut may be combined with it. A proof beside `vibes`, `fastlight`
or any shortcut is refused by name, since the replay is the exact
tracer's and would not match.

## Where a flag is said

| Where | How |
|---|---|
| the command line | `slayer draw JOB --vibes on --lamp-store off` |
| a job | `vibes: on`, `fastlight: on`, `shortcuts: {lamp-store: off}` |
| a set | `vibes` and `fastlight` under `scene`; no shortcuts |
| the schema | each flag's line is its documentation: the editor shows it on hover and `slayer --agent-help` lists it; a test holds that every flag in the schema is in `--agent-help` |
