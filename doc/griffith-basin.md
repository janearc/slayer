# Griffith basin set landmark

`world.griffith-basin` reuses the same deterministic ridge and Los Angeles basin constructor used by the Heard Griffith stage, through the shared `scene/city` package. It builds one grouped ridge mesh, batched street blocks and tower meshes, and lit window/street/traffic surfaces. The set path omits the legacy basin haze volume because current clear-boundary ocean rendering refuses scene volumes. The set declaration supplies an absolute map position in metres and a clockwise-from-north heading in degrees:

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

The shared builder uses a right-handed proper rotation. Its local positive Z city axis aims along the supplied heading; local X is west at zero heading, Y is up. `scale` uniformly compresses the constructor geometry, while `relief-scale` separately compresses terrain heights. `city-forward` moves the urban layout along local Z in metres (negative brings it toward the sea); `ridge-forward` shifts the land relief inland while leaving the urban building bases on the lowland plane. This shot-specific layout puts city blocks seaward of the Griffith ridge instead of hiding them behind the nearshore terrace. `city-detail` deterministically retains that fraction of repeated neighborhood lots, `terrain-samples` selects a 129/257/513-square terrain mesh, `city-light-scale` adjusts emitted light over the positive range 1e-9–20 (small values support night scenes with high print exposure), and `city-light-footprint` enlarges and spaces the window/lamp geometry for far-view pixel coverage. These are independent far-view shot controls; all defaults preserve the original Heard stage. The example values are an explicitly authored shot approximation: half-scale city geometry, quarter-scale relief, with lowland lifted to mean sea so it does not sit below the ocean. A finite coastal skirt extends from local Z −600m at mean sea to −1200m at 60m below it before geometry scale, overlapping the finite ocean edge under water. Inland terrain extends to local Z 7200m and X ±4500m. No infinite plane sits under the boat. These positions, boundaries, shoreline relationship, basin, and towers are not survey geometry, a DEM, or a faithful LA skyline. It is opt-in; ordinary sets do not receive a landscape. The current tower vocabulary is the existing 67 varied, stepped downtown towers, not the separate kilometer-scale arcology. Retain the source note with the render.
