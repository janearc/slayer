# Fast light

Fast light is an opt-in way to draw a scene with mist or smoke in less
time, for pictures that are for watching and not for measuring. It is a
device path. The CPU tracer stays exact and refuses it.

This note records what was measured on 12 November 2026, what was built
from it, what was built and taken out, and what the measurements say
about the work nobody has done yet. Every timing here is from one
machine on one day, with a dozen other renders sharing its GPU. The same
frame drew in 41, 54 and 116 seconds of trace at different hours. Read a
ratio between frames drawn one after the other; do not read the seconds.

## How it is asked for

Three keys, in a job and, the first two, under a set's `scene`:

    vibes: on
    fastlight: on
    shortcuts:
      lamp-store: off

`job/README.md`, "flags and shortcuts", is the contract. In short: a
shortcut is one named approximation; `fastlight` is the flag for the
light shortcuts; `vibes` is the flag for every flag. A shortcut is
decided by the first that says anything: its own word, then `fastlight`,
then `vibes`, each from the command line, then the job, then the set.

With nothing said the picture is the exact tracer's, byte for byte what
it was before these keys existed.

There are two light shortcuts today: `lamp-store`, and `sky-looks`, which
needs it. The last section says what `sky-looks` does.

A picture drawn with it says so. The job's canonical form, and so its
picture key and its frames ledger's key, hold the flags as said; the
copy of the job beside the frames says them; hillside prints a `fast
light:` line; `gpu.Profile.LampStoreBytes` is not nought.

Refused by name: on the CPU; through a rig's lens; for a named scene;
beside a proof (`-prove`), which replays pixels on the exact tracer;
beside `volume-lighting: cached`; for a world with more than eight
lamps, where the exact tracer picks among them at each point; and
`sky-looks` without `lamp-store`.

## What was measured first

The cloud sea's hero camera, 1920 by 1080, four rays a pixel,
`gpu-native`, at the hero's own detail, base f812bd7e. Seconds of trace
on the device, one run of twelve frames in turn:

| Frame | Seconds |
|---|---|
| whole | 54.2 first, 41.3 last |
| mist off | 23.7 |
| mist on, its lighting returned as nought | 21.5 |
| mist on, no shadow ray from a mist step | 27.1 |
| mist on, no march toward the lamps from a mist step | 47.1 |
| both of those off | 25.3 |
| lasers off | 44.3 |
| trees off | 31.3 |
| trees off, mist off | 0.7 |

The rest of a frame is small beside the trace: encode 2.1 s, a warm
compile 0.9 s, the hierarchy 1.1 s, the finish 0.3 s. Building the world
on the CPU was 12 to 20 s under that day's load.

What the table says:

- Stepping through the mist costs nothing that shows. The frame with
  the mist's lighting returned as nought is the frame with no mist.
- The mist's lighting is about half the frame, and nearly all of that
  is one shadow ray from every step toward the moon, through five
  thousand trees: about 800 million of them, some 35 nanoseconds each.
  The ten-step march toward each lamp is about 5 s.
- The other half is there with no mist at all, and is gone with no
  trees. The last section of this note is about that half.

## The method, and why not cascades

Radiance cascades (Osborne and Sannikov 2024, arXiv 2408.14425; the
two-dimensional proof is `improv/cascades`) solve for light arriving
from every direction at every place. They earn that where light comes
from the medium itself or from many places: fire lighting its own smoke
from within, a glowing nebula, light scattered more than once.

The cloud sea has five lamps, each with one direction at any point. The
question at a step is not "what arrives from everywhere" but "how much
of this one lamp gets here", five times. A store of that answer for each
lamp is smaller, simpler and nearly exact: its only error is how coarse
its cells are. It also keeps each lamp's direction, which a mist that
scatters forward needs and a store of summed light would lose.

So fast light, today, is a store for each lamp. Cascades would still
add:

- fire and embers as area sources lighting smoke, in place of the few
  point lamps a flame is cut into now;
