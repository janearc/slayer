# Scalar angular-spectrum propagation

`Field` represents one immutable, periodic, row-major complex scalar field at
one wavelength. `New` copies the starting samples and prepares their Fourier
modes. `At(z)` returns a new sample slice at a nonnegative distance in metres;
the field's stored state and each returned slice are independent. At zero
distance it returns an exact copy of the starting samples.

The carrier `exp(i*k*z)`, where `k=2*pi/wavelength`, is factored out. For
transverse wave numbers `kx` and `ky`, a propagating mode is multiplied by
`exp(i*(sqrt(k*k-kx*kx-ky*ky)-k)*z)`.

The small difference is evaluated with
the identity `-(kx*kx+ky*ky)/(k+sqrt(k*k-kx*kx-ky*ky))` to retain phase for
small angles. An evanescent mode is multiplied by
`exp(-sqrt(kx*kx+ky*ky-k*k)*z)*exp(-i*k*z)`; the latter factor preserves its
relative phase after removing the carrier.

This is a periodic scalar wave model at a single wavelength, not a vector,
polarization, broadband, or renderer model. `Intensity` squares the magnitude
of each supplied complex sample. Coherent fields must be added before calling
it; adding separate intensities describes an incoherent sum.

The Metal path retains the immutable spectrum, applies a complex transfer
on the GPU, then appends every inverse FFT stage to the same command batch.
Only the final field is read back.

Relative phases are reduced in float64
on the host before conversion to float32 transfer factors, preserving small
angles and avoiding metre-scale optical phase in a float32 shader. This
cost is included in the full-call benchmark. `OpenMetal` refuses an absent
device; it never silently substitutes the CPU.

Each plan has one caller.
A validation refusal leaves it usable; a dispatch, commit or result failure
closes it. `Close` is idempotent. A closed plan refuses even distance zero.

Validation compares resolved propagating and evanescent modes, near-field
aperture diffraction and large-grid fields against float64 CPU results.
It also checks destructive interference, repeated distances, independent
results, invalid distances, lifecycle and finite float32 range. This does
not remove periodic boundaries or turn scalar contributions into photon
trajectories.
