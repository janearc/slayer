# Reusing Melancholia's weather in generated systems

2026-11-27. Design, not an implementation or a claim of matching renders.

## Experimental checkpoint

The candidate branch now accepts a `flowing` block on a generated body,
using the existing `planet` columns, zones and weather reader. It derives
the radius and gravity and uses a fixed body-space prime meridian when
making the spectral face. The first pilot at exposure 0.003 and a matched
pilot at exposure 1 both produced a star field with no discernible planet.
One center-facing face sample at exposure 1 returned RGB around 4.2–4.5e-6;
a matched control at exposure 50,000 showed a visible blue-grey disc, with
coarse angular bands and no discernible fine storm detail at 320 pixels and
64 weather cells. The visual researcher graded it a staging study, not
presentation quality. Two-camera coherence remains unverified; a finer
256-cell pair is prepared.

The 256-cell pair rendered continuous broad belts from two cameras, but
the researcher could not identify a unique shared landmark and did not
accept it as a coherence demonstration. Its bands were judged flat and
abrupt. A separate `marked-256` study holds its weather and cameras fixed
and adds Melancholia's chosen edge blend, cap/clearing and churn columns;
those values are made up. In the flowing path the fluid supplies the flow
instead of the painted spots and folds: including the latter in the `storms`
block does not put an authored spot into this weather. In jobs 315 and 316
the marked optics produced much richer interwoven brown/cyan bands than the
unmarked pair, judged a substantial visual improvement by the researcher.
The two views share a qualitative style, but neither observer could follow
a unique small feature between them; they do not establish world-fixed
weather. They remain studies, not accepted presentation frames.

For a concrete next coherence observation, select a recognizable landmark
from the *existing* cached flow, without modifying the weather: sample its
vorticity on the recorded grid and choose a strong, isolated feature in the
overlap of two sunlit views. Record the body-frame latitude and longitude,
seed, actual checkpoint time and sampled spin. Place both cameras so that
the selected location is resolved and visible, with overlapping coverage;
choose the feature and checkpoint before the comparison, and track a second
feature if feasible. Calculate projections independently from the known
body and camera coordinates. Render each unannotated frame at full size. A
separate labeled copy may mark the projected locations, leaving originals
untouched. Do not select only features that happen to line up afterward. Inspect the
neighborhood's shapes in both views with foreshortening and illumination
accounted for. This would be visual evidence for that selected feature,
not proof for all features, times or cameras. If the flow has no suitably
distinct feature at that time, record that result and select a different
authored seed or moment explicitly; do not introduce an eye-relative spot
and call it fluid weather. This observation has not yet been made.

Current restrictions are deliberately narrower than the proposed design:
metre world units, a fixed generated giant at an explicit middle, a single
body and one solar-luminosity directional scene sun. Binary stars, radius
overrides, alternative appearance blocks, legacy planets, additional lights,
films, shutters, camera paths, zooms and lens rigs are refused. The prelit
face does not receive shadows from other scene objects and says so in the
draw record. World-only staging refuses this appearance; session staging
handles it after the camera is known. The weather and authored optical
columns are recorded as made up. A later multi-body or animated version must
address occlusion, moving camera and re-lighting explicitly.

## Why

The generated giant's current `texture.Banded` is a colour texture on a
`medium.Giant`. The CPU can show its authored swirl, but the matched GPU
studies lose much of the fine structure in a sampled volume paint grid.
Melancholia already has another way to make a
turbulent giant. Reuse that work rather than grow another unrelated storm
model. This is an optional appearance for a general generated body; the
existing `giant_look` path and its defaults remain available.

## What exists

`sets/melancholia.yaml` describes a made-up giant with spectral atmospheric
columns, latitude zones, haze, and a `weather` block. Its notes explicitly
say the weather replaces painted vortices. `sets/weather-giant.yaml` is a
standalone view. The values (including cells 1024, five-hour day, 1e7 s
drag, 1e6 s relaxation and 1600 km storms) are chosen for that scene, not
default parameters or a universal atmosphere. The scene notes also name
missing multiple scattering and Earth air in front of the planet.

