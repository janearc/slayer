# CPU wave height-field adapter

`NewHeightField` samples `oceantex.WaveCascades` at an explicit sea, time and
seed, then returns the existing `scene/shape.HeightField` type. The caller
supplies the square extent and sample count. The extent must fit within the
largest cascade period; smaller cascade fields repeat periodically across the
requested square. The count is from 2 to 1024 per side (at most 1,048,576
height samples). No time, seed, or resolution is inferred.

Oceantex coordinates are east, north, up. The adapter maps east to Slayer x,
height to Slayer y, and north to negative Slayer z. Every cascade is sampled
periodically and centered on the same scene origin. The requested square is
finite, including both edges; shape bounds and normals are those of the
resulting bilinear samples.

This is a vertical-height surface at one instant. It does not carry oceantex's
horizontal choppy displacement, unresolved footprint filtering, shore/wall
effects, shading, or optics. Changing time requires constructing a new field.
It is not the complete surfaced shot or the same surface computation as
`oceantex.View`.

Set-authored world halves may stage this generic shape through `stage.World`
or `stage.Add`; there is no ocean-specific renderer implementation.

The returned shape retains `HeightField`'s conservative distance marcher and
bilinear cell normals. It does not implement an exact ray hit or a ray span.
The existing bounded march can miss at grazing views, as recorded in
`DEBT.md`; scenes needing a guaranteed first hit there need a cell-walking
intersector first.

There is no ocean-specific Metal validation or medium optics. The authored sea
values are explicit inputs to the retained prototype; this adapter makes no
physical-source verification claim.

## CPU-only flat-mean continuation prototype

`NewFlatMeanContinuation(core, meanLevelMetres, collarWidthMetres)` is an
explicit opt-in wrapper for a finite height field. It preserves the core's
bilinear height values and normals exactly inside the core.

Outside, it extends the boundary bilinear cell to match height and slope, then
uses a smoothstep collar to damp waves to the caller's explicitly chosen mean
water level. The collar width is a separate physical input, limited to one
core width; the existing core sample count remains independently selected and
is never silently reduced.

The wrapper stores no second height grid. The CPU tracer's existing bounded
march supplies a finite per-ray step cap.

Use the wrapper as one `thing.Thing` with one clear material. Its core, collar
and mean region form one height function and one clear boundary; do not add an
overlapping clear plane. This prevents internal seams from applying Fresnel or
changing medium identity twice.

The outer square is still finite, however. A ray that crosses the water surface
outside its bounds gets no surface event, and a camera framing the outer edge
can still reveal the cutoff. Choose the finite extent from the scene's required
view range.

This is a controlled far-field approximation, not a physical continuation of
the sea spectrum: the waves are deliberately damped to flat water. It does not
add shoreline, shore reflection, water absorption, scattering, or a measured
site model. There is no set YAML or stage wiring for this prototype yet. The
Metal encoder refuses this custom shape; it does not substitute a height field
or a plane. No GPU parity is claimed.

## CPU tracer picture

`validation_draw_test.go` is a draw-tag, CPU-only fixture that assembles one
seeded height field with a marker sphere and sky horizon, then uses Slayer's
existing `trace.Sampled` and colour development. Run only this test through the
Game tests lane:

```sh
SLAYER_OCEAN_STUDY_OUT=/path/to/new-wave-study.png game lane tests -- \
  go test -tags=draw -count=1 -v ./arid \
  -run '^TestWaveHeightFieldValidationDraw$'
```

The output path must be a new `.png` file; without the environment variable,
the test still traces and checks that the frame is nonblack but writes no
image. The fixed scene inputs and known limitations are recorded beside the
validation images in the render gallery. This picture is not a validation of
water optics, horizontal displacement, or Metal rendering.
