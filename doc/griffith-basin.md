# Griffith basin set landmark

`scene.world.griffith-basin` places the manufactured ridge and city
used by the Heard Griffith stage. it draws a ridge mesh, streets,
towers and emissive windows and lamps. the placement uses an absolute
map position in metres and a clockwise-from-north heading in degrees.

```yaml
world:
  griffith-basin:
    at: {east: {value: -1222, unit: m}, north: {value: 481, unit: m}, up: {value: 35.625, unit: m}}
    heading: {value: 30, unit: deg}
    scale: 0.5
    relief-scale: 0.25
    city-forward: {value: -800, unit: m}
    ridge-forward: {value: 3000, unit: m}
    city-detail: 0.25
    terrain-samples: 257
    city-light-scale: 6
    city-light-footprint: 4
```

`scale` changes the whole model. `relief-scale` adjusts ridge heights
independently. `city-forward` and `ridge-forward` shift the city and ridge
along the model’s local positive Z axis, which faces the heading.
negative `city-forward` moves the buildings toward the sea.

use `city-detail` to retain a fraction of repeated lots and
`terrain-samples` to choose a square ridge grid of 129, 257 or 513 samples
per side. `city-light-scale` sets emitted light. `city-light-footprint`
changes how much of a distant pixel the window and lamp geometry occupies.

this placement is authored scenery. the coastal skirt slopes below
mean sea to meet a finite ocean. the city blocks and ridge are generated
geometry, with no survey registration. the set placement omits the
stage’s haze volume so the shot can use a clear ocean boundary.
see [the builder](../scene/city/griffith.go) for bounds and defaults.
