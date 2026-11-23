# Authored ocean surface wake envelope

`world.ocean.surface-wake` is an optional, shot-scoped height perturbation
applied to the same finite ocean height field that the renderer intersects.
It produces a tapered sinusoidal trough/ridge train aft of an authored craft
reference point, with a Gaussian cross-track falloff. The default ocean path
does not construct or evaluate a wake.

The `origin` is map east/north/up in the same coordinates as `world.ocean.middle`
at scene time zero. `heading` is clockwise from north; `speed` advances that
reference along the heading as scene time increases. A shot that uses a moving
craft must author the same path for the craft and wake.

The current interface does not bind the wake directly to a manufactured
craft's `Thing.Motion`; position/heading synchronization is therefore an
authored scene contract, not an enforced relation.

`amplitude`, `half-width`, `length`, and `wavelength` control the height field
in metres. They are artistic shot values. This envelope is not a fluid solve,
does not model a Kelvin wake or propulsion, and generates no foam or spray.
The wake deforms real ocean surface geometry and is bounded by the ocean's
sampled footprint.

When `shading-normal: spectral-slopes` is selected, the scene must explicitly
set `add-to-spectral-shading: true`. The analytic envelope slopes are then
added to the ocean's spectral slope shading field. A wake that omits this
contribution is refused so the visible geometry and spectral shading cannot
silently disagree.

In other shading modes the geometric height-field normal already includes
the perturbation; setting the spectral contribution is refused.

Metal scene-clock parity and automatic craft-motion binding remain untested.