- a gas that glows, such as a nebula, which the store does not touch;
- light scattered twice or more in thick mist or cloud, which nothing
  in the tree draws at all.

None of those is the cloud sea. Three dimensions and Metal for cascades
are a project of days and are not started.

## The lamp store

`gpu/reach.go` says the design beside the code. In short:

- For each lamp and each boxed medium, a grid of cells over the
  medium's box, 128 along its longest side. A cell holds the share of
  that lamp's light, each channel, that reaches the cell's middle past
  every thing and through every volume: the kernel's own `blocked` and
  `clear`, asked once a cell and not once a step.
- For a narrow beam, a table of 1,024 places along its axis in place of
  a grid. A beam a metre wide is narrower than any cell. Its profile
  across is still worked exactly at each step.
- It keeps what arrives, not what is scattered toward the eye. So it
  does not depend on the camera or on which ray of a pixel is asking,
  and the phase function is applied at the step from the lamp's true
  direction.
- It is baked on the device at the start of each frame that encodes the
  world, by two kernels that share the trace's buffers. A frame that
  only moves the camera (`gpu.Frames`) draws from the store already on
  the device and bakes nothing.

On the cloud sea the store is 16.6 MB (two media, one moon's grid in
each, five beam tables) and its bake is 0.19 s of device time, 1.4 s of
wall on the per-call path, which copies every input to the device for
each grid.

No store is kept on disk. A bake is under a hundredth of the frame it
serves, so a disk store with its key, cap and invalidation would save
nothing that shows. If a scene ever bakes for seconds, the key to use is
the door's (`job.PictureKey` and its inputs), narrowed to the medium,
the lamps, the things and the build.

What it gets wrong: a shadow cast into the mist is as soft as a cell,
12.5 m in the cloud sea, so a shaft of moonlight between two trees is
blurred where the exact march shows it.

What it leaves alone: surfaces; a gas (`gas.Of`), lit by its own
`gas_gathered`; a puff, a wisp, a shell, a ring and a wake.

Surfaces do not read the store and should not be made to. A surface
point lies between cells whose middles are inside its own trunk or under
its own ground. Those hold no light, so the surface would darken in
patches that follow the grid, and a tree's shadow on the ground would be
12.5 m soft.

## Measured against exact

The cloud sea, the hero's own detail, the two frames drawn one after
the other:

| | Trace | Whole command |
|---|---|---|
| exact | 115.8 s | 136 s |
| lamp store | 46.6 s | 89 s |

About two and a half times on the trace. Earlier the same day the same
exact frame traced in 41 to 54 s; at that hour the store's frame would
be about 20.

The difference between the two pictures, in the `gpu` package's own
measure, over the raw light: summed 1.98 percent of the exact picture's
light; the worst channel of any pixel 7.2 percent of the picture's
brightest. Developed to 8 bits: mean 0.8 of 255 levels, largest 19,
13.6 percent of pixels more than two levels apart. All of it is in the
mist between the trees.

The tests that hold it, in `gpu/reach_test.go`, on a smoke column under
a point lamp, a distant one and a beam:

- `TestFastLightTracksExactLighting`: the summed difference from the
  exact picture is at most two percent of the picture's light, the
  cached smoke grid's own budget, and the worst channel at most five
  percent of its peak. Measured: 0.0055 percent and 0.36 percent. The
  budget is the cloud sea's size, not the fixture's.
- `TestFastLightComesTogetherAsItsCellsGrow`: 0.031, 0.014 and 0.0055
  percent at 32, 64 and 128 cells.
- `TestFastLightIsNotTheExactPictureAndMovesWithItsLamps`: the store is
  read, and is the frame's own.
- `frames_test.go` draws the mode cold, warm, as a film and across
  camera moves, each held bit for bit to the per-call picture.

## The kernel and the layout

The header grew in the same release, for blur and depth of field and not
for fast light: from 256 to 336 bytes, five rows appended
(`gpu/exposure.go`). Fast light added no field and no row. Two existing
fields gained meanings, and `gpu/layout_test.go` reads them back from
the device:

