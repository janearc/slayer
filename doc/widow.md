# Widow: reproducing the two IMAX finals

Widow is the early orbital scene: Culture orbitals around a magnetar in
a nebular wake. `cmd/orbital` rendered it while Slayer separated from its
original tracer. This note preserves the two final images and the exact
historical tools used to make them.

## Archived output

The finals were rendered on 2026-10-05 and live in `~/reels/imax`:

- `strike-imax-0007.pfm` and `.png`: a beam strike viewed from inside the
  second beam. `strike-imax-minus2-tungsten.png` is its grade at two stops
  under and a 3200 K white balance.
- `zappy-close-imax-0007.pfm` and `.png`: an ungraded close view over the
  lattice, with the camera at `0,338,725`.

Both are single frames at moment 7, sized 4096 × 2864. `~/reels/imax/how`
contains camera specifications, the preview grid, and the original binaries.

## Reproduction versions

- `cmd/orbital` at `be73b24`, “an eye placed anywhere, and the pinhole’s
  width”; archived binary `orbital-eye`.
- `cmd/develop` at `81ea65a`, “print raws, at any exposure and white
  balance”.

Both binaries were built just before their commits. Later tracer changes
alter output, so use the archived binaries or check out those revisions
rather than expecting the current checkout to match.

## Commands

Camera coordinates are in thousands of kilometres relative to the lantern.
The archived `name|eye|look` specification for the strike is:

```text
two-inside|73.6,-96.5,-88.2|902.6,-1183.4,-1082.0
```

Render the strike:

```sh
GOMAXPROCS=2 orbital-eye -far -gas 8 -glow 0.05 -neon 0.4 \
  -w 4096 -h 2864 -n 1 -fps 1 -first 7 -frames 1 \
  -eye 73.6,-96.5,-88.2 -look 902.6,-1183.4,-1082.0 -up 0,1,0 \
  -wide 90 -raw -out ~/reels/imax/strike-imax.png
```

Render the close lattice view:

```sh
GOMAXPROCS=2 orbital-eye -far -gas 8 -glow 0.05 -neon 0.4 \
  -w 4096 -h 2864 -n 1 -fps 1 -first 7 -frames 1 \
  -eye 0,338,725 -look 0,0,0 -up 1,0,0 \
  -wide 90 -raw -out ~/reels/imax/zappy-close-imax.png
```

Grade the strike from its raw frame:

```sh
develop -exposure -2 -white 3200 \
  -out ~/reels/imax/strike-imax-minus2-tungsten.png \
  ~/reels/imax/strike-imax-0007.pfm
```

## Notes from the renders

`-n 2` traces four rays per pixel and gave the useful quality/cost tradeoff
in previews. These finals used `-n 1`, one ray per pixel. At the time,
a four-sample frame at this size was estimated to take about 2¼ hours.

The stepped glow comes from hard panel shadows rather than sampling noise.
Exposure and white balance belong in development: keep the raw light so
the grade can change without another long render.
