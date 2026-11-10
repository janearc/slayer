# burning-trousers

An independent test harness for Slayer's ray tracer. Expected results come
from physics, published data, or the harness's own arithmetic -- not from the
renderer under test.

The harness covers several kinds of evidence:

- Mocks isolate the renderer from Slayer's scene-building code.
- Fixtures compare simple scenes with closed-form answers: inverse-square
  lighting, slabs of gas, a white furnace, and mirrors.
- Random scenes test invariants: finite, nonnegative light; no energy from
  nothing; deterministic results; and invariance under a change of scale.
- Regression, performance, and mutation checks expose changed results,
  expensive paths, and gaps in the tests.

The harness now lives inside Slayer. [INSIDE.md](INSIDE.md) describes the
boundaries that preserve its independence.
