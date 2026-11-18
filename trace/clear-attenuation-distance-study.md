# Synthetic RGB clear-medium attenuation distance study

`TestClearAttenuationDistanceStudyDraw` uses Slayer's existing named
`thing.ClearMedium` and `trace.RenderWithInitialMedia` APIs. It places the
camera explicitly inside one uniform synthetic medium; no boundary is crossed
and no refractive-index change is involved. The test renders an identical
glowing reference plane in air and at camera-path lengths 0.25, 0.5, 1, 2,
and 4.5 metres through the same medium. It checks the center pixel against

`T(R,G,B) = exp(-sigma(R,G,B) * distance)`.

The chosen RGB extinction values are 0.08, 0.25, and 0.50 m^-1 at Slayer's
red, green, and blue channel wavelengths (610, 550, and 465 nm). They are
synthetic controls, not measurements or a preset for any water. The expected
transmission at 4.5 m is approximately (0.698, 0.325, 0.105). Scene units are
metres (`metresPerUnit = 1`).

The test also renders five identical, uniformly glowing small spheres in one
world at known radial distances 0.75, 1.50, 2.50, 3.50, and 4.50 m, all inside
that same explicitly supplied medium. Their angular positions spread them
across the image. Their apparent sizes change with distance; compare their
color loss as a qualitative composition, while the separate full-frame plane
controls and Beer-Lambert assertions are the quantitative ground truth.

This is camera-path attenuation of the unscattered beam only. There is no
in-scattering: scattered energy is not redirected into the view.

The world has no lamps or illuminated surfaces, so direct-light transport is
absent and is not validated. It contains no clear boundary, Fresnel reflection,
refraction, water volume, underwater site, or real-world coefficients. The
target's self-emission is deliberately independent of illumination so only the
camera path changes its RGB.

The output images are opt-in. Set
`SLAYER_CLEAR_ATTENUATION_STUDY_OUT` to a new directory and run the focused
draw test through Game's tests lane:

```sh
SLAYER_CLEAR_ATTENUATION_STUDY_OUT=/path/to/new-study game lane tests -- \
  go test -tags=draw -count=1 -v ./trace \
  -run '^TestClearAttenuationDistanceStudyDraw$'
```

No custom parser, YAML schema, GPU path, or window is involved. The Go fixture
is the inspectable scene source.
