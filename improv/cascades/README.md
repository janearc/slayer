# Radiance cascades study

This 2D experiment compares radiance cascades with a brute-force
reference. Its default scene adapts figure 1 of Osborne and Sannikov's
2024 radiative transfer paper. It computes the same scene and tone map
for all panels. [Radiance Cascades](https://arxiv.org/abs/2408.14425).

The `flatland` package contains the grid, reference ray march, and cascade
solver. The `figure` command in this directory writes a three-panel PNG:

1. Brute-force reference.
2. Basic radiance cascades.
3. Cascades with bilinear correction.

`-bilinear=false` displays the amplified absolute difference in the third
panel. `-diffgain` sets its amplification. `-scene single` compares the
light from one disc against analytic falloff. `-side` sets the power-of-two
grid width and `-out` names the PNG. The report includes ray counts, march
steps, wall time and error outside the opaque discs.

This is a three-band numerical experiment. Slayer's production renderer
uses a separate transport path.
