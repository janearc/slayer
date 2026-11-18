# Plane-front gravitational waves

This is the contract for linearized vacuum waves along one fixed
propagation direction. Plus and cross are independent real strain
profiles on a periodic coordinate in metres. Their time derivatives
are strain per second. Both travel at the defined SI speed of light,
299792458 metres per second, with `omega=c*abs(k)`.

The initial profiles and rates determine waves travelling in either
direction. One complex numerical field packs plus in its real part and
cross in its imaginary part. This is storage for two real polarizations;
it does not make strain an optical phasor. The shared wave propagator
advances both components. Camera requests do not advance simulation time.

The direction is normalized without overflowing its magnitude. A
transverse basis projects the least-aligned world axis into the normal
plane, then forms the other axis by a cross product. For propagation
along positive z this selects x and y. With that basis, the strain is
`h=plus*(p*p-q*q)+cross*(p*q+q*p)`, where the products are outer products.
The tensor is symmetric, transverse and traceless.

A chosen `MaxStrain` bounds `hypot(plus,cross)` at construction and every
query. It must be positive and at most the picked weak-field cap 0.01.
That cap is a model guard, without a rigorous bound on nonlinear error.
A display may magnify the strain separately. It must not feed that
magnification back into the physical field.

`Tensor` returns the metric perturbation, without the identity matrix.
`DifferentialStrain` returns half the difference between its projections
along two normalized arm directions. `Offset` is the first-order tidal
displacement `h*x/2`; it is a local observation, not an integrated orbit.
Invalid geometry, non-finite values and overflowing results are refused.

Construction copies inputs and checks the explicit sample budget before
allocation. Snapshots are independent. Refused queries return no partial
profiles and leave later queries usable. Nil and uninitialized fields
refuse both propagation and tensor construction.

This model does not evolve nonlinear spacetime, generate merger
waveforms or couple the metric to Slayer rays. It supplies plane-front
vacuum profiles and their local tensor observations. The initial tests
pin analytic travel, both polarizations, tensor identities, detector
response, budget, copying, weak-field and finite-value refusals.
[Einstein Toolkit plane-wave equations](https://einsteintoolkit.org/thornguide/EinsteinInitialData/IDLinearWaves/documentation.html).

`OpenMetal` prepares the same packed field on a separate Metal plan.
Plus, cross and both rates advance together in one paired Cauchy batch;
only final profiles return to the host. Local tensor observations retain
float64 arithmetic. Each plan has one caller and an idempotent `Close`.
Closed plans refuse every query, including zero time.

Weak-field refusals preserve the plan for later queries. Invalid device
arithmetic is refused without substituting CPU propagation. The actual
numerical tests use realistic strain scales with explicit float32 error
bounds, so a zero result cannot pass by falling below an absolute floor.
