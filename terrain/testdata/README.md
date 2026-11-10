# Terrain fixtures

`fixture-lzw.tif` and `fixture-big.tif` contain the raster produced by
`writeTiff` in `raster_test.go`: 100 x 70 float32 elevations, tiled in blocks
of 32. libtiff's `tiffcp` converts the plain fixture so the reader is tested
against independently encoded files.

```sh
TERRAIN_FIXTURE=/path/plain.tif game run terrain-fixture
tiffcp -c lzw /path/plain.tif fixture-lzw.tif
tiffcp -8 -c lzw /path/plain.tif fixture-big.tif
```

The second file uses BigTIFF, as does the USGS Kilauea elevation model.

## Floating-point prediction fixtures

fixture-float-predictor-le.tif and fixture-float-predictor-be.tif are authored
16 by 16 float32 grids. Native value at (column,row) is
row*16+column-0.5. They were encoded independently with installed libtiff 4.7.2:

    raw2tiff -w 16 -l 16 -d float -c none halfsteps.raw halfsteps.tif
    tiffcp -t -w 16 -l 16 -c lzw:3 -f msb2lsb -L halfsteps.tif fixture-float-predictor-le.tif
    tiffcp -t -w 16 -l 16 -c lzw:3 -f msb2lsb -B halfsteps.tif fixture-float-predictor-be.tif

The raw input is little-endian IEEE float32, row-major. The fixtures differ
in TIFF directory byte order; both use MSB-first compressed bit order.
They contain no geographic data or survey measurements.
