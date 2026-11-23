# Skyglow: haze and city light over a flat world

A night city is not black above the roofs. Its lamps, signs and lit
windows send light up into the air, the air scatters some of it back, and
the sky over the lights is a dim, coloured dome that is brightest low
over the brightest blocks. Slayer works that glow out from the lights.
Nothing about the sky is painted: take the lights away and the sky is
black.

This page says how to ask for it, what it draws, where it is wrong, and
what the GPU path would need. The package's own comment, in
`skyglow/doc.go`, has the model and its error in full.

## Asking for it

Say a flat layer of air in the set. A set with no terrain and no globe
may say an atmosphere layer with `extent: world`, which is the version 2
schema's word, and over no ground it means a layer of air over a flat
world:

```yaml
scene:
  light: moon
  volume-illumination: {model: isotropic-single, sky-samples: 2}
  weather:
    haze:
      kind: prescribed-mist
      extinction: {value: 0.0039, unit: m^-1}
      depth: {value: 300, unit: m}
      coverage: 1
      seed: 1
      asymmetry: 0.7
  atmosphere:
    layers:
      night air:
        weather: haze
        extent: world
        albedo: 0.9
```

The layer is a flat sheet of air from the ground, or from `base`, up to
`depth`, thick at its base and thinning to nothing by a smooth step. It
needs:

- `coverage: 1`: flat air has no patches.
- `volume-illumination`: lamps then scatter by the isotropic phase, which
  is the physical one, and not the legacy factor. A set that says
  neither is refused, by name.
- a world unit of one metre.
- one layer.

A set that says no flat layer is drawn exactly as before.

The extinction of a hazy night is about 3.9 over the visibility in
metres (Koschmieder): 3 km is 0.0013 per metre, 1 km is 0.0039. The
albedo is the share of the extinction that scatters; 0.9 is a city's
haze. Both are the set's to say, as a weather is.

## What it does

1. Every glowing surface and every point lamp in the world is gathered
   into bins 25 m across, each shining from its middle.
2. The air is cut into a grid of 32 by 10 by 32 cells over the lights and
   a unit of optical depth past them. The light that reaches each cell
   straight from the lights is the first order. Each cell then scatters
   some of what it took, in every direction, to every other cell, which
   is the second order, and so on. It stops when an order adds under a
   hundredth of the whole.
3. The camera's march through the air adds, at each step, the albedo
   times the glow there, as it adds the albedo times each lamp's light.
   A lamp in the air also scatters its own light along the ray, with the
   haze's forward lobe, and that is the halo round it.

The progress line says how many sources were solved, the grid, how much
of the scattered light each order held, and what the orders past the last
are likely to hold.

The haze takes Angstrom's law with an exponent of 1.3, so that blue is
met more than green and red less. That number is picked, from the range
of 1.0 to 1.8 continental haze is measured to have.

## Where it is wrong

- Emitters are binned. The glow is right a few bins from the lights and
  further; closer than that it is smeared. A lamp's own halo is the
  tracer's, not the solver's.
- Orders past the first are taken as isotropic. Real haze throws light
  forward and keeps it nearer where it began; the second order is spread
  wider here than it would be.
- Nothing hides anything: no wall, no ground. Light going down is lost,
  where the ground would give some back, and a panel's back face shines
  into its wall.
- Emission painted by a texture, spots, beams and suns are not taken as
  emitters; the progress line counts them as left out. A beam scatters
  in the air on its own, as a lamp does.
- The air outside the solved region is not there. The region is the
  lights' extent and a unit of optical depth past it, so a camera far
  from the city sees no haze until it is near.
- The glow is worked at one wavelength; the haze's tint reaches the
  picture in the direct light and in the scattering toward the eye, not in
  the order of scattering.

## The GPU

`--renderer gpu` draws flat air and its glow:

    slayer draw sets/neon-street-towers-wide-air.yaml --renderer gpu

The layer is a boxed medium on the GPU, as any mist that is not a puff
is: its density is asked on the CPU at the middle of each cell of a grid
over the solved region, 96 cells along the longest side, and the kernel
reads between them. Its tint by channel and its forward lobe go with it.

The glow is solved on the CPU, as for the CPU tracer, and its cells are
packed after the boxed media's. The kernel reads it between the eight
cell middles round a point and holds the edge value beyond them, the
read of `skyglow.Field.Mean`, and adds the albedo times it to each step
through the layer, where `trace/volumes.go` adds it. A lamp in the air
scatters its own light at each step, as on the CPU.

Where the two differ: the CPU asks the layer's density at each step,
and the GPU reads it between cells. The smooth step's thinning is off by
at most an eighth of a cell's height squared times its greatest
curvature, under two thousandths of the extinction at cells of 13 m in
air 300 m deep.

A glow the kernel has no grid of is refused by name, `the world's glow
field of kind ...`; so is one with no cells, or in a box a float cannot
hold. The GPU does not draw a world without its glow.
