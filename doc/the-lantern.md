# The lantern

A star inside a Dyson swarm casts shafts of light through gaps between
its panels, like a pierced tin lantern. `slayer/lantern` builds the scene
component; `swarm` renders it alone, and the orbital scene places it
between the rings.

This note records the design and its evolution on 2026-10-03. Settings
and proposed data syntax below belong to that snapshot; consult current
command help and package declarations for the live interface.

## Components

- A glowing sphere of `Radius` at `At`.
- A point emitter in `slayer/emit`, with a black-body spectrum at `Kelvin`
  and green-channel strength `Green` at the stellar radius. It implements
  `thing.Lamp` for both rendering paths.
- Optional polar beams, with axial strength `Glare` times the star's light
  and time supplied by the scene's clock.
- `Orbits` rings of panels, with `Panels` setting the innermost count.

## Settings

Lengths are relative to the stellar radius and scale with `Radius`.

- **At, Radius:** Stellar location and size

- **Orbits, Panels:** Ring count and innermost panel count; outer rings add
  panels to keep aspect ratios similar

- **Kelvin, Green:** Spectrum temperature and brightness calibration

- **Turn:** Innermost panel orbital period

- **Wide:** Innermost band width; default 0.4

- **Wear:** Zero for new panels, one for heavily worn panels

- **Skin, Lining:** Outer and star-facing materials

- **Shells.Spread:** Radial ring spacing, beginning at six stellar radii

- **Shells.Fastest:** Outermost period, interpolated outward from Turn

- **Shells.Narrowest:** Outermost width, interpolated outward from Wide

- **Shells.Tumble:** Innermost end-over-end period, accelerating outward like
  orbital turn

- **Beams, Clock, Glare:** Polar emitter, evaluation time, and axial
  brightness multiplier


Wear uses a metal field: slow noise cuts inward from edges, finer noise
frays them, faster peaks make holes, and displacement adds dents. A 32-cell
square grid locates the zero contour for the panel boundary.

Without a material override, panels use warm metal with partial reflection.
`BlackGlass(index)` has a blue cast; `Matte(index)` is matte black.
A lining makes each panel a pair of closely spaced sheets.

## Ring placement and motion

Ring poles follow a golden spiral over a hemisphere: a pole and its
opposite describe the same ring. Each ring gets a hashed initial phase
and a hashed transverse tumble axis.

Rotation about a ring's own pole moves panels along a fixed band, leaving
gaps between bands stationary. Tumbling moves the bands themselves,
preventing one gap from lighting the same patch of orbital floor in
every frame.

## Development history

The 2026-10-02 swarm used 13,200 panels on 60 hash-tilted orbits and a lamp
implemented in the command. The point emitter replaced that lamp, and the
lantern became a reusable component scaled by its star.

The 60-ring golden-spiral version used 0.4-radius bands and roughly 10%
open area, intercepting 87% of the light in `lantern_test.go`. At 720p its
motion read mostly as twinkling, so later versions used fewer, wider bands.
Coverage was measured for 3-12 rings and widths up to five radii.

Wear progressed from dropped cells, to edge cuts with visible steps, to a
smooth contour with controlled fraying. Material tests progressed from
black glass that reflected bright haze inside, through glass outside and
matte inside, to blue-black glass on both sides.

Later versions separated the rings radially, made outer rings narrower
and faster, settled on twelve rings, added tumbling, and introduced
moving polar beams.

## Scene configuration

The snapshot builds a `Lantern` value in Go. `swarm` exposes settings such
as `-orbits`, `-panels`, `-wide`, `-wear`, `-glass`, `-spread`, `-fastest`,
`-narrowest`, `-from`, and `-at`. The orbital scene configures its lantern
inside `farView`.

A proposed set-file representation, a list in the world, is:

```yaml
lanterns:
  lantern:
    at: [0, 0, 0]
    radius: 9
    rings: 12
    panels: 4
    wide: 9
    narrowest: 2
    spread: 1.2
    turn: 60       # seconds
    fastest: 15
    tumble: 40
    wear: 1
    skin: {black_glass: 2.2}
    lining: {black_glass: 2.2}
    star: 9000     # kelvin
    green: 6000
    beams: {polar: {width: 6, soft: 4, offset: 15, tilt: 50,
            spin: 1.5, cone: 35, precess: 5, glare: 30}}
```

This is proposed syntax, not a supported set-file example. Reading it
belongs to the set package and building it to stage; [the set
format](sets.md) says what a set file can already say, and [the orbital
design](the-orbital-scene.md) what a film description would also need.

## Decomposition into reusable primitives

The longer-term design expresses a lantern as a composition:

- An element supplies panel geometry and dimensions.
- Deformation applies wear: bites, holes, and dents.
- Per-face materials supply albedo, gloss, and refractive index.
- Per-element masses sum to a ring mass.
- An arrangement places elements around a circle.
- A binding relates that ring to a massive parent body.
- Gravity derives unconstrained orbital motion from those masses.

Artificially driven rings instead declare turn and tumble explicitly.
Their permitted motion would be bounded by mass and material strength,
with invalid configurations rejected by the engine.

The star remains a sphere plus point emitter; beams use a polar emitter;
reflection belongs to materials. Defining these parts once avoids
reimplementing the same structure in every large scene.
