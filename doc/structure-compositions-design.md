# Structures from parts: design for review

23 November 2026. This proposal records the design at rc-v0.1.5
94748bebcac89e68fb83f52e32e29f285d3ff731. It uses the gate bridge in the
hub-bridge sketch. For the implemented input see [sets](sets.md).
The [implementation record](structure-compositions-progress.md) records
what was staged and checked.

## Boundary

A composition is data read when a set is read. It describes named parts,
measures, placements, connections and roads. Code owns a small vocabulary of
parts and the checks; data owns a particular structure. A set can contain a
composition inline or name a YAML file relative to the set file. Both paths
produce the same checked, immutable construction. The set's existing recipe
and placement path then makes it, so seed, placement and detail work as for
other manufactured things. No new executable or command is needed.

`set.Recipe.Construction` currently selects `manufacture.Kind`, and a
`kind` wraps a `maker` with `made`, `extent` and `seen`. Add a composition
constructor at that boundary, backed by cities' `Design` and its existing
rendering/placement adapter. Do not add every composition to `table()`.
Reading, resolving and checking must finish before staging or drawing; retain
the resolved data, not a path to reread halfway through a draw. The recipe's
digest and every draw/kept-still cache key must include the actual resolved
composition bytes and reader version. A changed included file must invalidate
the output even if the set itself is unchanged. Until that is implemented,
refuse kept-output reuse for compositions rather than show stale geometry.

## A proposed set and composition

These are proposed keys, not current syntax. Lengths are metres; vectors are
local x, y, z (up is y). Named dimensions below are literal values, not an
expression language. A full gate file will enumerate its parts, including
both approaches and all four legs. This excerpt shows the relationships:

```yaml
scene:
  recipes:
    gate:
      composition: {file: gate-bridge.yaml}
  set:
    things:
      crossing:
        recipe: gate
        seed: 31
        east: 0
        north: 0
        height: {value: 0, unit: m}
```

An inline composition replaces `{file: ...}` with `{inline: ...}`. Exactly
one is required. The referenced file contains:

```yaml
version: 1
name: gate bridge
parts:
  - {id: leg-nw, type: leg, at: [-125, 0, -125], size: [50, 220, 50], ground: true}
  # Three more legs, with distinct IDs and actual contact to the body.
  - id: tower
    type: body
    at: [-125, 220, -125]
    size: [250, 530, 250]
    rests-on: [leg-nw, leg-ne, leg-sw, leg-se]
  - id: upper-deck
    type: deck
    at: [-1000, 75, -30]
    size: [2000, 3, 60]
    supported-by: [west-stays, east-stays]
  - id: west-stays
    type: stay-fan
    tower: tower
    deck: upper-deck
    tower-from: [-125, 420, -30]
    tower-to: [-125, 700, -30]
    deck-from: [-225, 78, -30]
    deck-to: [-1000, 78, -30]
    count: 8
    thickness: 1.2
roads:
  - id: upper-road
    on: upper-deck
    type: straight
    from: [-1000, 78, 0]
    to: [1000, 78, 0]
    width: 60
```

The excerpt deliberately references parts not shown. It is illustrative and
is not a valid standalone file. A complete file also gives the lower deck,
the under-deck and side storey bands, windows, columns, stays on both sides
of each arm, and road connections to the shores. A helix road specifies
centre, radius, start angle, turns, start and end elevations, width, and
orientation. A road is a continuous path, not merely a decorative deck.
IDs are unique across parts and roads. Reject unknown keys, types, duplicate
IDs, unresolved references, nonfinite or nonpositive dimensions, impossible
counts, and mutually exclusive keys. Errors name file, composition and part.
Bound file size, part count, fan count and path subdivision before allocation.
A file reference resolves relative to its containing set, with an explicit
error for an in-memory set without a usable base path. Do not allow recursive
composition includes in version 1. Update both set reader routes and the set
schema together, with matching refusal tests.

