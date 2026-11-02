# Terrain fixtures

`fixture-lzw.tif` and `fixture-big.tif` contain the raster produced by
`writeTiff` in `raster_test.go`: 100 × 70 float32 elevations, tiled in blocks
of 32. libtiff’s `tiffcp` converts the plain fixture so the reader is tested
against independently encoded files.

```sh
TERRAIN_FIXTURE=/path/plain.tif go test ./terrain -run WriteThePlainFixture
tiffcp -c lzw /path/plain.tif fixture-lzw.tif
tiffcp -8 -c lzw /path/plain.tif fixture-big.tif
```

The second file uses BigTIFF, as does the USGS Kīlauea elevation model.
