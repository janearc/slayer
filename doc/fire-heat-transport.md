# Heat can travel; the fuel stays put

This is a code-only draft. The new CPU and Metal paths have been written
and formatted, but have not been built, checked or rendered yet.

The existing surface model now has an optional, explicitly supplied heat
velocity. `fuelbed.NewAdvected` and `NewMetalAdvected` add the horizontal
advection term in Mandel et al., equation (1), section 2:
https://arxiv.org/pdf/0709.0086 .

Their equation supplies the model term; the first-order upwind discretization
below is our implementation choice.

East follows columns and south follows rows, in horizontal model metres per
second. The caller supplies both. These are heat-transport velocities, not
inferred weather, measured near-ground wind or a coupled atmosphere. Fuel
remains in its cell.

This is still a surface layer, with no crown ignition, moisture, terrain-slope
feedback or firebrand ignition.

## One simultaneous step

For timestep dt, cell width h, diffusion k and cooling A*C, define:

    d = dt*k/h^2
    w = dt*max(vEast, 0)/h
    e = dt*max(-vEast, 0)/h
    n = dt*max(vSouth, 0)/h
    s = dt*max(-vSouth, 0)/h
    c = 1 - (4*d + dt*A*C + w + e + n + s)

Every weight must be finite and between zero and one. The separate fuel
loss weight dt*Cs has the same bound. An excessive velocity refuses the
chosen timestep; it does not silently slow the flow or invent substeps.

Heat is the weighted sum of the old cell and its four old neighbours,
plus heat released by that cell's consumed fuel. Every cell reads the
same old state before the buffers swap. Positive east velocity receives
heat from the west, rather than carrying the hot spot westward.

Diffusion keeps the existing insulated boundary. Advection receives zero
excess temperature from outside the bed, and may carry heat out of it.
Those are different boundary choices on purpose. An open edge must not
keep recirculating its own hot value as incoming air.

First-order upwind smears a transported feature. Taylor expansion of the
chosen one-dimensional pure-advection step gives a leading numerical
diffusion of abs(v)*h*(1-Courant)/2, where Courant = abs(v)*dt/h.
This is a discretization cost, not a measured diffusion coefficient.

At the authored example's 0.5 m/s, one-metre cells and 0.1 s steps, that
term is 0.2375 m^2/s. It can rival the model's physical diffusion and
soften a moving front. Higher-order transport is separate future work;
increasing velocity does not preserve the same apparent front width.

Zero velocity takes the original solver path, including its arithmetic
and Metal parameter layout. The optional Metal entry shares the old
state buffers and coefficient prefix, then appends seven step weights.
It does not change the renderer's volume layout or mist marcher.

## Scene input

An existing `scene.fuel-beds` entry can supply:

```yaml
heat-transport:
  east: {value: 0.5, unit: m/s}
  south: {value: 0, unit: m/s}
```

These example values are authored. Both signed components and their units
are required and finite when the block is present. Omitting it selects
zero flow.

The schema and legacy scene reader describe the same optional block;
the solver checks the combined timestep during scene preparation.

Do not copy the mist's wind automatically. The heat layer is horizontal
and is merely draped over terrain for display. A future physical coupling
must define how atmospheric velocity reaches that layer and how its
horizontal coordinates relate to the world's local tangent frame.

## Review witnesses

`fuelbed/testdata/advection-contract.yaml` gives independent transport-only
steps with fuel absent, so a changed reaction rate cannot hide a reversed
transport direction. It also gives diagonal flow, ambient inflow and outflow
examples.

These are small authored arithmetic witnesses, not experimental fire
observations. They have not been executed yet.

Review also needs actual CPU/Metal agreement over reacting fuel, joint
timestep refusal, coefficient representability, zero-flow compatibility,
rewinds, owned snapshots and resource cleanup on failed submissions.

The proposed float32 acceptance budget is a maximum absolute per-cell
difference of 0.1 K in excess temperature and 1e-4 in fuel fraction, compared
with CPU binary64 at 0, 1, 10, 2 and 10 seconds. The repeated times require a
rewind/replay, not just a forward final-state comparison.

The reacting comparison uses the existing 3 by 3 reference bed, 1 m cells,
0.01 s steps and its original reaction/diffusion/cooling coefficients, with
authored east/south velocity 0.5/0.25 m/s. Transport-only witnesses use a 1e-5
K absolute budget through two steps; CPU arithmetic is held to 1e-12 K there.

These are limits set before execution, not observed errors.

Encoded thermal weights are checked nonnegative and their sum is limited to 1
plus four float32 ulps at unity. Tiny weights can round or flush to zero. This
check does not prove universal conservation or the error budget for arbitrary
grids, coefficients or longer horizons.

Those need their own stated acceptance cases. None of the new comparisons has
run yet.
