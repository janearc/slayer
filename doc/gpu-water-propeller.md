# Named water and native propeller cavities on Metal

The finite calm-water lunar route now composes explicitly wrapped,
fully submerged `ClearCloud` propeller density with named water. `ClearCloud`
exposes its immutable density recipe after its clock is applied. The GPU
packs the existing 27-word propeller card directly; it allocates no density
grid or particle geometry.

The kernel orders occupied cloud midpoints front to back, matching CPU
`throughClear`. Each gap accumulates water single scattering and Beer
attenuation.

Each occupied cloud cell accumulates its albedo times the same refracted lunar
source and water phase function, with cumulative outgoing water/cloud
attenuation. The lunar source also includes incoming cloud Beer. All admitted
clouds lie strictly below the interface, so its air source leg cannot intersect
a cloud. Opaque blockers remain tested on both source legs.

The existing `clear-cloud-water-step-mode` selects legacy per-gap quadrature
or explicit ray-spacing. This is a quality choice, not an exact optimization.
Zero cloud density equals the bare-water GPU pixel exactly; the old full
water integral is not applied a second time. Refracted secondary lunar-water
legs use the same passage rather than losing their water source contribution.

This first shared subset remains finite, flat and axis aligned. It refuses
sky quadrature, arbitrary cloud fields, spatial paint/emission, clouds crossing
the interface or leaving its footprint, and infinite continuations. The CPU
validator admits this finite-flat Moon subset in addition to its existing
infinite CPU route. No generic unsupported continuation has been enabled.

The later [exact calm infinite-plane subset](gpu-calm-infinite.md) admits only
a mathematically identical unbounded plane; displaced continuations remain
unsupported. This exception does not silently flatten an authored wavy scene.
