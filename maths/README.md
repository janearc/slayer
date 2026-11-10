# maths

Shared geometry and linear-light arithmetic for the tracer and film model:
points, directions, rays, bounding boxes, orientations, subspaces, RGB
light, and smooth interpolation. The package reads no files and keeps no
internal state. Coordinates are right-handed, with y up and negative z forward.

The baseline is Slayer's `maths` at `3b6549b` on `slay-onto-planet`, derived
from Lamb's `maths` at `ad2fff9` after a line-by-line review. The initial
copy preserved those files byte for byte; subsequent commits describe
their changes.

`maths/hexes` and `maths/noise` are separate packages. This package imports
neither.

Fields and package values such as the axes, `Up`, and `Identity` remain
publicly writable. Source comments identify that contract. Encapsulating
those values is a separate API decision.

Go builds the package without additional build tools. External tests in
`maths_test` use Rapid and Puzzlemouth's numbers package, with fixtures in
`testdata`. The package has no interfaces requiring mocks.
