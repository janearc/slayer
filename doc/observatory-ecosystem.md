# An observatory above the forest

[The YAML scene](observatory-ecosystem.yaml) is the integration scene: a
manufactured observatory on volcanic terrain, a seeded forest, and mist shared
by the grove. After the [Game setup](developing.md), run the fixed preset:

```sh
game build
game run observatory
```

The PNG is written to `out/observatory.png`. The USGS DEM must be unpacked on
the source shelf. The scene uses the GPU; it
does not silently fall back to the CPU when Metal is unavailable.

The terrain fields come from the existing Kilauea forest set. The site and
introduced coast redwoods are fictional. Recipe dimensions, weather, and camera
settings are authored scene choices, not measured conditions at Kilauea or
specifications copied from a real observatory.

## Recipes and specimens

`recipes` describes what can be made. `set.things` places individual specimens
and `set.stands` places groups. Each has an explicit seed. Changing a specimen's
placement must not change its construction.

An observatory is an assembly: enclosure, opening, mount, and telescope. Its
dimensions and pointing are explicit recipe inputs. Arrays can later place many
instances of an antenna recipe using the same distinction between construction
and arrangement.

The bundled [JSON Schema](../set/ecosystem.schema.json) defines the opt-in
ecosystem v1 subset of Set: recipes, pinhole films, placement and weather. Its
`$schema` ID selects that subset; validation never fetches the URL. Puzzlemouth
checks structure; Go checks references and construction.

This schema intentionally excludes legacy `scene.hud`, `scene.planet` and
`scene.world`; `set.assets`, `set.bounded_box` and `set.texts`; film `print`,
`frame`, `lens`, `shutter` and `glimpses`; camera `eye`; and stand `within`. It
is not a schema for the full legacy Set language.

Files using those legacy fields keep the reader without `$schema`.
`TestEcosystemSubsetAndReaderKeySetsStayInStep` holds the seven core object key
sets to their declared subset; shipped scenes and plans are also read in tests.
These checks do not turn domain relationships into schema rules.

Legacy sets without `$schema` retain their existing reader and diagnostics;
their YAML front door also goes through Puzzlemouth. No new CLI parser or flags
were added for these recipes.

The bundled schema and contextual source excerpts require APIs absent from
Puzzlemouth v0.1.3. The current `go.mod` pins released v0.1.4, which contains
them and requires Go 1.27.3.

Follow [review dependencies](review-dependencies.md) for a
standalone checkout; a parent Go workspace is optional, not a substitute for
release pins.

The original companion work was on `feat/source-excerpts` at
`c5f3f53`, following `feat/yaml-json-schema`.

## Complexity

The scene gives a default, with overrides on specimens and stands. It uses the
four names already recorded in `families.md`:

- `hull`: the specimen's coarse silhouette.
- `clumps`: coarse foliage surfaces or assembly detail.
- `quick`: a working preview of the specimen.
- `demo`: individual foliage or close-view assembly detail.

The four conifers support all four levels. Coast redwood and giant sequoia now
carry recursively branched shoots with individual needles or scales. Their
surface detail changes across levels; their physical leaf lengths do not.

Shared mesh groups reuse the shoots on both CPU and GPU. [Conifer
foliage](conifer-foliage.md) records the rules, botanical sources, and model
choices. The two original spruces retain their legacy previews.

These remain authored crown approximations. They do not model the full range of
mature, damaged, or reiterated crowns.

The optical observatory has a hollow drum and dome with an open slit, an
alt-azimuth mount, open truss, and monolithic or separated hexagonal mirror
panels. It is a visual instrument recipe, not an optical bench: there is no
guarantee of an unobstructed beam at every pointing.

Seeded construction and placement stay fixed across complexity levels. `Bounds`
retains a common physical extent for clearance and atmosphere.

The two loose shoots and the [spacecraft templates](space-observatories.md)
also support all four levels. Other kinds currently accept `quick` only;
unsupported levels are refused.

## Atmosphere

The forest refers to one weather description. Its shared field occupies the
union of specimen canopy footprints, from the terrain up to the smaller of mist
depth and canopy height. Overlapping canopies do not multiply its extinction.

Clearance is measured beyond the named object's enclosing circle; it keeps tree
bases outside that clearing.

The sample uses explicitly prescribed mist. Its extinction, depth, and coverage
are authored appearance controls. Coverage is a noise threshold, not a
calibrated cloud fraction: zero is clear and one fills the footprint. Wind moves
the patches; optional boundary fade softens each canopy edge.

The scattering colour, 0.95 grey, is also authored.

These are not values derived from the [redwood
references](redwood-atmosphere.md), and the scene does not claim a moisture
balance or condensation simulation.

A physical grove model would require sourced species and environmental
parameters. It should coexist with an explicitly authored atmosphere; the source
notes distinguish evidence from those modelling decisions.

## What reaches the GPU

Recipes make ordinary meshes or mesh groups. The GPU pools shoot meshes, reuses
their local instance hierarchies, and instances complete specimens through the
forest. A stand repeats four seeded specimens; it does not store separately
expanded needle geometry for all 5,000 trees.

The encoder samples the shared mist on the CPU into a bounded density grid once
per picture; the GPU reads that grid. It requests four samples through the mist
depth, subject to the existing 192-cell longest-axis cap.

Thin canopy edges in a kilometre scale field can be blurred or missed. The CPU
evaluates the field directly; agreement is approximate, not bit equality.

`gpu/ecosystem_test.go` renders each new kind at every complexity against the
CPU and checks that the grove actually attenuates light on the GPU. The
integration render has one observatory, 5,000 trees, and one mist volume. PNGs
and diagnostic logs stay outside the repositories.

Deciduous trees, Joshua trees, saguaros, brush, radomes and antenna arrays
remain further work. [General fire recipes](fire-recipes.md) now stand
independently of trees.

This scene is the first complete slice, not a claim that the whole ecosystem
catalogue is built.
