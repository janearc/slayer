# Mission census

Wave two began on 15 November, before wave one was complete. The 600-part floor and the separate maturity review still apply.
JIMO and New Horizons have first priority, followed by Cassini and Huygens.

The 0.1.4 weather satellite slate includes GOES-16 (GOES-R series),
NOAA-19 (POES), Suomi NPP (the flown NPOESS Preparatory Project), and the
unlaunched NPOESS operational concept. Choose a dated design for the
operational concept before modeling it.

For spacecraft detail passes, aim for more than 1,000 separately modeled
physical components. Report the component and triangle counts separately.
For imported models, distinguish source meshes and material groups from
physical components. The census still reports its historical 600-part floor
and leaves the maturity review pending.

`game run catalogue-census` compares the cited mission seed with the live
factory and emits JSON. Optional kind names select rows, for example:

```sh
game run catalogue-census -- jimo new-horizons cassini huygens
```

The source file is `cmd/catalogue/mission-census.json`. It begins with 60
science missions and concepts drawn from NASA, ESA and JAXA records.
The weather satellite entries extend that seed.
This is an initial survey, not a complete world census. Crewed vehicles,
Earth-orbiting fleets, other agencies, failed missions and additional
unlaunched studies remain to survey. The report states that limit.

Each candidate has a stable proposed kind, priority, launch status,
references and any design caveats. Unlaunched concepts remain eligible:
both JIMO and Darwin are included. A mission name does not register a
factory kind. Missing models have a null part count and a `not-built`
review; existing ones have measured counts and a `pending` review.
Crossing 600 never automatically grants maturity acceptance.

Cassini and Huygens are separate spacecraft records. Mission extensions
that reuse one vehicle should not silently become duplicate hardware.
Variants with materially different spacecraft configurations need explicit
identities. Other contributors' branches also need checking before a
missing model is rebuilt: the Venera work is outside this checkout.

The first geometry work needs these primary-source decisions:

- JIMO: choose one dated concept configuration. The 2012 NASA overview
  shows reactor/shield, Brayton conversion, main boom, heat rejection,
  spacecraft bus, xenon structure and electric-thruster pods. It is a
  conceptual vehicle; do not combine incompatible earlier studies.
- New Horizons: reconcile the flown vehicle against the prelaunch
  environmental-impact statement. Model its triangular bus, antenna,
  instrument apertures, propulsion and RTG as separate physical assemblies.
- Cassini: identify orbiter hardware and the attached-probe configuration.
- Huygens: distinguish cruise, entry/descent and landed configurations.

The initial references are entry points for research, not dimensional
certification. Model dimensions must enter the pinned research ledger
before geometry uses them. New models still require closed geometry,
material evidence, component accounting, tests and both presentation views.
