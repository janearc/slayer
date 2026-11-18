# Linear wave propagation

A periodic complex field and its time derivative specify the initial
state. Each Fourier mode obeys `q_tt = -omega^2*q`. Frequencies are
finite, nonnegative radians per second, one per row-major mode. They
remain a separate input so the numerical propagator does not confuse
water displacement with spacetime strain.

For elapsed seconds `t`, exact mode evolution is
`q(t)=q(0)*cos(omega*t)+q_t(0)*t*sinc(omega*t)` and
`q_t(t)=-q(0)*omega*sin(omega*t)+q_t(0)*cos(omega*t)`.
Here `sinc(theta)=sin(theta)/theta`, with value one at zero. The zero
mode may drift. Negative finite time is allowed for reversible evolution.

This is exact evolution of the discrete linear model, subject to floating
point error; it adds no numerical damping or finite-difference time step.

`New` copies both physical fields and the frequency table. `At` returns
independent value and rate slices. At zero it returns exact input copies.
Nil, uninitialized, invalid or overflowing requests return an error and
no partial state. Construction checks the explicit spectral cell budget
before grid-sized allocation. A refused request cannot change the field.

`NewWater` supplies the finite-depth surface-gravity dispersion relation
`omega^2=g*k*tanh(k*depth)`, with `k` the horizontal wavenumber magnitude.
Lengths and depth are metres, gravity is metres per square second, and
values and rates are surface height and vertical rate.

This is a linear,
constant-depth, periodic, inviscid model. It excludes breaking, variable
bathymetry, capillarity and nonlinear coupling to the Navier–Stokes solver.
The shallow and deep limits are checked independently.
[MIT dispersion notes](https://web.mit.edu/fluids-modules/www/potential_flows/LecturesHTML/lec19/node5.html).

Gravitational waves need separate tensor observation. For a fixed
propagation direction, plane-front vacuum perturbations carry plus and
cross profiles obeying the one-dimensional wave equation at the speed
of light.

Their tensor is transverse and traceless. This is linearized
vacuum gravity, without a nonlinear Einstein or binary-merger evolution.
The intended adapter retains both initial profiles and time derivatives.
[Einstein Toolkit plane waves](https://einsteintoolkit.org/thornguide/EinsteinInitialData/IDLinearWaves/documentation.html).

The initial CPU contract is tested against analytic modes, reversible
composition, zero-mode drift, independent modal energy, refusal and
copy properties. The planned Metal path retains both initial spectra,
applies the two-mode combination and inverse transforms in one batch,
then reads back the final value and rate. Neither model advances when a
camera asks for a sample; queries are absolute times from the initial state.

For periodic water with no supplied mass source, the mean surface rate
must vanish. `NewWater` refuses a rate mean larger than a picked
`64*float64-epsilon` times its component L1 sum. Smaller roundoff means
are projected to zero; its initial rate snapshot is that projected field.
The supplied slices remain unchanged. This prevents a zero-frequency
drift from being mistaken for a freely propagating water wave.

`OpenMetal` prepares a separate immutable plan. It retains both initial
spectra and computes the oscillator coefficients in float64 on the host.
One kernel forms value and rate spectra; two inverse transforms follow
in the same batch. The value result is preserved before the rate inverse
reuses shared scratch. One commit precedes both final readbacks.

The backend refuses unavailable hardware, invalid times and arithmetic
outside its checked float32 coefficient and spectrum range. It never
substitutes CPU computation. A rejected input leaves the plan usable;
a dispatch, commit or non-finite result failure ends it. Plans have one
caller and an idempotent `Close`. Closed plans refuse even time zero.
A live plan returns exact copies at zero without dispatching a batch.

Metal numerical checks use separate value and rate error scales from
their Cauchy inputs and the analytic frequency bound. There is no fixed
absolute error floor that could accept an extinguished gravitational
profile at realistic strain. Cancellation is measured against the
contributing field scale. Float32 output can underflow; CPU retains
float64 arithmetic for cases outside this backend range.
