# Fire on a wooded slope

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

This scene places a surface fire among procedural coast redwoods on the
Kilauea DEM, with the NASA lunar face visible above the downhill canopy.
The side view makes the terrain's rise visible. Camera, sky and tree
geometry stay fixed while heat, fuel, flame shape and lighting evolve.

The grove is fictional. DEM smoothing, tree placement, flame dimensions,
optics and sky positions are authored choices. The sky uses Slayer's lunar
and atmospheric components with explicit positions, rather than computing
an ephemeris from the scene's clock.

## The rendered views

Local outputs are in Grissom's `local/renders/fire-woodland`:

- `woodland-poster.png`: an original 720p frame at 42 simulation seconds.
- `woodland-motion-720p.mp4`: 96 actual frames at 24 fps, four seconds
  of real-time motion from 40 through 43.958 simulation seconds.
- `woodland-spread-720p.mp4`: 120 actual frames, covering 0 through
  59.5 simulation seconds, played at 24 fps for a five-second 12x diagnostic.
- `woodland-progression.png`: actual frames at 0, 30 and 59.5 seconds.
- `motion-frames/` and `spread-frames/`: original PNGs and timed manifests.

Both use 1280x720 with four rays per pixel on the native GPU renderer.
The frozen source build is `6299710`. Captions sit at the bottom to leave
the moon visible. There are no interpolated frames. PNG hashes, exact
simulation times, unique frame counts and decoded movie counts are verified.

Recorded frame time averaged 4.514 seconds for motion and 4.315 for spread.
These timings apply to this controlled 720p scene with four rays per pixel.
All 216 source frames are distinct; the movies decode to 96 and 120 frames.

## What the fire does

The existing [heat/fuel model](fire-fuel-design.md) advances temperature
and remaining surface fuel. Flame absorption, height and source strength
follow that state. The forest floor uses `surface: terrain`, so the
controlled study's diagnostic fuel-colour tiles do not cover the DEM.

The solver remains a horizontal grass-fire proxy placed on terrain. It
moves across ground at different elevations, including uphill; slope does
not accelerate it. Trees receive the changing light but their crowns do
not ignite or lose fuel. Wind and transported smoke remain future work.

## Moon fixes

The visible Moon exposed three renderer problems. Its astronomical distance
enlarged local shadow offsets, its purely emissive pixels requested diffuse
lighting, and its sphere blocked its own paired distant light. These are
fixed without moving the Moon closer or changing its physical radius.

Both CPU renderers, the full GPU suite and the full hillside suite passed.
The appearance choice then passed focused config and staging checks. The
moonlight test includes an ordinary occluder as its control. Earlier manual
fault trials have no replay patches and are not acceptance evidence.

The earlier claim of no new hard lint findings was wrong: the reviewed
branch adds 45 hard findings across code and documentation.
Global history and smokecheck gates remain recorded in the earlier
[fuel-study result](fire-fuel-result.md); this is not a release-ready claim.

## Reproduce

After the source shelf is set up, build with Game. The checked plans are
[real-time motion](fire-woodland-motion.yaml) and
[spread](fire-woodland-spread.yaml), both referencing
[the same scene](fire-woodland.yaml).

```sh
game build
GOMAXPROCS=2 game run hillside -- sequence --plan doc/fire-woodland-motion.yaml
GOMAXPROCS=2 game run hillside -- sequence --plan doc/fire-woodland-spread.yaml
```

The spread plan writes two frames per simulation second; encoding at 24 fps
makes 12x playback. The motion plan writes 24 frames per simulation second
and plays at that rate. Local captions are added after rendering; source
PNGs remain separate.

The lunar image is NASA's SVS CGI Moon Kit colour mosaic, credited to
NASA/GSFC/ASU and LROC. The pinned source is `svs-moon-kit-color` in
`sources.yaml`; the image remains on the external source shelf.
