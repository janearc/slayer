# Move the camera

a film job or sequence plan can move the camera through the scene.
Slayer samples the camera by elapsed scene time and writes numbered
PNGs when the job asks for frames.

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

the camera moves for eight seconds at 3.5 metres per second and
holds for two seconds. use ten seconds of scene time to include both.
a longer clip holds the final camera. a shorter clip ends during
the path.

Points use Slayer's local scene coordinates: x east, y up, z south. On
terrain, the origin is the window's northwest corner at its lowest elevation.
These are not geographic coordinates or heights above the ground under the
camera.

The older set-file camera arrays use east, north, up; convert their north
coordinate to negative z when writing this block.

`look-at` gives the fixed target. a move can give its own target
path as `look: {from: POINT, to: POINT}`. a hold with a moving target
pans in place. adjacent camera positions and look targets must meet,
and the camera path cannot cross its target. `up` defaults to the
scene’s y axis. supply a nonzero vector to change it.

each move gives either `speed` or `duration`. `speed` stays constant.
a timed move can set `ease: smooth` to start and finish at zero
velocity. linear easing is the default. use equal endpoints and a
positive duration for a hold.

camera time starts at zero relative to the film’s start. a job
starting its scene at 40 seconds still begins at the first camera
point. near a vertical view, look-at orientation can roll.

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

the first and last angles hold outside the zoom duration. without
zoom the scene keeps its fixed lens.

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
three frames sample 45, 65 and 85 seconds. the final frame samples
3625 seconds.

time-dependent fire, mist and motion use the accelerated scene
clock. authored Sun and Moon directions stay fixed.

## Example plans

check the resolved timing, camera and output paths before rendering.

```sh
bin/slayer draw doc/camera-smoke-sequence.yaml --dry-run
bin/slayer draw doc/camera-hyperlapse.yaml --dry-run
bin/slayer draw doc/fire-camera-sequence.yaml --dry-run
```

- [smoke camera](camera-smoke-sequence.yaml) moves for three seconds
  and holds for one. it writes 96 frames at 640 by 360 under
  `out/camera-smoke/`.
- [smoke hyperlapse](camera-hyperlapse.yaml) fits an hour of camera
  movement and zoom into 180 frames under `out/camera-hyperlapse/`.
- [woodland fire](fire-camera-sequence.yaml) moves past a fire and
  holds its ending view. it uses the woodland set’s external data.
  use `game run fire-camera-check` to check admission and inputs.
  its frames go under `out/fire-camera/`.

on a shared machine, submit renders through
[Azazel](azazel.md#submit-work). use the same job, files and build to
resume a sequence. see [frame resume](../door/README.md#resume).

the sequence format samples at one instant per frame. a
[job shutter](../job/README.md#shutter) can integrate multiple
instants and, on supported Metal paths, sample a moving camera
within a glimpse.