The gate's void is edited by changing leg heights, tower's base and height in
the YAML; all of those values must stay consistent. Version 1 does not imply
that editing a single value automatically shifts dependent parts. To meet
the one-edit demonstration without compiled constants, add a small named
parameter/reference facility: `parameters: {void: 220}`, with a typed
`{parameter: void}` scalar used for leg height and tower base; a body specified
by `top: 750` and `size: [250, 0, 250]` derives its height from its
base; zero is the explicit height placeholder. The same reference is changed
to 300 in the second draw. References resolve before geometry and checks;
unknown names, nonnumeric values and invalid resulting dimensions are
errors. Parameters are numeric literals, so references cannot contain cycles. No general arithmetic or executable templating is needed.

## Parts and existing work

- **Body and leg:** sized rectangular solids. `cities/solid.go` has `put`,
  placement and extrusion; the existing `Box` form and `Design` rendering
  carry geometry. A leg differs in its declared foundation, not a second
  mesh implementation. A body above four legs is one body plus four legs;
  the open space remains open. Do not turn `NewTower` into a gate special
  case: its tower stacks and culture are more specific than a plain body.
- **Storey band:** a sized repeating stack, storey height and count supplied
  in data. Reuse the existing city repeat/placement and box machinery;
  `house.go` and tower faces show storey and facade logic but bind choices
  to their own building plans. Extract a general piece only where its
  current contracts can be preserved. Bands under a deck need an explicit
  hanging connection, not an assertion that air holds them up.
- **Window ribbon:** face, sill, height, bay width, thickness, colour and
  lit share supplied in data. Reuse facade/glazing and seeded city drawing
  where their APIs permit; `house.go`, `facade.go`, `culture.go` and
  `towerface.go` already handle openings and lit glass, but are not yet a
  general public ribbon. Stable seed plus part ID and bay address must give
  the same lights independently of part ordering. No rooms behind glass.
- **Deck:** a sized solid using `Box`, city placement and bridge deck
  materials. `bridgedraw.go` already builds decks for a bridge's own plan;
  separate a general primitive only if it retains that contract.
- **Stays and hangers:** `bridgedraw.go`'s `chord` makes a member between
  points. A fan expands its declared endpoints and count to chords. Keep
  endpoint geometry and structural connection records together.
- **Road:** a surface/path on a deck or a helix with explicit radius and
  rise. Reuse city solids and existing road/street geometry where applicable;
  the existing bridge plan is not a general grade-checked path. The path
  and its geometry must come from the same resolved values.

The bridge sketch is a visual target, not source to import. Its tower is
250 m square, 750 m tall, on four 50 m legs above a 220 m void; the two
60 m decks are 16 m apart. Its under-deck block is 100 m wide and 36 m deep;
storeys are 3.5 m. The sketch uses a continuous deck through the void,
columns and paired fans. A complete authored gate should state all remaining
measures and materials rather than inherit hidden constants.

## Structural and road checks

These are geometric authoring checks, not a structural engineering or load
calculation. No claim of strength, stability, collision clearance or vehicle
safety follows from passing them.

Each solid has a contact surface or a declared support relation, checked
against the resolved geometry with one documented metre tolerance. Ground or
seabed roots must actually meet the composition's explicitly declared datum;
`ground: true` cannot make a floating part pass. For a resting connection,
check that the lower face meets a support's upper face and has positive
horizontal overlap. Resolve the directed support graph from rooted parts;
reject missing links, unsupported parts and cycles that have no rooted route.
For multiple legs require the declared contacts to exist; this check does not
prove the body is balanced. Window ribbons are attached to a named face and
validated against it, rather than asked to stand independently.

