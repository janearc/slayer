# Night water reflection sources

These publications are registered in the root `sources.yaml`. `sources
fetch KEY` brings a paper to the local citation shelf, and `sources path
KEY` shows where it is. The sources are research references; listing a
paper does not mean its model is implemented or its assumptions verified.
The paper bytes stay on the shelf, outside the repository.

| Key | Use and scope |
| --- | --- |
| `perez-vega-2024-bridge-reflections` | Nighttime artificial-light reflections near Berlin bridges, measured for polarization in a freshwater river; not a Futian brightness calibration. |
| `lynch-2011-glitter-and-glints` | Observations of sun and moon glitter, including small-wave glints and source-size effects. |
| `bruneton-2010-ocean-lighting` | Scale-dependent ocean geometry, normals and statistical reflectance, including reflected environment light; approximate local reflections. |
| `yan-2014-normal-map-glints` | Pixel-footprint normal distributions and specular glint sampling. |
| `cox-munk-1954-sea-roughness` | Daytime sun-glitter measurements used to infer sea-surface slope statistics. |
| `quan-fry-1995-seawater-index` | Refractive-index fit for seawater versus wavelength, salinity and temperature. |
| `podee-2021-water-reflection-antialiasing` | Spatial and temporal anti-aliasing of a spherical light source reflected by dynamic waves. |
| `veach-guibas-1995-combining-sampling` | Multiple importance sampling for combining complementary Monte Carlo sampling techniques. |

OceanTex already records Cox–Munk and Quan–Fry numeric claims in its own
`oceantex/sources.yaml`, including how those claims were originally read.
Adding the publications here does not revise those claims or their evidence.

For Futian, use the papers to investigate the projected surface footprint,
the treatment of unresolved slope variance and reflected scene light, and
sampling variance. Compare against matched raw renders and repeated samples;
do not infer a correction from a single attractive picture.
