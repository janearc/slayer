# Cloud below, guide stars above

A grove and a cloud deck are different shapes. Grove mist follows canopy
footprints. Regional mist follows the ground across a terrain footprint,
including clearings. A level deck keeps its height as the ground rises
through it. All three are prescribed weather.

A scene can describe overlapping regional layers:

```yaml
atmosphere:
  layers:
    low cloud:
      weather: valley mist
      extent: terrain
      albedo: 0.95
      base: {value: 0, unit: m}
    upper air:
      weather: thin air
      extent: terrain
      albedo: 0.95
      base: {value: 0, unit: m}
```

Each `weather` names a `prescribed-mist` recipe in the scene. Its extinction,
depth, coverage, seed and the scene wind retain their existing meanings.
The [small four-beam study](guide-star-deck.yaml) includes both recipes and
can be rendered through `game run guide-star-deck` after building.

`base` is a level scene-space height in metres above the terrain minimum,
not an altitude above sea level. The weather depth places the deck top.
Omitting `base` makes that layer follow terrain instead. Terrain clips
both kinds: cloud does not fill solid ground.

Layers share the terrain footprint and add optical effects where they
overlap. Their declared order is retained. There is no inferred weather
exchange between them, and putting two dense layers together doubles their
extinction where their profiles coincide.

A thin upper layer gives outgoing lasers something to scatter from above
the cloud. A beam has no glowing skin in vacuum. Choose the upper extinction
and depth explicitly; tree count does not determine either value.

The older single `atmosphere: {weather, extent, albedo}` form still means one
terrain-following regional field. Use either that form or `layers`, rather
than mixing them in one block. Fields retain immutable terrain heights
without retaining the scene-construction caches.

The four outgoing beams take visual inspiration from [ESO 4LGSF][eso].
The example does not copy that instrument geometry or calibrate its power.
The individual launch widths, offsets, pointing and display strengths are
picked scenery values; the source does not define our Gaussian radius.

[eso]:
https://www.eso.org/public/teles-instr/paranal-observatory/vlt/vlt-instr/4lgsf/

CPU density checks hold the deck profile at equal heights over different
terrain elevations. Scene checks hold layer order, optics, vertical bounds
and retention. An actual Metal comparison checks a beam scattered through
the deck against the CPU reference.

A moving film is still needed to assess noise advection, beam stability
and foliage changes together.
