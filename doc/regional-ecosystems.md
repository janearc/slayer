# A population belongs to its region

This implementation is a code draft, with builds, geometry checks and
renders deferred to the review pause. It adds authored mixed populations
and a local terrain adapter; it does not yet stream a whole world.

## Fixed roots

The import path is `github.com/janearc/slayer/scene/biome`. The older
top-level `biome` package describes local scenery choices; this package
owns fixed spatial candidates. They have separate responsibilities today.

`scene/biome` defines a region in fixed source east/south metres. Its
lattice is anchored at source coordinate zero, independent of the current
terrain window or camera. A query selects roots without rebuilding the
population from an index in that query's result.

An identity contains the region name and seed, lattice column and row,
and an ordinal within the cell. Versioned SHA-256 draws supply position,
mixture choice, geometry seed and orientation. The full tuple is the
identity; a compact seed hash alone is not collision-free identity.

Stability assumes a fixed region definition, including the mixture order,
cell size and density. Editing that definition can change what an ID names.
Generator v2 separates the orientation draw from the geometry seed; it
deliberately changes the draft v1 population before any appearance baseline.

Each cell's candidate count is randomized rounding of its mean density:
the integer part plus one with probability equal to the fractional part.
Positions are uniform draws inside that cell. This is an authored
distribution, not a Poisson process or a model of recruitment and growth.

Recipe weights choose the mixture. Terrain-height and clearance filters
may reject candidates, so the final count can be below the requested
candidate density. Rejected candidates are not replaced by new draws.

Query budgets bound examined cells and generated candidates, including
those rejected at boundaries. A budget failure returns an error and no
partial candidate list. A small cell halo protects query-boundary rounding.
Roots use half-open ownership so adjacent queries do not duplicate them.

SelectWithUsage reports examined cells and generated candidates separately
from selected roots. A scene can divide one total allowance across region
queries; query-edge rejections still cost work and are not free credits.

## YAML

Declare construction recipes separately, then mix them in a region:

```yaml
ecosystems:
  oak and brush:
    seed: 21
    cell: {value: 16, unit: m}
    density: {value: 60, unit: ha^-1}
    variety: 4
    complexity: clumps
    bounds:
      west: 263800
      east: 263896
      north: 2148768
      south: 2148672
    mixture:
      autumn oak: 1
      red brush: 3
    atmosphere:
      weather: garden mist
      extent: canopy-and-ground
      fade-at-boundary: true
```

This block belongs under `scene`. Every mixture key names a solid recipe
in `scene.recipes`; fire recipes are refused. Complexity inherits the
scene setting when omitted. Variety limits distinct geometry prototypes
per recipe in that region, without changing root placement.

Optional `above` and `below` select an open band of source elevations.
Optional `exclude` takes `around` and a metre-valued `clearance`. The
adapter includes the candidate's canopy radius in that clearance.

Atmosphere uses the existing shared prescribed mist over the population's
canopies and ground. Density does not become a transpiration or water rate.
The same prescribed-weather sources and limitations still apply.

When atmosphere is present, weather, extent and fade-at-boundary are all
required, including when the desired fade-at-boundary value is false.

## Current adapter and next boundary

Hillside queries the fixed region through its existing terrain window. It
samples ground on the actual mesh diagonals, places shared geometry and keeps
root orientation independent of enumeration order.

Selection preserves overlapping candidate IDs, horizontal positions, recipes,
seeds and prototype choices for a fixed definition.

Placement is crop-dependent today. Offset windows can sample and smooth
different ground under the same candidate. Habitat filtering, final height
and geometric exclusion can consequently differ. Candidate invariance does
not promise identical surviving placed trees across terrain crops.

This adapter still uses the current cropped, planar terrain stage. Source
projection/datum conversion, curved-world placement, visible-size detail,
aggregate distant canopy and persistent moving-frame residency are not
implemented by this population slice.

A future visible-region query must include a halo for maximum canopy
extent. Selecting only roots inside the image would clip branches whose
roots lie just outside it. Canopy bounds and temporal acceptance remain
requirements for the streamed-world adapter.

The default query work limits are picked in `selection-policy.yaml` and read
through Puzzlemouth. The library applies its supplied per-query limit;
Hillside now shares one default allowance across all ecosystem regions in that
prepared scene.

Exhaustion refuses preparation, rather than quietly dropping later regions or
granting each another full allowance. Rendering and source managers must also
bound their combined populations, geometry prototypes and live terrain
resources.

These are count/work limits, not a world-wide memory cap. A visible-world
manager must budget the active view's regions and distant representations
instead of trying to enumerate every root on a planet.
