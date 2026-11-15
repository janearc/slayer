# Next: a quasar that looks as good as it is correct

The photon ring short was dropped on 2026-11-12: its pictures were
plain. This note is for whoever builds
a better-looking model. The explaining work stays and still holds:

- `rings` measures where each image of the disc appears. The tracer does
  draw the third image, and its width agrees with Johnson 2020's e^-pi.
- `trace` has a new let-go parameter, `thing.Field.LetGo`.
- The reach finding is in `ringshots`.
- The sourced numbers are in `facts`.
- `ladder` has the steps of ten for the pull-back.

## What there is

One model, `template.Quasar`. The light bending, the black and the disc's
beaming are computed. The disc's colour is a black body over a picked
peak of 12000 K, with brightness going as the fourth power of the
temperature. The jets are a cone of noise in a picked blue. Smoke 36
looks at it from inside the disc's outer edge, 10 degrees above its
plane, through a tone curve.

## Why it looks plain

- The disc is flat and pale. Streaks are off ("nought is honest"), so a
  smooth gradient fills the frame. Near the shadow the tone curve takes
  it to white, which reads as plastic.
- The peak temperature is picked so the colour is near white-blue, with
  orange only at the rim. Almost nothing in the frame has a hue.
- There is no surround. The disc sits on a black field of stars, with
  no haze, no outer gas and no scattered light.
- The jets are a noise cone and read as painted. Their beaming is right;
  their shape has no flow in it.
- The camera is inside the disc, so a third of the frame is
  featureless disc face.

## What the app's icon has that this lacks

The icon is Miami's art in `internal/appicon`. It has a streaked, wound
orange disc seen from outside at about 35 degrees. Its jets are knotted
and violet-white, it has a soft haze round everything, and it has
contrast: dark gaps between bright streaks.

## With the knobs that exist (a look, labelled as one)

- Streaks on with shear: `Dressed(1, 1.5)` is the setting `thin.go`
  calls Miami's.
- A cooler peak, for orange.
- The camera outside the disc, about 150 masses out at 30 degrees, as in
  smoke 25.
- Jets on with knots and dimmer.
- A gentler tone curve: a lower `Top`, so the brightest ring is not
  clipped flat.

## Needs new work

- A disc with real structure: spiral density waves and turbulence in
  the medium, not only in the texture, so the far image bent over the
  top carries detail.
- A jet that is a flow: a velocity field with shocks (knots that ride
  out), not a noise cone.
- The surround: an optically thin haze or corona round the disc, lit by
  it, and the broad-line gas.
- A tone map that holds a range of a thousand to one. Perhaps several
  exposures, laid together and said so.

## What I would try first

1. The Miami-styled dressing on smoke 25's camera, at full rays. It is
   an hour's work, and it shows how far the existing knobs go.
2. Density structure in `medium.Disc` (spiral arms with noise),
   because every lensed image inherits it. Check that each smoke still
   draws byte for byte as before (smoke 36's PFM hash is
   756f22e1...e35).
