# The equipment of astronomy

A summit needs somewhere to put its tools. These are reusable scenery
kinds, so a crane can move between a mirror-handling yard and a loading
area without becoming part of one observatory's enclosure mesh.

Use any of these as a scene thing's kind:

- mobile-crane: deployed outriggers, lattice boom, cable and hook;
- mirror-gantry: wide lifting beam, trolley and hanging hook;
- scaffold-tower: braced bays, timber decks and rails;
- service-truck, utility-pickup and equipment-trailer;
- shipping-crate and equipment-cabinet;
- generator and cooling-unit;
- access-stairs, work-platform and handrail;
- cable-tray and weather-mast.

For a local specimen book:

    game run catalogue -- render mobile-crane scaffold-tower service-truck \
      --out out/summit-yard

The geometry book is manufacture/yard-equipment.yaml, read by Puzzlemouth.
All lengths are metres. Its closed boxes and rods become ordinary mesh
parts; round members change tessellation across the four detail levels.
The physical envelopes stay fixed.

These are picked proportions, parked poses and display colours. A hook
does not imply a lifting simulation, a scaffold is not a load-rated design,
and opaque cab windows do not simulate glass transmission.

Place equipment with room to work. Leave a truck access lane, keep a
crane's outriggers clear of other equipment, and put handrails beside
platform edges. The factory supplies the pieces; the site YAML supplies
their working arrangement.
