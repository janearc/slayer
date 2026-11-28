# Balanced moon study

The moon-shadow diagnostics use an artistic giant face whose spectral
conversion and the ordinary scene lamp are on different brightness scales.
For example, the GPU eclipse raw image at 25000 exposure contained moon RGB
samples around 2.8–3.4 million and giant samples around 0.05–0.3. The
display clipped the moon to white. That comparison is a measurement of this
particular render, not a calibrated photometric relationship.

In `art-eclipse-soft.yaml` and `art-foreground-soft.yaml` the authored sun
lamp's gray tint is 8e-8; in `art-eclipse-bright.yaml` it is 2.4e-7. The
existing precomputed flowing face does not inherit that tint. This deliberately
balances an ordinary moon against a separately normalized giant; it does not
describe the moon's physical albedo or a single physically consistent sun.
The exact tint values are artistic experiments chosen from the measured
linear values. The unchanged single directional source still produces the
same binary hard shadow, without penumbra or scattered light. The GPU path is
used because the earlier CPU pictures were heavily speckled; its flowing
cloud texture is smoother and should not be called equivalent to CPU detail.

NASA's [Cassini eclipse examples](https://science.nasa.gov/missions/cassini/spectacular-eclipses-in-the-saturn-system/)
show a moon casting a shadow on planetary clouds and inform the visual
composition, not this system's invented size or geometry. The authors of
[*Physically Based Rendering*, Film and the Imaging Pipeline](https://www.pbr-book.org/3ed-2018/Sampling_and_Reconstruction/Film_and_the_Imaging_Pipeline)
explain why scene radiance may exceed a display's range; that motivates
retaining linear PFM files as well as display PNGs. It does not justify this
particular split lighting scale or promise that display tone mapping recovers
the clipped image. Those two linked pages are not pinned. Related references
are on the shelf as `nasa-cassini-shadows-on-saturn` (PIA12737, a
near-infrared image of Mimas's shadow) and
`pharr-jakob-humphreys-2023-film-imaging` (the fourth edition). Their
manifest hashes pin the fetched HTML, not linked image assets.

## First observations

Azazel jobs 376–378 rendered these three studies with GPU candidate
`53ce509a7`. The jobs now write pictures and linear outputs in
`out/2026-11-27-flowing-art-{eclipse-soft,eclipse-bright,foreground-soft}/`.
An independent visual researcher saw lit and dark sides on the gray moon at
8e-8 and almost-white but still shaded appearance at 2.4e-7. Foreground
occlusion is clear. The moon remains visibly faceted, while the giant's
black hard-edged shadow is stark; these are diagnostic images, not finished
investor imagery. The lamp balance does not imply a shared physical exposure.

The follow-up `art-eclipse-detailed.yaml` retains the 8e-8 balance, raises
the moon's existing ground tiling detail from 2 to 12, reduces its invented
radius from 12,000 km to 7,000 km to reduce the shadow's dominance, and
increases the image from 640 to 1024 square. These are deliberate artistic
composition choices, not measurements of a real moon. Its appearance must be
assessed from the completed render, not inferred from those inputs.

Azazel job 383 produced the observed image. The job now writes
`proof.png` and `light.pfm` in
`out/2026-11-27-flowing-art-eclipse-detailed/`. The independent
visual researcher saw a smoother moon silhouette and visible irregular
surface shading, although the moon is small and detail is limited. The
smaller hard black oval remains conspicuous, and the wavy belts remain soft.
The composition has more breathing room, but remains an art study rather
than clearly investor-ready. Because radius, tiling and resolution changed
together, these visual differences cannot be attributed to one parameter.

## Artistic shadow fill

`art-eclipse-fill.yaml` repeats the detailed scene with `shadow_fill: 0.2`,
retaining 20% of the prelit flowing face where the directional sun is blocked.
The value is invented for appearance, not derived from the references. It
does not model penumbra, scattering or extra incident light; the edge remains
hard. Zero is the default. Azazel job 386 used GPU candidate `d145b6f9d`,
and its job now writes the picture and linear output in
`out/2026-11-27-flowing-art-eclipse-fill/`. Compared with job 383,
an independent visual researcher found the shadow still legible but less
dominating, with the underlying cloud bands visible instead of a featureless
black hole. The image remains stylized and the edge hard.
