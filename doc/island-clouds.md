# Weather beyond the observatory

Optional root-level `cloud-layers` describes global spherical shells:

```yaml
cloud-layers:
- low-height-metres: 1200
  high-height-metres: 2600
  patch-scale-metres: 14000
  extinction-per-metre: 0.0012
  coverage: 0.5
  seed: 23
  wind-metres-per-second: [4, 0, 1]
  colour-rgb: [0.95, 0.95, 0.95]
```

These are authored scenery inputs, not Hawaiian weather observations. Heights
are radial offsets from the same declared sphere as the terrain. Patch size,
coverage threshold, wind and extinction are chosen appearance controls. The
field extends offshore without an island-sized bounding oval.

`medium.ShellWeather` combines the existing advected noise with a radial
profile `4*h*(1-h)`, where h runs from zero to one between the shell bounds.
The coefficient four normalizes the parabola's midpoint to one. Both bounds
are clear. Coverage is a noise threshold, not a measured cloud fraction.

The centre and noise coordinates stay in the fixed planetary frame. Absolute
scene time moves the noise phase. Camera-relative translation moves the
volume's placement, not the weather pattern. Terrain and the ocean retain
ordinary visibility; this is not a condensation or cloud-dynamics solver.

The CPU and Metal paths split each ray crossing into at most two occupied
legs. They skip the empty inner ball and retain the far leg when visible.
Camera integration and lamp attenuation both use those intervals. The kernel
allows twice the checked volume count for crossing records.

Metal evaluates the density directly at march points. It does not sample a
planet-sized density box. `gpu.Profile.ShellVolumes` counts these native
fields; they do not contribute CPU boxed-grid sample cells. Integration still
uses the renderer's finite camera and shadow sampling budgets.

Radiance agrees with `1-exp(-tau)` within 0.0002 for homogeneous coverage.
That allowance includes midpoint quadrature and float32 error; it is not a
general precision guarantee. It does not
validate arbitrary cloud shapes, grazing rays, shadow integration, phase
functions, multiple scattering or temporal filtering.

Smooth proxy water, the uniform background and terrain detail transitions
are visible limits.
