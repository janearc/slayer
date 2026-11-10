# Running Grissom

Grissom looks at the sky, the earth, and the sky from a place on the
earth. Its window lives in `cmd/grissom` for now. It draws triangles with
Cocoa and Metal; this command does not run Slayer's ray tracer.

Celestial-data has a named set for everything the viewer reads. Start
there. The viewer still expects some older directory and file names;
the links below bridge those names to the set's layout.

## Checkouts and build

Use macOS with Metal, the Xcode command-line tools, cgo enabled, and the
Go toolchain named by Slayer's `go.mod`. The layout used here has three
sibling checkouts and a local data directory outside them:

```text
grissom/
  slayer/
  celestial/
  celestial-data/
  local/data/
```

The [review dependency setup](review-dependencies.md) builds Slayer with
`GOWORK=off`; a parent workspace is optional for companion development.
Carrie is not needed to build or run this viewer.

Celestial-data needs `de05f2c` or later for `fetch-set`,
and `3c042c3` or later for Chroma's current listing. Use its subsequent
NASA URL and SkyView wait fixes too; see its `DOWNLOADS.md`.

Configure Slayer's source shelf using the [Game workflow](developing.md).
From the Slayer checkout, build the viewer and the sibling fetcher:

```sh
game build
(cd ../celestial-data && game build)
```

Slayer's `bin/grissom` is the stamped viewer. Its separately configured
`SLAYER_SHELF` is not Grissom's survey directory.

## Fetch the set

Stay in the Slayer checkout. The fixed `grissom-data` run fetches into
the sibling `local/data` shown above. Allow roughly 2 GiB for the inputs,
then more room for
Go's build cache and the viewer's working memory.

```sh
export GRISSOM_DATA="$(cd .. && pwd)/local/data"
game run grissom-data
```

The run supplies `--into ../local/data` relative to `celestial-data/`.
For another destination, change that run's argument or use the fetcher's
own Game workflow. `GRISSOM_DATA` below is a shell convenience for the
links and viewer arguments; it does not change this fixed run.

The command fetches
the two GLIMPSE detail mosaics, the MeerKAT intensity mosaic, the PPMAP
column-density map, the Earth inputs, four whole-sky maps, and Chroma's
colourway CSS files. Twilight is the default colourway.

The set lays out its files as:

```text
data/
  sgra/glimpse/GLM_00000+0000_mosaic_I1.fits
  sgra/glimpse/GLM_00000+0000_mosaic_I4.fits
  sgra/meerkat/MeerKAT_Galactic_Centre_1284MHz-StokesI.fits
  sgra/ppmap/l000_cdens.fits
  earth/ne_50m_land.geojson
  earth/ASCIIdata_splib07a.zip
  earth/world.200407.3x5400x2700.jpg
  earth/BlackMarble_2016_01deg.jpg
  sky/skyview-Mellinger-Green-3600.fits
  sky/skyview-H-Alpha-Comp-3600.fits
  sky/skyview-IRIS-100-1800.fits
  sky/skyview-408MHz-3600.fits
  colourways/twilight.css
```

There are other colourways too. A new colourway in Chroma's GitHub
listing is found without a change here. Keep the USGS ZIP compressed;
the viewer reads its measured grass and seawater spectra from the ZIP.

The two infrared files are about 481 MiB each, the radio file 813 MiB,
and the dust map 8 MiB. A fetch reports each completed file, not progress
during that file. Large files can take a long time on a slow link.

One failure does not stop the other files. Run the same command again
after a failure: completed files are kept. A failed file starts again
from the beginning; the fetcher does not yet resume partial downloads.

## The viewer's older names

The current viewer needs these compatibility links after fetching:

```sh
ln -s glimpse "$GRISSOM_DATA/sgra/data-irac"
ln -s meerkat "$GRISSOM_DATA/sgra/data-meerkat"
ln -s ppmap "$GRISSOM_DATA/sgra/data-ppmap-l000_cdens"
ln -s skyview-Mellinger-Green-3600.fits \
  "$GRISSOM_DATA/sky/skyview-Mellinger-G-3600.fits"
```

