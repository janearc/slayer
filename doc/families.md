# Tree architectures, foliage, and quality

This design began on 2026-10-14 and was extended through 2026-10-17.
It separates a species' branching rule, foliage, and parameters so a new
species can reuse existing growth code. The first implemented stage covers
the two spruces at `quick` quality; later stages below are a roadmap.

The [observatory ecosystem](observatory-ecosystem.md) now provides named
complexity levels, and [redwood and sequoia foliage](conifer-foliage.md)
uses recursive shoots. The sequence below remains the original design;
face-density selection and the additional branching architectures are planned.

## From species functions to shared rules

The original `spruceGrowth` and `norwayGrowth` in `manufacture/grow.go`
encoded each species separately. Their shared trunk, whorls, and habit
belong in one architecture. Species-specific counts, dimensions, and
colours belong in table rows.

The other limitations were visual: four- to six-sided foliage sleeves
looked like fronds up close, and previews and finished stills used the
same level of detail.

## Species data

Each row names:

- **Architecture:** the branching algorithm, returning a `growth` from
  counts and habit parameters.
- **Foliage:** the geometry carried by terminal shoots at each quality.
- **Parameters:** dimensions, counts, habit, and appearance.

Botanical family is recorded for provenance, not used as the growth rule.
The planned six species span five families. The architectures simplify
Halle and Oldeman's botanical models.

- **whorled:** Branching pattern: Trunk with branch tiers; branches carry
  paired rows of branchlets; Initial species: Spruce, Norway spruce

- **spire:** Branching pattern: Tapering trunk, flared base, irregular upswept
  branches, clumped crown; Initial species: Sequoia

- **leader:** Branching pattern: Single leader with spiral branches and
  further subdivisions; Initial species: Poplar, alder

- **forking:** Branching pattern: Low forks into heavy limbs; crown wider than
  tall; Initial species: Oak

- **burl:** Branching pattern: Crooked stems from a basal burl, turning and
  branching at joints; Initial species: Manzanita


These rules reuse `limb`, `sleeve`, `whorls`, `ring`, `habit`,
`stemCarrying`, and `newLeaf`. Add a missing arrangement, such as singly
spiralled attachments, once beside the existing primitives and test it
independently.

- **needles:** Geometry: Thin three-sided spikes spiralling toward the shoot
  tip; Species: Spruces

- **scales:** Geometry: Overlapping scales close to the shoot; Species:
  Sequoia

- **broad:** Geometry: Stalked blades with species-specific outlines; Species:
  Poplar, alder, oak

- **small:** Geometry: Small leathery ovals held edge-up; Species: Manzanita


The provisional choices are giant sequoia and valley oak. Coast redwood
would instead require flat needles in two rows; the species choice must
remain explicit.

## Quality changes detail, not identity

A kind and seed must produce the same tree at every quality: limb
positions, lengths, and girths remain bitwise identical. Higher quality
refines tessellation and foliage rather than changing growth.

The requested density is faces per cubic metre of crown. Choose the
nearest supported complexity without exceeding that budget, refining
within a level where possible.

- **demo:** Representation: Individual needles, scales, or leaves; Use: Stills
  and close views

- **quick:** Representation: Sleeve around each shoot; Use: Previews and
  crowds

- **clumps:** Representation: Sleeve around each branch and its shoots; Use:
  Distant trees

- **hull:** Representation: Crown pyramid and trunk prism; Use: Farthest views
  and coarse checks


Every level preserves crown placement, height, and width. Intermediate
densities adjust sleeve or needle tessellation. `quick` must reproduce
the original two spruce meshes face for face before later levels are added.

The first-stage regression, `TestQuickIsTodaysTrees`, checks both spruces
at twelve seeds. Its baseline was the skins from Fermat commit `e0859a6`;
since commit `85c1c37` made physical needle shoots the named-factory
foliage, the pinned skins are the woody parts of those needled trees.

## Foliage selection and ferns

A caller may choose a species' available foliage representation. A spruce
can retain the original sleeves or use individual needles at levels with
shoots. The hull has no foliage attachments.

Ferns follow the same progression: individual leaflets at demo, a strip
per frond at quick, wedges at clumps, and one fan at hull quality.

## Hawaiian vegetation

The Kilauea and Mauna Kea scenes requested on 2026-10-16 need different
plants: 'ohi'a lehua, 'ama'u, and hapu'u on Kilauea; mamane, naio,
and 'ahinahina at higher elevations on Mauna Kea.

'Ohi'a needs sympodial branching, a stout twisted crown, and broad leathery
leaves. A tree fern combines a trunk with a fern crown. Initial source
research starts with Forest Service silvics and park vegetation maps;
those sources are not yet claimed as completed parameter verification.

Call the branching rule an `architecture`. `form` already names the shape
of an individual part in `grow.go`; this distinction was settled on
2026-10-16.

## API and cost

Existing constructors in `manufacture/offered.go` keep making quick trees.
Quality selection should follow the named-kind API, exporting only what
external callers require. The proposed hillside flags are `-quality`,
accepting a level or face density, and `-foliage`; quick remains the default.

`mostFaces` in `manufacture/good.go` holds quality caps. At the recorded
cost of roughly 1 KB per face, a demo tree can require hundreds of
megabytes. `hillside -variety` reuses a few specimens across a forest.
Reducing the engine's per-face storage is separate work.

## Implementation sequence

Each stage gets its own commit and tests, passes `game check`, and adds
no new `game lint` failures.

1. Introduce architecture, foliage, and quality types; move both spruces
   to whorled rows and preserve quick meshes. This stage is implemented.
2. Add hull, clumps, and face-density selection with stable dimensions
   and identical limbs.
3. Add individual needles and selectable foliage; compare close views
   against the sleeve representation.
4. Add hillside quality and foliage flags, and fern quality levels.
5. Add sympodial 'ohi'a with broad leathery leaves.
6. Add hapu'u, then mamane, naio, and 'ahinahina.
7. Add leader branching for poplar and alder.
8. Add forking oak.
9. Add spire sequoia.
10. Add burl manzanita.

## Validation and scope

Every species belongs in `solids` in `manufacture/kinds_test.go` and must
produce valid geometry for forty seeds. `rows_test.go` rejects invalid
rows. Closed skins must be closed and outward-facing, checked externally
by `burning-trousers/manufactory`.

Derive the largest allowed tree from parameter extremes and check its
quality cap; seed sampling alone once missed a 104,576-face Norway spruce.
Chosen dimensions and colours retain author and date, following
`spruceHabit` and `norwayShoots`.

The original work boundary is `manufacture/`, `cmd/hillside/`, and this
document. Tracer, GPU, and crowd changes belong to the engine work in
`trace/` and `gpu/`. Request engine support rather than duplicating it here.

Completion requires review by someone other than the implementation author
and hillside renders of every species and quality from a shared seed.
The 2026-10-17 scope names trees and homes as the manufactory's kinds.