The existing reader holds a single `scene.planet` in `set.Planet`: columns,
zones, optional `storms`, `weather`, heat, cooking and other appearance.
`session/space.go:body` constructs `planet.Body` and its spectral columns;
`session/weather.go:weathered` constructs `weather.Weather`, reaches the
specified moment via its persistent cache, and passes it to `Body.Flowing`.
`planet.Face.airAt` reads `planet.Flow` in place of the painted folds and
spots; its origin latitude and vorticity select and mix columns and the
cap/clearing marks. The face evaluates received starlight and returns
prelit colour as a sphere's `Shine`. On GPU the legacy path may lazily chart
that face in latitude/longitude at an authored samples-per-pixel density;
CPU evaluates it per ray. See `session/space.go:charted`, `planet/chart.go`
and `session/weather_film.go`: films advance cached weather and reconstruct
the face as time changes.

Generated systems instead use `world.bodies`, `generate.Make`, and
`stage/build.go:bodies` to place `scene/bodies.Body` instances, including
their orbital motion, scaling and multiple lights. The old face is not a
part already carried by that type. Its `NewFace` constructs its longitude
frame from the eye and pole; `planet.Flow` describes longitude zero toward
the eye. A naive body-to-face copy rotates the same weather when the camera
moves. The old face is prelit from one star, and the sphere's `Shine` does
not automatically receive the general world's other lights or their
shadows. Its current selection and reconstruction is tied to the legacy
single `scene.planet`, not a named list of generated system bodies.

## Bounded first bridge

1. Add an opt-in `flowing` appearance to a generated giant's body. Reuse a
   shared typed description and the existing readers for atmospheric
   columns, zones, weather and cap/clearing material, with the same checked
   units; do not copy the YAML parser. Require explicit `made_up: true` for
   invented scene choices and print the selected parameters. Omission keeps
   the current giant appearance. Refuse the flowing option on a rocky body
   and refuse incompatible `giant_look` rather than silently mixing them.
2. Make the generated radius and mass/gravity the body inputs, instead of
   a separate hand-entered planet radius. Weather remains reusable through
   `weather.NewGrid`, `NewStirring`, `New`, `Reach` and `planet.Flowing`.
   The set explicitly supplies seed, grid, day, drag, relaxation, storm
   rate/width/strength, moment and checkpoint spacing. Columns, haze,
   deck albedo, zones and mark columns are independent authored controls;
   do not infer fictional compositions or silently hardcode Melancholia.
3. Establish a fixed prime meridian in the body's own frame, with its pole,
   rotation and orbit. Decouple this from the camera-facing axes currently
   created by `planet.NewFace`. Transform both view and star into that
   stable body frame; the weather's longitude and seed remain identical
   for cameras on opposite sides. Define planet radius and camera distance
   in the same units expected by `planet.Face`; convert once from the world
   unit and preserve camera-relative placement. No image-derived
   coordinates should be fed back into the weather state.
4. For the first visible slice, accept exactly one named illuminating star
   or directional sun, and a still moment. Stage the one flowing appearance
   at each body's actual world transform alongside ordinary bodies, with
   no duplicate old giant volume. Derive the chosen star's direction and
   spectral illumination at that body. Explicitly refuse unsupported
   multi-star illumination, mutual shadows, or moving flow in a film rather
   than claim they work. Do not add a second copy of the star's light to
   the prelit face. Subsequent slices can add multi-light composition and
   the dynamic film lifecycle; both need real designs, not hidden fallback.

The first slice should support multiple generated bodies in one set, but
only those asking for the flowing appearance need this face. Each named
body has independent weather state and a stable cache identity containing
all weather inputs, model/version and radius. Check the existing weather
cache key before relying on it; do not assume it includes those fields.
Keep cached weather and render assets, and let the normal Azazel queue
govern resources. A 1024-cell weather run and spectral chart may be costly;
measure actual time, memory, cache reuse and image detail with small preview
first. A chart is an approximation and a view/light-dependent product;
never reuse a stale chart across changed eye, light, time or body frame.

## Visual acceptance and honest limits

Build two distinct giants through the same constructor, with individually
authored columns and weather seeds, in a system with another visible body.
Show each from a disc and close camera, then a second longitude. Compare a
fixed body feature across views and, for a locked camera, compare CPU face
and GPU chart at the actual chosen chart density. Inspect every full-sized
frame for coherent bands, varied vortices, high-frequency structure and
seams; a CPU/GPU difference is an observed difference until measured, not
a parity claim. Document exact seed, time, inputs, light and visual artistic
choices. Compare with the existing Melancholia render and its stated limits,
not merely a passing command. The expected result is a compelling authored
visual system, not a claim of Jupiter-accurate dynamics, volumetric cloud
relief or physically plausible worlds.
