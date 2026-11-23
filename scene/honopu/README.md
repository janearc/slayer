# Honopū Beach

The wet frame and exposed terrain use the same `cove.New` constructor.
The surveyed 3,264 × 2,880 m window covers the Nā Pali sea cliffs and
Honopū Valley, with sand, rock and vegetation shaded procedurally.
A gridded surface can represent cliff faces, with no overhanging arch.

NOAA's [Hawaiʻi CUDEM][cudem] at 1/9 arc-second supplies the original
elevations. The render uses 4 m UTM samples and retains local mean sea
level. The window has 817 × 721 valid samples, from −49.821 to
1,127.729 m. It has no synthetic offshore infill.

[cudem]: https://coast.noaa.gov/dataviewer/#/lidar/search/where:ID=9428

## Made ground

`detail: made` lays one stretch of the survey finer than the survey is: a
wall facing north, about 105 m tall, seen from `view: wall`, 100 m out over
the sea. Every surveyed height stays where the survey put it.

The detail
between them is made, cut along the surface's normal by three rules that
read the survey: flutes down the fall line, deeper where more drains;
level beds standing out of the face; and a roughness read from how broken
the survey is over several samples, made only at wavelengths shorter than
the survey's and no steeper than the ground.

`made-flutes`, `made-beds` and
`made-roughness` lay one rule each. The draw prints to its standard error
how many corners are surveyed and how many are made, the edges it reached,
and how far the made corners are from the survey: the median, the 90th and
99th share of them, and the most.

The original is `ncei19_n22x25_w159x75_2021v1.tif`.
The source tile and the embedded Slayer window have these SHA-256 sums.

```text
1c73d38004fa26eeba80b0a9e1bdd317b097eb007cebe53bcf76f39149e0bcec
81923d108b6cdeff59c587fab6e92efc5d50c78c48f151e9920433e94d135800
```

Set `HONOPU_CUDEM` to the source GeoTIFF and `SLAYER_JOBS` to a checkout
of slayer-jobs. Build the cropper with
`cc -O2 "$SLAYER_JOBS/honopu/prepare.c" -o bin/prepare-honopu` followed by
`$(pkg-config --cflags --libs proj libtiff-4)`. Pipe its output through
`gzip -n` to produce `scene/honopu/honopu-window.bin.gz`.