- `Header.lighting`, four whole numbers. Before: x, the volume a cached
  smoke bake is for, and three noughts. Now also: for a store bake, the
  volume in x and the lamp in y; for the trace, a bit a lamp in z for
  the lamps that keep grids, and in w one more than the first number of
  the beams' tables among the boxes.

  With `sky-looks` the trace's x also carries the samples along a
  pixel's side, which `sky_share` in the kernel reads when w is above
  nought; with fewer than two it takes all sixteen looks. All nought
  with no store.
- `Volume.lighting`, four whole numbers: a grid's cells and first
  number, or minus one. With a store in the header it is the first wide
  lamp's grid, the other lamps' following it in order; without one it is
  the cached mode's source grid, as before.

The kernel gained `reach_stored`, `reach_along`, `reach_within`,
`reach_of`, a branch in `scattered_direct`'s lamp loop, two conditions in
`scattered` (a lopsided mist, which never reads a grid, and the store),
`sky_share`, and two bake entry points, `reach_grid` and `reach_beam`.

## Built, measured at nothing, and taken out

`disc-shadows` (commit 13adcc0e holds it). The exact tracer sends eight
shadow rays from each lit surface point across the moon's disc, about 66
million a frame. The shortcut sent one ray a camera sample toward one of
the eight ways, the samples of a pixel spread through the eight, so the
soft edge was kept and was grainier: 3.0, 0.9 and 0.19 percent from the
eight-ray picture at 1, 4 and 16 samples a pixel.

It saved nothing. With every surface shadow ray removed outright, the
no-mist frame went from 30.8 and 31.6 s to 32.7 and 34.2 s. At 35 ns a
shadow ray, 66 million are about 2 s, under that day's drift.

## What a frame depends on

Each stage of a frame reads only some of the scene. A stage whose inputs
have not changed since the last frame could be kept. From the kernel:

| Stage | Reads | Kept today |
|---|---|---|
| encode the world | things, media, lamps, the time | by `gpu.Frames`, when only the camera moved |
| hierarchy over the meshes | triangles, where they stand | by `gpu.Frames`, until they change |
| density grids of boxed media | the medium, the time | re-made with the encode |
| lamp store | media, lamps, things | on the device across camera moves |
| first hit of each sample (`first`) | the camera, the things | no |
| surface shadows (`unhidden`) | the first hits, lamps, things, media | no |
| the march to the first hit (`through`) | the camera, the first hits, media, the lamp store | no |
| a mirror's bounce, a clear thing's legs | all of the above | no |
| finish, develop | the samples, the print | no |

`gpu.Frames` already answers "did the world change" with the caller's
generation number and "did the camera move" by comparing headers. Each
store below would add a question of the same kind.

## Not built, and where each goes

**The first-hit store.** For a frame whose camera and things are those
of the last frame, keep each sample's first hit (distance, face, normal)
on the device and skip `first`. On the cloud sea `first` is everything
left after the lamp store, so for a locked-off camera this is the rest
of the frame; for a moving camera it is nothing.

