# Takes: moving a camera through time

A take carries a camera along a path while the shutter opens for successive
frames. It lives in `take/` because it coordinates Slayer's scene and
tracer.

## Timing and motion blur

A take combines a path, pace, and duration. The path implements
`thing.Motion` and returns the camera's subspace at a time. Tracks, orbits,
and caller-defined motions all fit that contract.

Frame n opens at `n / rate`. The shutter angle sets the fraction of that
frame interval spent open. Glimpses sample the open interval at evenly
spaced instants while the camera keeps moving.

Slayer renders each glimpse. A ninjafilm roll integrates their light and
develops the frame, producing motion blur. Frame times derive directly
from the rate rather than accumulating per-frame delays, avoiding drift.

## Go API

- `take.Cranked(rate, shutter, glimpses)` constructs a pace. use a
  positive finite rate, an opening above zero and at most one full turn
  in radians, and at least one glimpse.
- `take.Take(path, pace, length)` constructs a take with a non-nil
  path and positive duration.
- `shot.Frames()` counts complete frame intervals.
- `shot.Glimpses(i)` returns the samples for frame `i`, counted from zero.
  each sample exposes `At()` for scene time and `Eye()` for the camera
  transform.

a glimpse sits at the midpoint of its share of the open interval.
use its time to evaluate the world and its transform to place the
camera. the [ninjafilm roll](../ninjafilm/README.md) integrates and
develops the resulting frames.

## job films

a job uses its [film clock](../job/README.md#time) to count frames
and map playback time to scene time. its
[shutter](../job/README.md#shutter) averages linear light across
glimpses. each glimpse costs another trace. start with one for a fixed
scene and increase the count to resolve motion in the shutter interval.

on supported Metal paths, each ray can also sample a moving camera
within a glimpse. moving scenery is evaluated only at the glimpse
instants.
