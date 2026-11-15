# cascades

A numerical proof of radiance cascades in two dimensions, set beside a
brute-force reference, as one picture.

The method is from Osborne and Sannikov 2024, "Radiance Cascades: A Novel
High-Resolution Formal Solution for Multidimensional Non-LTE Radiative
Transfer", arXiv 2408.14425. The scene is that paper's figure 1: opaque
glowing discs, two black discs, and a square that absorbs red weakly and
green and blue a hundred times more.

## Layout

- `flatland/` is the library: the grid, the ray march, the reference (long
  characteristics), and the cascades with the optional bilinear fix of the
  paper's section 2.5.
- `figure/` is a `package main`: it builds the scene, runs the methods,
  prints a report and writes the PNG.

## Run

    GOMAXPROCS=2 nice go run ./improv/cascades/figure -side 256 -out figure.png

Panels, left to right: the reference, the basic cascades, and the cascades
with the bilinear fix. With `-bilinear=false` the third panel is the
absolute difference of the first two, times `-diffgain`. `-scene single`
puts one disc in empty space and prints its falloff next to the exact
answer. `-side` must be a power of two. All panels share one tone map:
exposure, then x/(1+x), then sRGB.

The report lists, per method, the rays traced, march steps and wall time,
and the error against the reference outside the opaque discs.

## What it is not

It is not the renderer. It is two-dimensional, it solves radiative
transfer in three independent bands and nothing else, and it has no
tests. It is a look at whether the method works before any decision
about it.
