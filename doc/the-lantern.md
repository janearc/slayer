# The lantern

A star inside a Dyson swarm casts shafts of light through gaps between
its panels, like a pierced tin lantern. `slayer/lantern` builds the scene
component; `swarm` renders it alone, and the orbital scene places it
between the rings.

the [lantern portrait job](examples/jobs/lantern-portrait.yaml) uses the
named `swarm` scene. run `slayer list scenes` to find the settings a job
can pass to `swarm` or `zappy`.

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

## Zappy

The lantern with beams at the middle of the orbital's far view has a
name: zappy. It is a star of 9,000 km in
twelve shells of blue-black glass panels that tumble, with two beams
from its poles that spin every 1.5 seconds and precess every 5.

Zappy is defined once, in package `lantern`, beside the lantern it grew
from: `lantern.Zappy()` gives its values, read from `lantern/zappy.yaml`
with who picked each, and `Place` puts it in a world with a clock for its
beams. The orbital's far view builds its middle from it, and the named
scene `zappy` draws it alone, in a thin haze so the beams show.

## Scene configuration

A `Lantern` value is built in Go. `swarm` exposes settings such as
`-orbits`, `-panels`, `-wide`, `-wear`, `-glass`, `-spread`, `-fastest`,
`-narrowest`, `-from`, and `-at`. The orbital scene's far view places zappy.
There is no set-file form of a lantern; [the set format](sets.md) says what
a set file can say.
