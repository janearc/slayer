# Project 705 propeller cavitation: evidence and controls

## Physical distinction

Propeller cavitation is water vapour cavities formed when pressure falls below
the vapour pressure and then collapses as pressure recovers.

The Navy chapter describes inception at the blade tips, followed by
trailing-edge and then back-face sheet cavitation as speed rises. It also says
increased depth raises the critical speed by raising ambient pressure.

IMO MEPC.1/Circ.833 describes broadband noise from many growing and collapsing
bubbles, and blade-passage harmonics from sheet and tip-vortex cavity volume
changes.

This differs from a buoyant air or residual-gas plume. The current
`NewCavitationCloud` constructor places discrete spheres; its existing name is
historical and those spheres are not a vapour-cavitation model.

The current `medium.Fizz` field represents a rising, irregular density column.
Keep both as authored residual-bubble or mist approximations. A surface wake or
entrainment field is another separate effect. None of those should be labeled
propeller vapour cavitation.

## Alfa evidence boundary

The local Alfa source supports a five-blade main screw and its model places the
main propeller along craft-local `+Z` at the stern. Blade count is sourced;
diameter, pitch, chord and profile are chosen because the cited source has no
dimensioned drawing.

The model data contains no shaft RPM, blade loading, relative inflow, local
pressure coefficient, water temperature, or vapour pressure. A secondary Naval
Forces News report repeats a K-64 trial value of 300 shaft RPM at 42 knots and
80% reactor power, but gives no propeller pressure data or depth for that
point.

Treat it only as a reported high-speed phase reference, not the RPM for this
60m scene and not a cavitation threshold. No Alfa cavitation-inception
threshold can be derived from the available data.

The physical onset relation is `p_min = p_ambient + Cp_min * (rho * U_rel^2 / 2)`.
Inception requires `p_min < p_vapour`. Depth can supply a hydrostatic ambient
pressure estimate when water density is authored; blade tip speed requires
shaft RPM and radius. But the decisive `Cp_min` depends on blade geometry,
loading, and inflow, which are not available here. Do not invent a threshold
or infer it from vessel speed alone.

## Implemented bounded visual field

`scene.world.vapor-cavities` builds one compact analytic density medium. Its
five filaments are locked to the five-blade main screw and need no particle
mesh or spawned sphere list. Each core is a Gaussian tube around a helical
centerline. The phase uses `2*pi*RPM/60 * (time - distance/slipstream)` plus
one of five equally spaced blade phases. That gives downstream advection at
the authored slipstream speed and repeatable frame-time motion.

The shared constructor and scene field require an explicit `blades` count
from 1 through 16. The bound is a numerical work budget, not a hydrodynamic
claim. The Alfa adapter authors 5; two-blade auxiliaries and other propellers
reuse the same field. Gaussian cores are truncated at three standard
deviations so their evaluated density agrees with their finite envelope.

The optional optical-density breakup is controlled explicitly by
`fragment-scale-m`, `fragment-contrast`, and `fragment-seed`. Contrast 0
leaves the continuous cores unchanged. Contrast 1 progressively separates
them into density packets downstream, without increasing density or creating
particle meshes.

The noise is evaluated in the convected coordinate
`distance - slipstream*time`; attached cores stay intact near the rotor.
The breakup ramp reaches full strength halfway along the active field.

The chosen noise zero-crossing transition is a numerical visual approximation,
not a vapor-collapse or bubble-size measurement. All three controls are
authored in the scene; the generator defaults to a disabled contrast of 0.

The core radius is the effective width of unresolved cavity optical density,
not the radius of one vapor bubble. Individual microcavities remain unresolved.

The set must say `visual-onset: true`; false keeps the volume empty. The
`pressure-deficit-pa` and `recovery-pa-per-m` controls are chosen visual
pressure margin and recovery gradient.

Their ratio caps downstream support, and the density fades linearly with
remaining margin. They are not measured Alfa pressure values and do not
establish cavitation onset. `length-m` adds a second hard cap. The shape is
finite, so the mixed-transport march stays bounded.

This first field represents tip-vortex cores and authored density breakup.
It does not create blade sheet cavities, blade-passage source modulation,
physically resolved collapse fragments, residual
microbubbles, turbulent slipstream, or surface entrainment. Keep the existing
`fizzes` and gas spheres as separate residual-gas approximations, and keep
surface wake effects separate. No one of them is a fluid simulation.

When `clear-water: true`, the stage wraps the field in `medium.ClearCloud` and
the CPU integrates it with clear-water extinction and the shared water source
along each ray segment. The cloud uses the existing water phase function;
there is no independent bubble Mie phase function yet. GPU transport remains
explicitly refused. The rendered field is an authored visual approximation,
not CFD, a pressure solution, or quantitative cavitation prediction.

## Required controls and validation

Controls must carry provenance individually:

- sourced: five main blades, craft-local screw axis, and model geometry fields
  explicitly identified as sourced;
- required when solving onset: shaft RPM, local inflow/slipstream speed, water
  density and temperature/vapour pressure, and a sourced or measured minimum
  pressure coefficient for the loaded blade;
- model-derived: the current Alfa main screw radius (2.1 m) and its axis;
  these come from the chosen local model, whose source documents distinguish
  sourced dimensions from chosen profile values;
- visual-only until sourced: RPM used solely to animate phase, slipstream
  speed, core radius, density, pressure deficit and recovery gradient. A
  visual-only value does not establish real cavitation.

Missing RPM or `Cp_min` must never fall back to an invented Alfa threshold.
There is no physical pressure gate in this first visual field. Vapor cavities
stay separate from residual microbubbles and surface entrainment.

## Sources

- U.S. Navy, *Principles of Underwater Sound*, Chapter 8, §8.8.1.3,
  [Cavitation](https://man.fas.org/dod-101/navy/docs/fun/part08.htm).
- International Maritime Organization, MEPC.1/Circ.833, 7 April 2014,
  Annex §4.2, [Guidelines for the Reduction of Underwater Noise from Commercial
  Shipping](https://wwwcdn.imo.org/localresources/en/OurWork/PartnershipsProjects/Documents/GloNoise-Library/MEPC.1-Circ.833-7April2014.pdf).
- Secondary report of the K-64 test point, [Naval Forces News / Army
  Recognition](https://www.armyrecognition.com/archives/archives-naval-defense/naval-defense-2016/russian-navy-looking-at-ways-to-reduce-the-crew-size-of-its-nuclear-submarines).
- Project 705 source/geometry and provenance: [alfa-model.md](alfa-model.md)
  and `manufacture/alfa.yaml`.
