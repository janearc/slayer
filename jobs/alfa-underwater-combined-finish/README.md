# Moonlit Alfa with independent gas scattering

These native GPU still jobs draw the complete 81-metre Alfa and a separate
five-blade stern detail at 1080×960, Ninjafilm half-portrait. They retain the
approved moonlit water, exposure and ten-stop development. There is no local
lamp, city, seabed, fish or kelp. The GPU ocean is explicitly calm and infinite.

`whole-combined` combines screw-origin vapor cores with a separate unresolved
residual gas population. The whole camera keeps Moon azimuth 230°, altitude
40°. `detail-combined` uses the previously documented 301° Moon azimuth for
its closer view. Light strength and altitude remain unchanged. Both gas
populations use explicit HG asymmetry 0.82; bulk water retains its own phase.

The main blade material group rotates around its native factory shaft pivot.
An RPM anchor explicitly shares 300 RPM between geometry and vapor phase.
The geometric phase offset aligns the five blades with the existing cavity
basis. The factory mesh and material pointers remain shared.

Matched controls remove cavities, remove residual gas, remove both, or select
explicit isotropic HG0 for both gas populations. The depth90 control shifts
camera, hull, screw and wash 30 metres deeper without changing composition.

`whole-combined-narrow` is a separate authored wash comparison: radius 0.8 m,
height 25 m, density 0.03/m, starting one metre aft of the screw. The initial
wash has radius 2.4 m, height 30 m, density 0.02/m, starting four metres aft.
These choices are synthetic, not measured submarine gas persistence.

The broad wash has a visible birth-plane edge. Helical vapor cores remain
coherent at this clock; the residual mist is a separate advected density,
not resolved millimetre bubble geometry or a hydrodynamic simulation. Neither
representation establishes physical bubble size, lifetime or acoustic noise.
Original approved images and clips remain unchanged.

The ITTC [cavitation descriptions](https://www.ittc.info/media/11816/75-02-03-032.pdf)
support separating tip cavities and gas populations. NIST documents the
[HG phase model](https://pages.nist.gov/SCATMECH/docs/phase.htm). These sources
support the model choices, not calibration of the authored coefficients.

Run `game run alfa-combined-whole`, `alfa-combined-detail`,
`alfa-combined-narrow`, or the five named `alfa-combined-*` control targets.
Outputs go under `out/alfa-underwater-combined-finish`. Actual shared proofs
were rendered with the clean combined renderer at 403047d7; source, binary,
recipes, raw output and matched controls are retained beside those images.
