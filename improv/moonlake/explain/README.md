# explain

A light show's explainer frame: the finished film, whole, with a panel
under it that shows the machinery playing it. A stranger should see that
every light in the picture is fired by the song's own beats and notes,
and see it happen. Moonlake is the first show to use it; its parts are
in `improv/moonlake/explainer/rig.go`.

## What it composes

Given a moment and the film's picture for that moment, the package
returns one 1080 by 1920 picture:

- Top: the film, whole, scaled to 1080 across with its shape kept (4096
  by 3008 is 1080 by 793). Not cropped.
- Under it, the panel, read top to bottom:
  1. The beat. A square for each beat of the bar. The beat now flashes
     and falls back; the downbeat's square is the ink colour, the
     others green. A beat counted on through a break, not heard, is
     drawn hollow. Beside the squares, the bar: "Bar 25".
  2. The lamps, one to six, in rows of three. A lamp is as bright as
     its light is this frame; its ring flashes when the show says a cue
     for it fired.
  3. The chant, while a line is sung: the line whole, large, in ink.
  4. One plain sentence from the script, which changes where the
     script says.

Fewer, larger things. Nothing is drawn under 42 pixels high on a
1080-wide frame, which is 14 pixels on a 360-wide phone.

## What a show supplies (Parts)

- `Clock`: which bar and beat a moment is (`NewBeat`, or `OffGrid`
  before the grid), and how many beats its bar holds.
- `Lights`: one to six lamps (`NewLight`): a label, a colour, a level
  reading and a ring reading, each a function of the moment.
- `Ground`: the panel's ground at a moment, or nil for the hud theme's.
  `Tinted` makes a legible one leaning toward a colour.
- `Script`: the sentences, the chant's lines and the shorts, read from
  a yaml with a `Placer` that turns "bar.beat" or seconds into seconds.

## Type and colour

From the house legibility notes: sentence case, no letter-spaced
capitals, Go's proportional face (not the hud's monospace) at weight
400. Ink is the hud theme's cream.

The ground may lean toward any colour. `Tinted` mixes the theme's navy
toward it in linear light, then scales the result until the ink stands
on it at the theme's own contrast (11.7 to 1), inside the 10 to 12.5
band. If the result would be within 40 degrees of the ink's hue and
saturated past 0.35, the tint is halved until it is not. `Legible`
checks both rules, and the tests run it over every colour.

## The pure parts, each with tests

- `Script.Says`: the sentence standing at a moment, the latest at or
  before it. A line at a place the show lacks is refused by name.
- `Script.Chant`: the chant's line, up from its start to its end.
- `Script.TooFast`: every sentence inside a short fits its time at
  `readingRate`, characters a second.
- `Tinted` and `Legible`: the ground's arithmetic.

The drawing is checked by looking.

## How the chant is timed

By the song's own lyric track: each line's start and end as the track
gives them. The transcribed chant notes were tried first and are too
sparse: moonlake's opening lines have gaps of two to four seconds inside
them, so a line timed by its notes went down mid-line. No word within a
line is singled out, since nothing times the words.
