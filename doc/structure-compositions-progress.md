# Composition and taxonomy: work in progress

Base: rc-v0.1.5 at 94748beb. The checked-in design describes intent;
this note records what has actually been built and what remains.

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

Superseded numbered squid, Hatz, Purussaurus, pliosaur and walrus studies
and their tests were removed after checking in-repository callers; the active
models remain. `NewHumboldtSquid` selects a checked profile and uses the
shared squid and mollusc construction. Checked squid profiles are usable in
set recipes and catalogue descriptors, with an explicit phase in radians; the
catalogue includes a Humboldt squid. Profiles can vary size, mantle radius,
head width, fin span, arm and tentacle lengths, and coat. They share the
illustrative pulse and do not validate other species' anatomy.

Checked profile books now supply the migrated seasonal plants, conifers,
cacti, ferns, ray-finned fish, birds, pinnipeds and the Allosaurus theropod
study. Their migrated scenes and catalogue cards select profiles; the former
species kind names are not public factory or set aliases. A private fixed
candidate list preserves the default scavenged cluster's historical seeded
draws without making those labels authorable kinds. City and planted-tower
recipes can also freeze an explicitly supplied tree book. The shared recursive
branching mechanism is separate from plant taxonomy.

This is still an increment, not completion of the proposed tree. The fish
builder covers the current bony ray-finned forms; the theropod scaffold is the
Allosaurus study, not a general dinosaur builder. The orca, pterosaur, crocodylian and other models need further work where
real shared boundaries can reduce duplication while preserving anatomical
variation. Profile YAML and
shared numerical construction policies remain in Slayer, including embedded
defaults; a separate models repository has not been created or populated.
The removed study functions were exported; their only user confirmed the removal.
