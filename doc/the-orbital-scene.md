# The orbital scene

Two Culture orbitals surround a star inside a Dyson swarm, all suspended
in a nebula. One scene unit is 1,000 km.

try the [orbital portrait job](examples/jobs/orbital-portrait.yaml) from
the checkout root:

```sh
game build slayer
game run slayer -- assess doc/examples/jobs/orbital-portrait.yaml
game run slayer -- draw doc/examples/jobs/orbital-portrait.yaml --out out/orbital.png
```

the job requests Metal. `slayer list scenes` lists `orbital` parameters,
including the far view and the options that place architecture and terrain
on the rings. [the lantern](the-lantern.md) describes the central swarm.

## Scene components

- **Orbitals:** two adjacent bands sharing an axis and rotating in opposite
  directions. The snapshot specifies a radius of 1.84 million km, a 24-hour
  day for approximately one g, width 192,000 km, and walls 1,000 km high.
- **Floors:** hexagonal biomes painted by `biome.Make`: seas, coasts,
  deserts, forests, downhill rivers, lakes, peak volcanoes, and cities
  beside water. City emission can be warm or neon.
- **Inner walls:** a black-marble pattern of city lights facing the gap.
- **Lantern:** a 9,000 km star at 9,000 K surrounded by twelve panel rings.
  Rings are spaced 1.2 stellar radii apart, from six to about nineteen radii.
- **Ring motion:** widths narrow from nine radii inward to two outward;
  orbital periods shorten from 60 to 15 seconds. End-over-end tumble
  periods shorten from 40 to 10 seconds.
- **Panels:** worn edges, holes, and dents, with blue-black glass on both sides.
- **Beams:** two poles offset 15 degrees from opposition, inclined 50 degrees
  to an axis spinning every 1.5 seconds. The axis precesses around a
  35-degree cone every five seconds. Axial light is 30 times the star's.
- **Nebula:** a custom wake medium, cleared along the orbital path with
  denser material outside and a ridge at the channel edge; density `4.5e-5`.
- **Camera:** the named scene defaults to a 90-degree pinhole. the far
  view defaults to 3.6 million km at 25 degrees above the orbital plane.
  the portrait job supplies its own lens and viewpoint.

## Zappy

The beamed star in its swarm at the middle of the far view is zappy, the
lantern grown up. It is no longer built inside the orbital: it is defined
once in package `lantern` (`lantern.Zappy()`, its values in
`lantern/zappy.yaml`), and the far view places it from there. The named
scene `zappy` draws it alone. The orbital's hoops are set from zappy's
beam axis.
