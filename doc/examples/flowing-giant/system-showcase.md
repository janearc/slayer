# Amber and two moons: camera study

`system-showcase.yaml` authors one generated flowing giant, two fixed rocky
Mercury-template moons, and three pinhole cameras: `disc`, `second`, and
`close`. The previews use the same constructor inputs and saved weather
moment for all views; camera-dependent face rendering does not by itself
establish whole-field visual coherence. Moon positions and radii are
invented for composition. The second moon has radius 4,000 km at
[-30,000,000, 90,000,000, 25,000,000] east/north/up metres; the eclipsing
moon has radius 7,000 km at [-79,730,000, -79,730,000, 41,040,000]. The
giant's authored weather, gas columns and zones are in the set.

The scene inherits the deliberate split lamp normalization in
`balanced-moon-study.md`: the ordinary-surface lamp's gray tint is 8e-8,
while the separately prelit giant face does not inherit it. This is an
artistic balance, not one calibrated sun. The giant retains 20% of its
prelit contribution where blocked as an invented shadow fill. The boundary
remains that of an opaque single-direction sun, without physical penumbra
or modeled indirect light. See that study for the measurements and source
references, including pinned NASA Cassini eclipse imagery and PBRT's film
and imaging chapter. The fill, colors, placements and weather values are
artistic choices, not a reconstruction of Saturn or its moons.

Preview jobs use the current GPU renderer at 768 × 768, two rays per axis,
exposure 25000 and 50° width. They retain raw linear PFM beside PNG. The
views are proposals until independently inspected.

Azazel jobs 402–404 produced the observed images. The jobs now write
`proof.png` and `light.pfm` in
`out/2026-11-27-system-showcase-{overview,alternate,cloud-close}/`.
An independent visual researcher found both overview and alternate show
the giant and two separate moons, with the alternate's phase and diagonal
limb adding depth. Both are centered and static, however, the moons are
small and the bands are soft and repetitive; the overview does not clearly
connect the shadow to its source moon. The close view conveys scale but
reveals block-like, blurred weather with no resolved vortex or cloud volume;
the shadow sits near the edge without its moon in frame. These images
demonstrate multiple cameras on one system and expose limitations, but are
not yet compelling investor hero imagery.
