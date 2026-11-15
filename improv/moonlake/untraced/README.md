# untraced

"slayer: untraced": moonlake's light show in four panes, in step with the
song.

## The panes

- **The show, rendered.** The finished 4k film, read at the moment and
  cropped to fill its pane (the film is the 15/70 aperture, about 4 by
  3; the wide pane is 16 by 9). Always the show as rendered.
- **The explaining pane.** The explainer's beat squares and bar, its six
  lamps and its sentence (package `explain`, its script `explain.yaml`),
  drawn here to fill the pane: the lamps in one row, the type sized to
  the pane. The lamps' levels are read off the cue sheet's moments as
  the explainer reads them.
- **Above, north up** and **the ridge, moon behind.** The walkthrough's
  stills from those cameras, lit for inspection and graded down to a
  dark blueprint, with the rig drawn on them from the cue state at the
  moment. Tagged as drawn.

  Every fixture is marked in its kind's shape and its lamp's colour and
  named once; each laser beam, searchlight, strobe flash, backlight and
  the ship's cone is drawn where the show sends it.

The geometry the drawing uses is package `rig`: the same functions
`lightshow` lights the show with, moved out of it. Nothing is placed by
eye.

## Traced panes

With `-passes`, the two cameras' passes (the plate, the backlights
alone, the overhead's daylight), traced at the 4k plates' settings, are
summed at the cues' levels by `rig.Layer`, the sum `lightshow` uses, and
developed as the film was: the set rendered from that camera with the
show's light on it.

The beams in the air stay drawn over it, and the pane says so:
"Rendered: the set by the show's light" and "Drawn: beams. Not shown:
water, moon". The show lays its water and the moon's disc in from the
viewer's camera only.

A traced pane shows the rendered set when the show's light falls on it
(`Rendered`): a backlight flashing or the ship's light on; the drawing
otherwise. Under the moon alone the set traces nearly black from these
cameras: the moon, 2.3 degrees up, grazes the land.

## What full rendering from another camera would take

`lightshow`'s camera is the show's eye only (height, tilt and focal), its
masks assume a level horizon, and `seamatte` and `mistlake` take the
water from horizon angles. For the beams, haze and water to be rendered
from another camera: a general camera in `lightshow`, a sky mask traced
from that camera, and the water's plane from the camera; a day or more,
checked frame for frame against the 4k film.

## Tests

`untraced_test.go`: a beam starts at its head's pixel and runs along its
way, a point off it stays dark; the lasers draw only when cued, in the
cue's colour; each other light draws when on and not at nought; the cue
lines are read and a wrong number refused; a pane at another moment is
refused; the film is read at a size that keeps its shape; the traced sum
brightens when a flood fires and refuses a pass of the wrong size.

## Moves and accents

The drawn panes' windows move inside their stills (`moves.go`): in on
the fixtures the panel is talking about, and out again, each move over
one bar from a bar line, eased, the two panes never at once. When a
fixture fires its mark pulses in its lamp's colour in both drawings, as
its lamp lights in the panel.

## What a second show would supply

The general parts are `improv/walk`: the layouts and fits, captions,
the projection of a place to a pixel through a set's camera or an eye,
and the drawing helpers. What is moonlake's is this command and
package `rig`. For another show to have an untraced of its own:

1. **The rendered film**, the show as the audience sees it, readable
   by `interfaces/movie`, and its song.
2. **Two or more cameras on the set and a still from each**: a set copy
   per camera with a rig (frame and lens) and the traced still, or an
   eye and a pinhole for a scene that is not a set; and where its named
   things stand (`hillside -places`, or places in the scene's units).
3. **The fixtures: where each stands and where it points**, as a
   function of the show's state, in one package the show's own
   lighting code calls too (as `rig` is for `lightshow`), so a drawn
   beam is never placed by eye.
4. **The cue state over time**: for a moment, each fixture's level,
   colour and aim (moonlake's is `show.Sheet.At`).
5. **The panel's parts**: the clock (bar and beat), up to six lamps with
   their levels, and the script of sentences (package `explain`).
6. **Optionally, traced passes per camera** for the rendered mode: the
   plate and one pass per fixture whose light falls on the set, traced
   at the show's own settings, and the sum the show uses to light them.
