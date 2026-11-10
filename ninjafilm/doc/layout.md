# Package layout

ninjafilm separates film primitives into small packages. Clients primarily
use the top-level API and the lens, lights, maths, and sensor APIs; the
remaining packages implement stock loading and development.

- **`ninjafilm`:** Stock selection through `Stocks` and `Roll`

- **`roll`:** Loaded stock and frame development through `Print`

- **`shelf`:** Stock descriptions assembled from `data/*.csv`

- **`lay`:** Stacked records or a mosaic layout

- **`record`:** One photosensitive layer

- **`crystal`:** Grain counts and statistical variation

- **`curve`:** Exposure-to-density response

- **`span`:** Wavelength intervals and RGB-channel membership

- **`hash`:** Reproducible seeded randomness

- **`number`:** Small numeric helpers

- **`data`:** Embedded tables, chosen parameters, and CSV parsing

- **`colour`:** The darkroom: exposure, bloom, roll-off, sRGB, false colour,
  masks, and grain

- **`grain`:** Seeded gradient noise

- **`film`:** Film gauges, projections, camera rays, and lamps

- **`frame`:** Pixel grids containing linear light, coverage, and depth

- **`lens`:** Thin-lens projection and exposure integration

- **`sensor`:** Light-collecting primitives

- **`lights`:** Spectral power in watts

- **`maths`:** Typed power, temperature, area, and Planck's law

- **`mocks`:** Generated mocks for `Lay`, `Shelf`, `Projection`, `Emitter`,
  and `Beamer`


## Dependencies

The packages use Slayer's maths and the libtheme-css version selected by
Slayer's `go.mod`; they do not depend on Lamb.

Puzzlemouth reads the chosen parameters in `data/picked-numbers.yml`.
Each chosen value records its author so it remains distinguishable from
measured data.
