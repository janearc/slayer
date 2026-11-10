# Hillside's equivalent mutations

The complete pass at 3b78a38 observed these 25 surviving sites, represented
by 22 printed mutation names. The entries in .mutants classify functional
equivalents and the numerical cases described below. They do not skip
execution, count defensive returns as covered, or claim equal running cost.

A printed name can identify several source locations. The original report
also killed necessary guards with those same names. The
[site audit](hillside-mutation-sites.json) keeps every location and outcome.
The final pass must preserve that distinction.

New report-formatting tests exercise single-nanosecond inputs as formatter
data. The old reporting mutations were replaced by the extraction, rather
than credited as killed.

## Checked constructors

prepareFronts repeats constructors after the reader has checked the recipe.
The smoke constructor changes only its base to twice a positive finite front
radius, so its validity conditions still hold.

Spark constructors change only seeds and bounded finite wind; those changes
cannot invalidate their checked source.

The palette refusal needs care. Direct invalid-exposure callers can fail
spark preparation. If its forwarding return is removed, the following field
constructor refuses the same exposure before retaining a front or changing
the scene.

The later field-constructor return is necessary and was killed by the real
invalid-palette test. The audit retains all four locations.

Ordinary fire preparation repeats manufacture.Burn after the reader has
checked the copied recipe, changing only seeds. Its factory returns a body
or field implementing sampled emission, so the subsequent type refusal
cannot occur.

Place checks exposure before spark preparation; the remaining repeated spark
check is therefore redundant at this site. Other placement and lamp-error
returns remain necessary and were killed.

The first-medium assertion follows this concrete factory, not an arbitrary
interface supplied by a test. The fixed ember octahedron and the lunar
disc's fixed positive mean radius also cannot fail their constructors in the
command's initialized state.

Source, palette, atmosphere and face-file failures are different locations
and remain tested.

The ordinary-fire edit from time zero to one nanosecond changes only the
sampled light slice that this caller discards. The stored medium is the
same immutable body or field; actual scene lamps are sampled separately
at the requested time.

## Windows and initializers

The first group-end mutation starts with Start-Duration. The first loop
iteration immediately takes the maximum with Start+Duration, restoring
the original end before it is read. Groups are nonempty, durations positive,
and their checked times do not overflow.

Outer flame-window filters repeat immutable field pieces' own activity
windows. Removing the filter or evaluating its exact expired end adds work,
but the inner empty-piece check prevents any volume or lamp from being
added.

Changing the inclusive start is different: the exact-start regression
requires that real flame and its light.

Outer particle-window filters similarly repeat the emitter's lifetime and
the retained adapter's burn-birth checks. Before ignition there are no
particles. At or beyond burn plus lifetime, every surviving particle's birth
is beyond burnout, so the adapter rejects it.

Returned particle ages stay bounded even at distant duration values.
Removing the outer filter can add evaluation work, without changing
geometry, light or counts.

Entering the smoke block with an empty active list leaves weight zero.
The following positive-weight guard prevents construction, division and
append. Only local initialization work changes.

A temporary fuel-patch Y value is overwritten by the actual ground height
and clearance before any read. Changing that initializer is an exact dead
store. Changing the group anchor or final height is a separate case.

The particle duration comparison from zero to one nanosecond distinguishes
a one-nanosecond burn. This private adapter receives either zero for an
indefinite ordinary source or a checked front burn of at least a millisecond.
It does not restrict the public scheduling library's input domain.

## Occupancy cap

The normalized occupancy is at most one in real arithmetic: the immutable
plan's peak bounds its active count, and each smoothstep share is in [0,1].
Changing the final cap from one to two therefore changes no exact result.

For floating point, let u = 2^-53. Each computed nonnegative share is at
most 1+4u. Near the peak, Sterbenz's condition makes the subtraction exact
and the products round; below x=.75 there is a substantial margin below one.
The nonnegative sum has at most 100000 terms.

Its gamma(99999) bound, the division and final multiplication give a
conservative relative difference below 1.2e-11 between the two caps. Plume
density is linear in yield. Conversion to float32 can differ by an adjacent
representable value; bit identity is not established.

The summation tool is from
[Higham (1993), equations 2.2-2.6, pages 784-785](https://nhigham.com/wp-content/uploads/2023/10/high93s.pdf).

The exact-subtraction condition is restated by
[Lee, Sharma and Aiken (2018), section 5.2, theorem 5.1](https://cs.stanford.edu/people/wonyeol/papers/2018-popl.pdf).

The bound above is a local derivation, not a published Slayer measurement
or a universal image-error bound. Ordinary partial-occupancy scaling faults
remain covered by independent quarter, half and full smoke samples.

## Fuel group coordinates

Five mutations change only a private group's coordinate origin. Each patch
is stored at at-anchor, then placed back at anchor; its represented world
position remains at. Group lamps restore the same origin after forming their
local weighted centre. Solver indices, seeds and strengths are unchanged.

Floating point prevents an identical-bits claim. Legal cells span .25 to 2
metres, with at most 24 cells before a group starts. These mutations can
move a coordinate origin by at most 96 metres for a sign change, 90 metres
for division, or one metre vertically.

Those distances are coordinate changes, not error bounds.

The rerunnable [GPU rebasing test](../gpu/a_09_fuel_rebasing_test.go)
checks 30 authored 64-patch fields: three cell sizes, two strengths and
five shifts.

It checks unchanged production grid dimensions and the existing 1e-4
composite absorption/emission budget on a 37 by 29 by 31 conformance grid.
The observed worst relative differences were 1.20256701377e-5 for density
and 1.2645642624e-5 for emission.

These are numerical equivalences under that existing local conformance
budget. The samples are representative; they do not prove a maximum over
every legal scene, a whole-image JND bound or equal rendering cost.

World-placement regressions deliberately hold actual coordinates and cell
identity rather than specifying a private origin.

## Gates

The source arguments above do not execute unreachable defensive returns. The
separate coverage requirement remains open until its actual final profile is
assessed. The updated source requires its own complete mutation pass;
earlier partials, external stops and isolated replays are not that gate.

Raw results and exact source correspondence remain separate from these
classifications.