A stay or hanger has two endpoints, each geometrically on its named part.
Its direction of support is explicit: the deck depends on the stay and the
stay leads to the anchored tower; touching the deck at its other endpoint is
not a circular claim that the deck anchors itself. Each member of a fan is
checked, not only the fan's bounding box. A hanging band can depend on an
anchored deck; its attachment points must actually meet the deck. Validate
all attachments and root reachability before making geometry. A displaced
endpoint, floating foot and unrooted cycle each get a named refusal.

For each road, check its full ordered path and all joins from one declared
end to the other. Grade is absolute vertical rise divided by horizontal run,
not slope angle: straight segments use their endpoints; a constant-radius,
constant-rise helix uses `abs(rise)/(2*pi*radius*abs(turns))`. Zero horizontal
run, discontinuity, reversed or disconnected joins, and any grade over 0.04
are refused by road and segment name. A road must lie on its declared support
surface; a valid slope in empty space is not a drivable crossing. If a later
road form has varying grade, it needs an analytic maximum or a conservative
bound; sparse point sampling is not proof of the limit. Check ramps and
shore connections as part of the end-to-end route, not just the level spans.

## Proof and migration

Test the reader's two forms, path resolution, changed external bytes, bad
references and all limits. For every primitive test geometry, placement and
refusals, with anti-tests that actually perturb the condition under test.
Test rooted and floating structures, detached fan ends, cycles and valid
hanging assemblies. Test 4% exactly, just above it, zero run, disconnected
roads and helices in both directions. A fixture makes the gate from data;
compare its geometry and ferruccio wireframe with the sketch's wireframe 08, then change void
220 to 300 in YAML and redraw with the same binary. Record the binary and
input digests, both outputs and the named refusals of deliberately invalid
compositions. Draws use the brief's Scrooge, shelf and cache settings.

The existing tower, condominium and bridge could migrate later by mapping
culture-driven decisions into explicit reusable part inputs, preserving seed,
detail, materials and reference geometry. Their present interfaces and kinds
stay put in this work. Culture YAML should eventually be loadable from disk
with explicit provenance and cache invalidation, so an authored culture need
not require a rebuild; its embedded copies can remain a versioned default.
Moving it is outside this change.

## Manufacture taxonomy

> Historical design note: the inventory and instruction to keep existing kind
> names below predate the explicit decision to remove old kind aliases and
> update references. That decision supersedes the instruction. See the
> [constructor review](constructor-hierarchy-review.md) for the current
> implementation and remaining boundaries.

This design also calls for compact taxonomic structure and shared
constructors. The current `manufacture` is a single Go package with extensive private helpers
and 957 files below it; moving species to Go subpackages changes visibility
and risks import cycles, so moving files into directories is not sufficient.
The numbered Humboldt squid and Hatz files include distinct successive designs,
not merely fragments: catalogue their registered kinds, dependencies and
reference geometry before deciding which variants can share a constructor.
Keep existing kind names and geometry contracts while extracting general
mollusc/squid features into lower-level packages that do not import their
callers. Species constructors should pass their own parameters to shared
constructors; do not disguise duplicated species builders as inheritance.
Move one coherent family at a time, preserve source attribution, update
callers, and prove each existing kind still resolves and keeps its expected
geometry. Inventory all numbered families rather than assuming those two are
exhaustive. The package boundaries and which variants are truly equivalent
need examination before committing to a wholesale rearrangement.

## Historical gate fixture limits

This section describes the first fixture, before the lower-deck support and
wireframe proof were revised. For the current fixture, its clearance limits,
and the 220 m and 300 m exports, see the
[composition progress note](structure-compositions-progress.md).

The first data fixture has a 36 m under-deck envelope: ten 3.5 m floors
and a 1 m base. Each deck road joins grounded shore blocks via ramps.
The lower approaches rise 61 m over 1550 m; the upper rise 77 m over
2000 m. Both are below 4%. The shore blocks represent support, not a
terrain or water scene. Its lower deck has its own stays; the sketch's
lower-deck support and columns still need comparison with a drawn view. The first fixture is a construction study, not the requested wireframe
proof or evidence of structural strength.
