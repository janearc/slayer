# gate composition implementation record

Base: rc-v0.1.5 at 94748beb. This records the staged gate fixture and its
checks. For the current recipe entry point, see [sets](sets.md).

A set can read an inline or external composition, resolve numeric parameters,
validate it and stage its parts. External composition bytes are included in
job snapshots and render keys. The city parts include bodies, legs, decks,
storey bands, seeded window ribbons, stay fans, ramps and helix roads.
Grounded contacts, connected roads, slope and helix clearance are checked;
these checks do not calculate strength or check collisions with unrelated
parts. Full-width joins and support contacts are checked, but collisions
with supports and other objects are not. Helix coordinates and tessellation
are bounded; the clearance reserve is not a formal floating-point proof.

The gate fixture is in `set/testdata/gate-composition.yaml`. Its 60 m deck
slabs carry centred 58.5 m road corridors, continuing across grounded
approach ramps with grades below 4%. Its shores are
simple blocks, not terrain. Upper-deck stays connect to the grounded tower;
the lower deck hangs from the upper deck through repeated vertical members.
The stays attach at z = ±30 m, at the edges of the 60 m deck, rather than
at the outer edges of the 100 m inhabited band. Each 1.5 m wide hanger
projects 0.5 m onto the nominal deck width and extends 1 m into the adjacent
side band. The contact checks do not establish clearance through that band.
Upper stay chords project about 0.6 m onto the nominal deck width and
road-facing window ribbons project 0.05 m. Their checked extents lie outside
the declared 58.5 m corridor at both void heights. Validation does not perform
a general vehicle sweep or establish clearance around other parts. Window
geometry is not a pixel-identical copy of the hub-bridge sketch.

The updated Ferruccio working views are
`gate-composition-proof-current/220-clear.png` and
`300-clear.png` outside the repository. Both are Metal widget-tree wireframe
exports, not OS screenshots or full renders. The 300 m source is a copy of
the 220 m source with only the void parameter changed. Both used the same
Game 5 binaries stamped `e698eb5e`; the fixture was committed at `a68bebd2`,
with no production Go changes between those commits. SHA-256: Slayer
`344442de8e1ef6af0bab29d298c1f9161699855ec917c2561b6ae7b70833d6a9`;
Ferruccio `02ad78585bea73a9c9baf4a7eed2f6be9d851b57f2360bc83ada3a21c2e96c99`.


Focused composition, squid, gate, job snapshot and Ferruccio schema tests
passed through the Game test lane, including space-scene staging. These checks do not establish the whole
project's coverage, mutation or anti-test gates. No full review or release
gate has been run.

For the profile migrations that accompanied this work, see the
[constructor hierarchy](manufacture-constructor-hierarchy.md) and the
[historical review](constructor-hierarchy-review.md).
