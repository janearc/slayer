# City buildings from the manufactory

`city-grid-10000` constructs exactly 10,000 individually seeded buildings:
25 by 25 blocks with 4 by 4 lots in every block. The independent example is
[city-grid-10000.yaml](city-grid-10000.yaml); its
[still job](examples/jobs/city-grid-10000.yaml) uses the existing Slayer front door:

```sh
game build
game run slayer -- draw doc/examples/jobs/city-grid-10000.yaml
game run slayer -- draw doc/examples/jobs/city-grid-10000.yaml --camera 'traffic circle'
```

The second command uses the overview camera; the third examines an actual
four-approach traffic circle. Output is `out/city-grid-10000.png`; choose
`--out` to keep separate views. Cameras and world coordinates in the set
are east, north, up. The Go model uses x east, y up, z south. The footprint
is approximately 3.2 kilometres square, with 128 metre street spacing.

All dimensions, proportions, district weights, traffic organization and
material colours are authored scenery choices. They are not a survey,
engineering standard, traffic simulation or claim of habitable interiors.
Named dimensions, family ranges, massing proportions and preparation
budgets are recorded with units and provenance in
[city-choices.yaml](../manufacture/city-choices.yaml).

`city-rowhouse`, `city-apartment` and `city-tower` are standalone factory
kinds. Houses have gabled roofs and two or three storeys; apartments have
flat roofs and four to eight; towers have podiums, inset shafts, crowns and
twelve to twenty-three. Seeds select dimensions, material and storey count.
A repeated five-block district pattern concentrates towers and apartments
while keeping most interior lots residential.

`quick` has at most three broad glazing bands on each of two elevations.
These bands suggest facade rhythm at city distance; they do not enumerate
every floor's windows. `hull` keeps the same envelope and architectural
massing without glazing. Other detail levels are refused. No attached
lamps are generated. The example uses one authored distant sun.

A custom grid uses the same recipe and construction path:

```go
grid := manufacture.DefaultCityGrid()
grid.BlocksX, grid.BlocksZ = 10, 8 // 1,280 actual buildings
recipe, err := manufacture.City(grid)
if err != nil { return err }
made, err := manufacture.Make(recipe, manufacture.Sow(31))
if err != nil { return err }
parts, err := made.Things(maths.Local())
if err != nil { return err }
for _, part := range parts { world.Add(part) }
```

The function-body fragment assumes an existing `world`, and imports
`manufacture` and `maths` from Slayer. Dimensions in `CityGrid` use
Celestial `measure.Length`. Constructors refuse invalid dimensions,
impossible lots and populations above 100,000. Two or four lots across a
block are supported. Set `RoundaboutEvery` to zero to omit traffic circles.

`PlanCity(grid, manufacture.Sow(31))` returns placements without building
meshes. `Buildings()` returns each lot, envelope, kind name, seed and
placement. Reconstruct one with `KindNamed(b.Kind)`, `Make(row, Sow(b.Seed))`
and `Things(b.Place)`. `Roundabouts()` returns junction centres and outside
radii. Accessors return owned slices; save the slice once when looping.

Seeds depend on block and lot addresses, so extending the far grid edges
preserves existing building identities. The centred city's origin shifts
when its extent changes; translate the whole specimen to keep an edge fixed.

Major streets form a connected grid. Cross alleys give each lot frontage;
buildings reserve clearance from streets, alleys and circles. Circle
approaches terminate at four chords around an asphalt ring and raised
planted island. No straight road crosses an island. A conservative diagonal
corner reserve applies to all blocks, so adding a junction does not require
moving existing lots. Buildings occupy disjoint lot envelopes.

Disjoint surfaces are baked into meshes by material, bounded at 60,000
faces per mesh, and exposed through ordinary `MeshGroup` things. This avoids
an acceleration structure for every house or window.

`CityGrid.BatchPlacement` independently selects the city batching transform
path in the Go API. Its zero value and `CityBatchFused` select the default
fused append, which avoids temporary placed surfaces. `CityBatchNested`
selects the historical nested surface-placement path.

Both preserve exact
output and mesh-failure behavior. This setting changes transient construction
allocations only; it does not share retained geometry.

Set and job YAML do not
expose it on direct `kind` placements.

A set can opt in with a named city
recipe, referenced by the existing `recipe` field:

```yaml
scene:
  recipes:
    city construction:
      kind: city-grid-10000
      batch-placement: nested
  set:
    things:
      city:
        recipe: city construction
        seed: 31
```

The same recipe form accepts `city-clustered-10000`,
`city-clustered-green-10000` and `city-clustered-night-10000`.

The mode is
independent of `complexity`, lighting and other geometry settings.

It is
refused on non-city recipes and in repeated stand or ecosystem construction.
Existing direct `kind` placements and job keys keep their defaults. A job
still points to its authored set; the set's content remains part of the job's
input provenance.