It needs a second buffer of the sample grid's size held between frames,
a `gpu.Frames` rule beside `cameraOnly` ("nothing a camera ray can hit
moved"), and a test that a still camera's film is bit for bit the film
without it.

**A picture for each lamp.** For a still camera and a still scene whose
lamps only change level, a frame is a sum of one stored picture a lamp.
The light show does this by hand (`improv/moonlake/passes`, `cueshow`).
In the tracer it is the sample grid drawn once a lamp with the others'
strength nought, kept as raws, and a `finish` step that adds them by a
cue sheet. It adds nothing to the cloud sea, whose lamps do not change.

**The forest as a coarse medium.** A grid over a stand holding how much
foliage is in each cell, built once from the real trees, so that a
shadow ray entering the stand walks cells and never meets a needle, and
the lamp store extends into the canopy. Its case is the look and not
the time: the shadow rays it would replace are the cheap ones. What it
must have:

- an albedo for foliage, sourced or picked and said to be, so a cell
  scatters some of what it stops and crowns glow on their lit side
  (single scattering from the store's light in the cell; a second
  bounce would read the neighbouring cells, and should say so);
- one stopping rule for every scene: a ray ends when what is left of it
  is under a threshold. The tracer has that threshold already, `faint`
  in `gpu/parts.go`, one thousandth, the same for every scene, and
  `through` stops on it. A walk through foliage cells should use the
  same number and no depth.

Per-scene numbers a fast path would inherit, found while reading: none
sets how far a ray goes. `strides` (48 steps across a volume), `shades`
(10 toward a lamp) and `faint` are constants of the tracer. A set's
`volume-illumination: samples` and its grid are per scene, for sky light
in a volume. `march_steps` (256) bounds a height field's march.

**Water.** Today water is a surface with nothing under it.

A clear material bends rays by Snell's law and splits them by Fresnel
(`Material.Through`, in the tracer and the kernel); a wavy surface is a
height field (`shape.HeightField`); `metal.WaterLibrary` and `oceantex`
hold the sea's optics (index, Fresnel, slope statistics) and its
textures.

`doc/gpu-muscle-design.md` says it plainly: clear water absorbs nothing
on this path unless the scene holds a volume, and neither tracer reads
`oceantex`'s band absorption. The light show's lake
(`improv/moonlake/mirrorlake`, `mistlake`, `seawater`, `seamatte`) is
laid over a finished raw as a matte painter would, outside the tracer.

As a member of this family, a body of water is a boxed medium under a
clear surface, with cells in the lamp store. What it needs that mist
does not:

- the store's bake must cross the surface: lose Fresnel's reflected
  share, bend the lamp's direction by Snell, then march in the water;
- extinction that depends on colour, a band at a time as the tracer
  carries colour (`Volume.tint` already holds three), from published
  coefficients for pure water and for natural waters, named in
  `sources.yaml`; red is taken first, which is why deep water is
  blue-green;
- the camera's leg under the surface already marches volumes
  (`through` on each refracted leg) and already stops on `faint`, which
  in clear water is metres and in turbid water is centimetres, with
  nothing set per scene;
- a test with a known answer: a slab of water over a white floor at
  several depths, where the floor's colour follows Beer's law down and
  back, held against the CPU tracer.

The lake under the light show is the first scene to show it.

## What is left of a frame: the sky's sixteen looks

After the lamp store, what is left of a cloud sea frame is the cost of
the camera's samples in the forest. An earlier version of this section
read it as closest-hit queries through overlapping instances and put it
down as the hierarchy's debt. That reading was wrong and is withdrawn.

All numbers are the cloud sea's hero camera, no mist, 1920 by 1080, four
rays a pixel (8,294,400 camera rays), `gpu-native`, on the shared GPU,
seconds of trace:

| Frame | Seconds |
|---|---|
| first hit not asked | 0.05 |
| first hit found, nothing shaded | 1.5 |
| as it is, skylight returning nought | 3.7 |
| whole frame | 28 to 31 |
| no trees | 0.7 |

So the hierarchy is fine, software fallbacks included (removing every
one changed nothing: 122,283 of 3,714,698 candidates). Surface shadow
rays are not it either: removing all 66 million changed no timing that
could be read.

The cost is `skylit` in `gpu/trace.metal`: sixteen shadow rays round the
normal at every surface point of every camera sample, about 133 million
a frame, 24 to 27 s of it. An any-hit shadow ray through the forest is
about 35 ns.

The shortcut `sky-looks`, under `fastlight`, shares the sixteen among a
pixel's samples: at four rays a pixel each takes four, at sixteen each
takes one. At one ray a pixel nothing changes, and sky-lit shade is a
little rougher within a pixel. No timing of it is recorded in this note
yet. The exact picture still pays all sixteen at every sample.
