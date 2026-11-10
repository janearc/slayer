# Guide-star beams

A sodium laser makes a small artificial beacon high in the atmosphere.
[Keck's LGS description](https://www2.keck.hawaii.edu/optics/lgsao/lgsbasics.html)
places sodium near 90 km, with a variable layer thickness, and describes
excitation at 589 nm. Both Keck telescopes use laser guide stars.

That is the instrumental inspiration. A yellow line in a picture does not
amount to an adaptive-optics simulation.

`emit.Beam` describes the light on the way out. Its launch radius is the
1/e^2 intensity radius. The radius grows linearly with axial distance, at
the tangent of the declared divergence half-angle.

Peak irradiance falls with the squared ratio of launch radius to local
radius, keeping the transverse Gaussian integral constant. This authored
geometric envelope omits diffraction, atmospheric refraction and focus at
the sodium layer.

The CPU and Metal tracers use the same envelope. Illumination and shadows
use rays parallel to the beam axis, including when the envelope diverges;
that approximation is intended for narrow guide-star beams.

Neither tracer gives the beam an emissive skin. A surface or participating
medium has to scatter the light before it appears in the picture. In
vacuum, a side view of the beam stays dark.

The launch and far planes are excluded. Reach is an axial length, not an
altitude above sea level. Reaching a particular atmospheric layer requires
accounting for launch altitude and pointing. Setting reach to 90 km alone
does not do that calculation.

RGB and spectral strengths mean peak launch-plane irradiance in this
approximation. They are not a point lamp's unit-distance strengths. A
monochromatic spectrum can have signed linear RGB coordinates outside the
display gamut; the beam preserves those coordinates until development.

State display exposure, beam radius, divergence and scene haze beside each
render. A conspicuous beam is not evidence of measured laser power, sodium
return flux or guide-star magnitude. Sodium fluorescence and the
adaptive-optics control loop remain separate work.

Volume marching refines the existing sample cells near a ray's conservative
beam bounds. The authored budgets live in `render/beamstep/choices.yaml`:
padding by three maximum beam radii, four target samples per launch radius,
and at most 64 subdivisions of an original cell.

Cells outside those bounds keep their original midpoint. Gaussian tails
still receive ordinary integration; the bounds are not a light cutoff.
Merging several beam intervals may refine their gaps too. Each ray gets
its own bounds calculation rather than trusting a neighbour's pilot ray.

[Layered atmosphere](atmosphere-layers.md) separates a low cloud deck from
thin air above it. The four-beam study uses both fields explicitly.

The scene's `lasers` block declares outgoing beams. Each launch gives an
`origin`, `pointing`, `radius`, axial `reach`, `divergence`, `wavelength` and
`display-strength`. See [the prototype scene](guide-star-mist.yaml) for syntax.

`display-strength` scales the spectral line before observer conversion; it
is not a target RGB luminance. The prototype's brightness and film exposure
are chosen to show the beam and moonlit surfaces. They are display settings,
not measurements of an observatory's laser or a photographic exposure.

Origin offsets use world east, up and south, in metres. With an `object`,
they are added to that named object's placed origin. They do not inherit
its random yaw. Pointing is azimuth east of north and altitude above the
horizon, in degrees. Divergence is the geometric half-angle.

These origin fields differ from legacy camera coordinate arrays, which use
[east, north, up]. The explicit axis names are intentional. The reader
requires every offset and both pointing angles, including explicit zeroes.

The prototype uses `scene.atmosphere` to select a prescribed weather field
over the terrain, including the observatory clearing. It gives a weather
name, `extent: terrain` and an explicit scalar scattering `albedo`.

The layer follows the terrain height, with the weather's declared depth,
coverage and wind. It shares the canopy mist's appearance controls without
requiring a tree under each point. Its extent includes the terrain bounds
and the declared depth above the highest ground.

This example replaces canopy mist with regional mist. Attaching both adds
their extinction. Neither field simulates transpiration, transported water
or condensation; the [redwood research](redwood-atmosphere.md) still governs
what we can say about those relationships.

## Keep a common boresight

A scene can name a shared direction and have several launches use it:

    laser-pointings:
      science target: {azimuth: 180, altitude: 70}

A launch then says pointing: science target. Without an offset, every
launch using that name has exactly the same world-space axis. This keeps
separate launch positions from acquiring unrelated pointing directions.
The original explicit pointing: {azimuth, altitude} form still works.

An optional guide-star-offset gives horizontal and vertical quantities
in arcsec, both explicit even when one is zero. They are gnomonic
sky-tangent offsets about the boresight. Horizontal follows increasing
azimuth; vertical follows increasing elevation.

This remains well defined at the zenith because the boresight azimuth
fixes the tangent-frame roll. A simple change in azimuth would shrink a
horizontal offset at high elevation. The reader instead projects the
tangent plane onto the viewing sphere.

It resolves names after reading the whole scene, so declaration order
does not select a different direction. The [four-beam
deck](guide-star-deck.yaml) now uses this form.

Its selected 64-arcsecond off-axis radius comes from [ESO's GALACSI
wide-field layout](https://cdn3.eso.org/sci/facilities/develop/ao/sys/galacsi.html),
pinned as eso-galacsi-layout. It is one supported instrument pattern,
not a universal rule for four lasers.

These offsets specify sky directions, not a finite-range beacon plane.
They omit launch-baseline parallax, sodium-layer focus and refraction.
Parallel world-space axes can converge in a perspective image; that
vanishing point is different from the old scene's degree-scale splay.