This is first-time setup: do not repeat `ln -s` against existing links.
The detail files are required. The sky maps and Earth photographs are
optional to the viewer even though the named set fetches them all.

Without photographs, Earth uses Natural Earth's land polygons and USGS
grass and seawater reflectance under D65 daylight. Without sky maps,
the whole-sky background is plain. Missing optional files are skipped.
At least one readable colourway is required, including for a shot.

## First render

Start small. The defaults are depth 13 for the detail maps, 7 for Earth,
and 8 for the whole sky. A first check can use 7, 4, and 4:

```sh
game run grissom -- \
  -data "$GRISSOM_DATA/sgra" -earth "$GRISSOM_DATA/earth" \
  -sky "$GRISSOM_DATA/sky" -colourways "$GRISSOM_DATA/colourways" \
  -depth 7 -earth-depth 4 -wide-depth 4 \
  -shot "$PWD/out/check" -view -1
```

Create `out/` first. These application arguments require the forwarding
fix described in the [Game workflow](developing.md#build-and-run); older
Game builds need a configured run with those fixed arguments.

This writes `out/check-sky.png`, `out/check-earth.png`, and
`out/check-standing.png`. `-shot` is an output prefix, despite its help text
calling it a PNG.

Choose `-view 0`, `1`, or `2` for just sky, Earth, or standing. Omit
`-shot` and `-view` to open the window with the same data.

The small detail mesh is deliberately coarse. Raise `-depth` for a
useful close view. A detailed check has also run with depth 13, Earth
depth 6, and whole-sky depth 5. Each extra whole-sky depth multiplies
the number of cells by four.

All survey pixels are read and laid again at startup; lower depth does
not turn the input mosaics into small cutouts. Once built and supplied
with local data, the viewer needs no network.

Offscreen shots still need Metal. A restricted process may report
`no metal device` even after every dataset has loaded successfully.

## Finding your way around

Drag to turn the view and scroll to change its scale. `e` switches
between sky and Earth. On Earth, `p` places an observer at the crosshair;
`f` switches between Earth and that observer's standing view. Comma and
period move the clock back and forward by an hour.

`1` through `4` toggle the detail layers; `5` onward toggle the sky maps
that were actually loaded. `o` cycles outline strength; brackets change
outline width. `+` and `-` change lightness, `c` colourfulness, and `b`
the background. `m` cycles the photographs and the measured Earth.

`h` changes HUD size, `t` the colourway, and `q` quits. Choose a starting
colourway with `-colourway NAME`, using the CSS file name without `.css`.

`-look lon,lat` starts Earth and the observer at those coordinates.
The standing view uses the current time unless you move its clock.
Shots of that view are therefore not fixed-time reference images.

## Sources stay with the data

The infrared data comes from
[IRSA's GLIMPSE II mosaics](https://irsa.ipac.caltech.edu/data/SPITZER/GLIMPSE/images/II/1.2_mosaics_v3.5/).

The [MeerKAT release](https://doi.org/10.48479/fyst-hj47) specifies
CC BY-NC 4.0 and cites [Heywood et al.](https://arxiv.org/abs/2201.10541).

The [PPMAP tile](http://www.astro.cardiff.ac.uk/research/ViaLactea/PPMAP_Results/l000_results/)
is the Hi-GAL column-density product, not its temperature map.

[Natural Earth](https://www.naturalearthdata.com/about/terms-of-use/)
is public domain; its requested credit is "Made with Natural Earth."

The public-domain [USGS library](https://doi.org/10.5066/F7RR1WDJ)
is described by Kokaly and others (2017). The archive hash is recorded
in Celestial-data's `usgs/sources.yaml`.

Each SkyView survey has its own terms; SkyView serving it is not a
common licence. NASA images and pictures made from them stay local,
following the Earth-image adapter's policy. They are not repository assets.

The fetcher checks transfer lengths but does not yet write a provenance
manifest. Grissom's layer source records contain paths but leave their
other provenance fields blank. Keep archive query output and hashes of
the files actually used, including where any copied cache came from.

A hash identifies cached bytes; it does not authenticate their archive
origin. Record that distinction rather than calling a cache a new fetch.
