# Move the camera

A sequence plan can describe camera travel as well as scene time. It retains
the scene, samples the camera at each frame's elapsed time, and writes the
same numbered PNGs and resume manifest as a static-camera sequence.

```yaml
camera:
  unit: m
  look-at: {x: 96, y: 49, z: 96}
  moves:
    - from: {x: 184, y: 44, z: 110}
      to: {x: 184, y: 44, z: 82}
      speed: {value: 3.5, unit: m/s}
    - from: {x: 184, y: 44, z: 82}
      to: {x: 184, y: 44, z: 82}
      duration: {value: 2, unit: s}
```

That is an eight-second move at 3.5 metres per second followed by a two-second
hold. Set the plan's `duration` to ten seconds to include both. A longer film
holds the final camera; a shorter one ends during the path.

Points use Slayer's local scene coordinates: x east, y up, z south. On
terrain, the origin is the window's northwest corner at its lowest elevation.
These are not geographic coordinates or heights above the ground under the
camera.

The older set-file camera arrays use east, north, up; convert their north
coordinate to negative z when writing this block.

`look-at` is independent of travel. It stays fixed while the camera moves. The
path must not pass through it. Up defaults to the scene's y axis; an optional
`up: {x: ..., y: ..., z: ...}` supplies a nonzero alternative.

The existing look-at orientation chooses another up near a vertical view, so a
path through that region can roll; this is not a banking controller.

Every move gives exactly one of `speed` and `duration`. Duration can use
`ease: smooth`, a cubic ease with zero velocity at either end. Speed means
constant speed, so combining it with smooth easing is refused. The default
ease is linear.

Consecutive endpoints must meet; holds use equal endpoints with a positive
duration.

Camera time starts at zero relative to the sequence's start. A sequence
that starts its fire at 40 seconds still begins at the first camera point.
Reordering frames or resuming does not advance a mutable playhead. The plan
hash covers the camera, so a changed move cannot resume into an old film.

## Zoom while moving

An optional `camera.zoom` changes horizontal field of view independently
of camera position. The angles use degrees. Duration uses scene seconds,
as the movement durations do; empty easing means linear angle changes.

```yaml
  zoom:
    from: {value: 24, unit: deg}
    to: {value: 70, unit: deg}
    duration: {value: 10, unit: s}
    ease: smooth
```

Add this beside `moves` in the camera block. Smaller angles frame the
subject tightly; larger angles reveal more surroundings. This controls a
pinhole lens's field of view, not aperture, exposure or depth of field.

The first and last angles hold outside the zoom's duration. The lens and
position are sampled from time, so reordered or resumed frames keep the
same framing. Existing plans without `zoom` keep their scene's fixed lens.

## Hyperlapse: an hour in thirty seconds

The ordinary `duration` is playback time. Optional `scene-duration` says
how much scene time the clip covers. Without it, scene and playback time
advance together, preserving existing plans.

```yaml
fps: 6
duration: {value: 30, unit: s}
scene-duration: {value: 3600, unit: s}
```

This writes 180 frames at 120x time compression. Each frame advances the
scene by twenty seconds. Camera speeds remain metres per scene second;
camera movement and zoom durations must cover the intended hour too.

Scene `start` is added after this mapping. At start 45 seconds, the first
three frames sample 45,65 and85 seconds. The final frame samples 3625
seconds; the hour's end is the end of the last displayed frame interval.

Time-dependent fire, mist and motion receive the accelerated scene clock.
Explicit Sun and Moon directions stay authored and fixed; this option
does not add ephemeris-driven sky motion or invent intermediate frames.

[camera-hyperlapse.yaml](camera-hyperlapse.yaml) is a complete smoke example
with an hour-long camera move and a tight-to-wide zoom. It needs no survey
data to render. With the binary built, run:

```sh
game run hillside -- sequence --plan doc/camera-hyperlapse.yaml
```

The manifest records the actual scene timestamp for every frame. Resume
checks those timestamps and the plan hash, so altered time compression or
zoom cannot silently reuse an old sequence.

The [wooded-summit result](camera-hyperlapse-result.md) records an actual
thirty-second orbit and zoom, with all 180 frames and timestamps checked.

## A working wooded-slope plan

For a first look without survey data, use the self-contained smoke plan:

```sh
game run demo camera
```

It draws smoke 1 with a three-second move at 2 m/s and a one-second hold: 96
actual 640x360 frames at 24 fps. The poster goes to `out/demos/camera.png`;
frames and the resume manifest go to `out/demos/camera/`. This example uses
the CPU and an authored metre scale.

The original smoke scenes can use different coordinate scales; choose camera
points and speeds accordingly.

[camera-smoke-sequence.yaml](camera-smoke-sequence.yaml) is the complete plan.
It replaces `scene` with `smoke: 1` and gives the `film` size and sampling.
The same sequence writer and resume checks serve both routes. Smoke input
files, when any are declared, must be available and are hashed for resuming.

The plan selects the original scene factory; it does not recreate that scene
with special demonstration geometry.

The selector needs Game's argument-forwarding fix. `game run demo-one` is a
fixed smoke still for older Game builds. The [demo guide](demos.md) explains
the dependency and the menu.

[fire-camera-sequence.yaml](fire-camera-sequence.yaml) moves sideways past the
existing woodland fire and holds the ending view. It needs the same external
terrain and lunar data as the [woodland scene](fire-woodland-result.md). Build
first with `game build`, then:

```sh
game run fire-camera-check
game run hillside -- sequence --plan doc/fire-camera-sequence.yaml
```

Application argument forwarding requires the Game fix described in
[developing.md](developing.md). The fixed dry-run target works without it.
Frames go to `out/fire-camera/`. `--output DIR` can choose another directory;
relative CLI paths use the current working directory.

`--resume` verifies completed frames before reusing them. The interval is half
open: ten seconds at 24 fps samples 0 through 9 + 23/24 seconds. The explicit
hold keeps the endpoint visible.

To encode the actual frames, from the checkout root:

```sh
ffmpeg -framerate 24 -i out/fire-camera/frame-%06d.png \
  -c:v libx264 -crf 18 -pix_fmt yuv420p out/fire-camera.mp4
```

Encoding needs FFmpeg. PNGs and the resume manifest do not. Camera motion does
not yet add motion blur, collision avoidance, a moving look target, curved
splines, or geographic flight planning.

The six-second forest preview attempted during development stopped at its 1.25
GiB RSS watchdog before producing a frame. Both native and software GPU
traversal reached that limit. It is not a completed camera demonstration or a
measured optimization. Scene-preparation memory remains a performance task.
