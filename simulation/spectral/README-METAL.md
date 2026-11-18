# Fourier transforms on Metal

`OpenMetal` prepares one fixed rectangular grid. Its `Forward` and
`Inverse` methods accept row-major complex values and return a new slice.
The input slice stays untouched. `Close` releases the device buffers and
may be called twice; a closed or failed plan refuses further transforms.
One caller uses a plan at a time.

The dimensions are powers of two, at least two. `MaxCells` is a positive
caller-supplied ceiling on width times height, checked before allocation.
Input length must equal that product, and both components must be finite.
Device indexing and buffer sizes must also fit their checked integer
representations. A zero plan is closed.

For width W and height H, the forward coefficient at (kx, ky) is

```text
sum[y=0..H-1, x=0..W-1] input[y*W+x]
    * exp(-2*pi*i*(kx*x/W + ky*y/H))
```

The inverse uses the positive exponent and divides by W times H. Both
results use the same row-major order. There is no frequency shift.

## Precision

The CPU transform uses complex float64 arithmetic. Metal uses float32
real and imaginary components. The Go API still takes and returns
`complex128`; those boundary conversions do not make GPU arithmetic
float64. Bitwise agreement with the CPU is not promised.

A Metal call refuses a component outside float32's finite range. It also
refuses when the sum of the absolute real and imaginary components would
exceed a quarter of float32's largest finite value. That conservative
headroom keeps the unscaled sums away from overflow before inverse scaling.
Very small components may underflow in conversion to float32.

Small direct transforms and the CPU implementation provide independent
comparisons. The draw tests use a picked relative tolerance of 0.00003,
with an absolute floor of that amount around zero. They also check
round trips, Parseval energy, signs, rectangles and input preservation.
A missing device or a kernel compilation failure fails those tests.

## Work kept on the device

The implementation is a separable radix-two transform. One permutation
places both coordinates in bit-reversed order. Row butterflies run first,
then column butterflies. An inverse scales the final buffer on the GPU.
The plan retains two full complex buffers and one smaller twiddle buffer.
Twiddles are computed once during plan construction and stored as float32.

Every stage is a separate dispatch, with stages ordered in one Metal
command batch. A public transform packs its input, appends every dispatch,
commits once, waits, and converts the final buffer to a new Go slice.
There is no CPU readback between stages.

`metal.Session` has no exported device-name getter. Plan construction
queries the existing `metal.Kernel.Device` interface with the same source
and default device, then closes that probe. Both openings use the Metal
package's shared library cache. This cost belongs to setup measurements.

## Composition with waves and flow

`simulation/internal/metalfft` owns the shader source and dispatch encoder.
Its `Source` and `Kernels` let a fluid solver open one session containing
both Fourier kernels and its real-space kernels. `New` creates an encoder
over that caller-owned session and retains its twiddles.

`Encoder.Append(input, output, scratch, inverse)` appends a transform and
returns the buffer containing its result. It never commits or reads data
back. The caller may append nonlinear work and further transforms before
one commit for its whole step. The caller owns and closes the session.

Output and scratch are distinct full complex buffers. Scratch differs
from input. Input may equal output for deliberate in-place work; otherwise
both work buffers differ from input, which is then preserved. Every buffer
must belong to the encoder's session. Existing `metal.Buffer` does not
expose its owning session, so membership is a caller precondition rather
than a check the encoder can perform.

The public plan uses the in-place form internally while preserving its
caller's Go slice. A failed dispatch or commit ends the public plan.
Buffer validation refusals enqueue nothing. After a dispatch error, an
internal caller discards the batch and session; a partial batch is not resumed.

## Measuring the result

The draw benchmark compares fixed grids using two CPU workers at most.
It reports plan setup, the first transform, and the median of subsequent
warm transforms separately. Warm GPU measurements include input packing,
the commit and wait, and final conversion/readback. They exclude setup.
The CPU result includes its output allocation and full transform.

Use the same dimensions, deterministic input and sample count for both.
Run one case at a time through the shared GPU lane, with the machine's
deadline and memory supervision. Record the build, device, dimensions,
sample count, precision, and whether setup is included. A measured GPU
speedup applies to that case; it is not inferred from kernel timings alone.
