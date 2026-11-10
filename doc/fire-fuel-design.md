# A fire with fuel to lose

The first night film used a plane to schedule tree-shaped flames. That is
useful choreography, but it does not explain how a fire spreads. This study
starts smaller: one hot spot in a horizontal layer of surface fuel.

Temperature and remaining fuel follow Mandel et al., *A wildland fire model
with data assimilation*, equations (1-2), section 7.1 coefficients:
https://arxiv.org/abs/0709.0086 . These are their grass-fire proxy coefficients,
not measurements of redwood litter. There is no crown ignition model here.

Section 2 defines diffusivity in m^2/s. The extra K^-3 printed on it in
section 7.1 conflicts with equation (1); we use section 2's definition.
This discrepancy is also recorded beside the pinned coefficient data.

With u = T - ambient, r = exp(-B/u) for u > 0 and zero otherwise:

    du/dt = k Laplacian(u) + A (S r - C u)
    dS/dt = -Cs S r

The CPU reference and Metal stencil use explicit Euler, central spatial heat
differences, and insulated boundaries. Fuel stays put. A bounded fixed step
keeps diffusion/cooling coefficients nonnegative and fuel positive.

The optional [heat-transport draft](fire-heat-transport.md) adds explicitly
supplied horizontal advection through a separate CPU/Metal entry. It has not
been checked yet. Slope feedback, moisture, gas flow and a crown layer remain
open; the transport velocity is not an atmospheric wind solution.

A missing-fuel strip can still transmit heat: it is not an insulating wall.
Snapshots use the last completed fixed tick, with less than one tick lag.
Rewinding resets and replays the initial state. Render order cannot create
extra ignitions. The GPU buffers persist between frames.

The display is deliberately separate. A flame patch's absorption follows
reaction strength, and its height follows the cube root of that strength.
The recipe supplies authored gas temperature and shape; layer-average PDE
temperature is not a measured flame gas temperature.

Remaining fuel darkens a thin surface layer. This visual proxy for
consumption does not model charcoal chemistry. There are no timed burn
windows. This first study excludes smoke transport and firebrand ignition.

Before rendering, check cold fuel stays cold, isolated fuel loses mass and
releases the corresponding heat, and empty hot cells cool without burning.
Check heat reaches neighbouring fuel, a wide fuel break stops the wave,
smaller time steps converge, and CPU/Metal agree.

Refuse invalid dimensions, non-finite inputs, unstable time steps and
requested times outside 0..86400s. Returned snapshots cannot mutate the bed.
The committed tests check these properties. Earlier manual fault trials
have no preserved replay patches and are not acceptance evidence.

Display strength is min(1, S*r/0.35); strengths below 0.002 draw no flame.
These are authored display choices, not ignition criteria. The solver still
evolves every cell. Bounded flame jitter within cells avoids a regular grid
of luminous dots without moving the fuel.

One finite lamp per 8x8 group retains the sampled source power, using a
weighted centroid and authored group radius; local lighting is approximate.
The example uses an authored moon fill so spent ground remains legible.
The render drapes the horizontal model over the DEM without slope feedback.

The optional `surface` choice defaults to `fuel`, which paints remaining
fuel on a thin overlay. `terrain` preserves the existing ground appearance;
it changes the display only. Temperature, consumption, flames and lamps
continue to use the same state, without allocating the diagnostic tiles.
