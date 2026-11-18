# Two-dimensional spectral transform

`Forward` and `Inverse` transform a rectangular grid of complex values. The
grid is stored as one slice in row-major order: row `y`, column `x` is at
`y*Width+x`. Both dimensions must be powers of two and at least two cells
wide.

The forward transform uses the negative exponential convention:

`F(ky,kx) = sum f(y,x) exp(-2 pi i (kx*x/Width + ky*y/Height))`.

The inverse uses the positive exponent and divides by `Width*Height`, so an
inverse of a forward transform returns the original grid within floating-point
roundoff. Forward output is unscaled. `MaxCells` is a caller-supplied upper
bound on the number of values; dimensions, overflow, budget and input length
are checked before work is done.

Inputs must contain finite real and imaginary parts, and every computed output
must remain finite. Arithmetic overflow returns an error and no result slice.
Calls do not modify the input slice. Invalid dimensions, a nonpositive budget,
an oversized grid, a wrong input length, or non-finite values also return an
error and no result slice.

This package is numerical infrastructure only. It does not define a physical
field, wavelength, optical phase, or rendering model.
