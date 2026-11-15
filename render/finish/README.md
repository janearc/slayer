# finish

The darkroom end of a film. A frame comes in as linear light and goes out
as the picture a person watches, with words drawn on it. The package knows
nothing of how the frame was traced, so a frame made by another tool takes
the same road.

It reads a job (package `job`) only through its accessors: `Print`,
`Stock`, `Titles`, `Scene().Frames()` and `Film()`.

## What it holds

- `Developer`: one frame's light to a picture. A job's `print` is a
  `render/colour.Tone`. A job's `stock` is a ninjafilm roll, one roll for
  the whole job, then brightened to be seen (see below). A job that says
  neither uses the tone the caller passes. A stock not on the shelf is
  refused and the refusal lists the shelf.

  Several glimpses of a frame (motion blur) go to the stock together,
  which is what `Roll.Print` is for; a plain tone takes their mean in
  linear light (`Mean`).
- `Titles`: a job's `titles` drawn on a developed picture at a film time,
  through package `hud`. Only the screen kinds (text, frame, bar) are
  taken; a role the theme has not is refused by name when the titles are
  built, not on the frame that first shows it.
- `Plates`: frames already rendered, in a directory. `.pfm` is read with
  package `raw` and is linear light. `.png` is decoded and each code value
  turned back to linear with `render/colour`'s `Decode`; a PNG's light is
  already clipped at white, so no exposure brings back what it lost. Name
  order; one size; one kind. Frames are read one at a time, not all at
  once.
- `Fit`: frames in that are not the film's size. When the job's film says
  a size the plate is scaled to cover it and the middle is kept; the
  `Cut` says what was lost. A film that says half a size is refused.

## How a stock is brightened

`Roll.Print` returns transmission: the light a projector throws through
the frame, never more than one, and dim (ninjafilm/doc/filming.md). It is
developed for the screen by `ninjafilm/colour`, with the exposure that
puts a white card filmed at the stock's rating on the screen's white.

Those are about 2 for `soot`, `ash` and `smoke`, 4 for `dusk` and
`lantern`, 16 for `pollen`, from that document. A test holds that every
stock on the shelf has one.

The print holds more than that white card. Where light was far brighter
than white the film clears to the least density of its curves
(`Roll.Clearest`): for `dusk` and `lantern` two and a half times the
screen's white (2.52), for `pollen` one and a fifth (1.20), for the
silver stocks a quarter of a percent past it (1.0024), where their
shoulder moves no byte.

So the screen has a shoulder (`ninjafilm/colour`'s `Squeeze`): it lands
the clearest on white and rolls what lies between off by luminance,
bleaching colours too strong to fit toward white.

The knee is `shoulderShare` of the stops the clearest holds above
white, that many stops below white: 0.69 for `dusk` and `lantern`, 0.93
for `pollen`.

Below the knee a print is shown as it was, so a picture whose
luminance stays under the knee prints byte for byte as it did before
the shoulder, through every stock; a test holds the bytes of one whose
brightest part is 0.87 of white. Between the knee and white a print is
a little darker than it was. A stock whose clearest is not past white
has no shoulder.

The share, 0.4, was picked on the night fire through `lantern`: 0.6
also moved pictures with nothing past white, and 0.25 left less room
above the knee for the highlights.

What the shoulder cannot do is find detail the film has already lost.
Pushed three stops, a fire's light lands far past the end of `lantern`'s
curve, where the film is nearly clear whatever the light, and it comes
out pale and nearly flat.

Without it the screen clipped each part of the light at white by itself.
A fire's red and green are past white and its blue is not, so every
flame came out one flat yellow, at any rating.

The frame number is the film's frame index, so a frame's grain is the same
each time it is drawn and new on the next frame.

## Not here

No tracing, no files written (tests aside), no encoding, no sound. The
movie and the sound are another package's.
