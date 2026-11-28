# Authored ocean interface validation

This synthetic scene checks that a Slayer set can stage a finite procedural
wave surface and draw a clear interface around a sphere. The water index,
wave inputs, light and colours are test choices, not measurements of a
particular location.

The surface bends and reflects rays, but this scene does not author an
absorbing or scattering water volume. Its flood light does not test the sun's
path through a water interface. The field is finite, vertically displaced
only, and inherits the height-field marcher's grazing limitations.

Set `SLAYER_CACHE` to an absolute cache directory.
Assess without drawing with `slayer assess validation/ocean-interface/job.yaml`.
Submit `slayer draw validation/ocean-interface/job.yaml --out /path/to/interface.png --fresh`
through [Azazel](../../doc/azazel.md) to make a new still.

The 256 × 144, one-ray CPU study and exact render receipt are in
`renders/frames/2026-11-16-studio-authored-ocean-interface/`. The initial
assessment failure caused by unquoted source values is preserved there too.

## Synthetic attenuation comparison

`attenuation.patch` adds one optical field to the same scene: synthetic
RGB extinction `[0.6, 0.2, 0.05]` per metre. Apply it to a copy of
`scene.yaml`, with `job.yaml` beside the copy; the unpatched scene is the
control. The CPU comparison is retained at
`renders/frames/2026-11-16-studio-authored-water-optics/`.

The patches and images demonstrate camera-path extinction through the wave
interface, not measured water, in-scattering, sun refraction or a finished
ocean shot. An unbounded ray inside the attenuating medium loses its background light;
without in-scattering this makes much of the surface dark.
