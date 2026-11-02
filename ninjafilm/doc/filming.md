# Filming with ninjafilm

A clean render is only the beginning of a film image. The cameras carried
into the hills were hand-cranked, their shutters closed for part of each
frame, and their stocks often insensitive to red.

ninjafilm supplies that photographic response. You bring the scene,
camera, and renderer; a roll integrates glimpses taken during a shutter
opening and develops them into a frame.

## What you need before you start

| thing | whose | what it is |
|---|---|---|
| a world | slayer, `scene/thing` | the set: things, lamps |
| a camera | slayer | where the eye is at each instant the shutter is open |
| a renderer | slayer's tracer, or slayer `gpu` | gives a `*frame.Frame` of linear light for an eye and a time |
| a roll | ninjafilm | a stock, loaded |

## The rendering loop

The camera is slayer's. It hands each glimpse of a frame to the
renderer, and the glimpses of one frame to the roll:

```go
// a roll of a stock. the seed is which roll.
roll, err := ninjafilm.Roll("ash", 1921)
if err != nil {
	return err
}
// the roll makes a frame of film of what the renderer gave.
printed, err := roll.Print(number, glimpses)
```

Use a stable frame number to reproduce that frame’s grain. The exposure
section below explains how to rate and display each stock.

## The shelf

`ninjafilm.Stocks()` lists the names. These are the six, in the order
the hills got them.

| name | what it is | what it sees | how it prints | grain | cut to |
|---|---|---|---|---|---|
| `soot` | one silver record | blue only | red and green things print black. A blue sky prints white | fine | super 35 |
| `ash` | one silver record | blue and green | red prints black: a red sash, blood, a setting sun. The stock of the 1910s | medium | super 35 |
| `smoke` | one silver record | everything | greys as the eye would rank them. The 1920s | coarse | super 35 |
| `dusk` | two dye records, a red one and a green one | red and green | in red-orange and blue-green only. It never saw blue: a blue thing prints as a black one does | coarse | super 35 |
| `pollen` | three kinds of dyed element scattered over one silver record | everything, a third of it at each element | full colour, dim, with a coloured speckle you can see | fine | full frame |
| `lantern` | three dye records, one under another | everything | full colour, the deepest blacks, the least grain. The slide | finest | full frame |

A stock is rows in two tables, `data/stocks.csv` and `data/records.csv`.
a new stock is new rows; no code changes. `data/SOURCES.md` says where
each number was published, and which were fetched and which recalled.

## Exposure, and the screen

A stock is rated for a middle grey: a surface that gives back an eighth
of the light of a white one lands on the middle of the stock's curve. If
your scene's light is in some other unit, the picture is too thin or too
dense, as it would be with the wrong aperture.

That grey is libtheme's, halfway up its lightness scale. It is not the
photographer's grey card, which gives back 18 percent and is half a stop
lighter. If you light a scene by a grey card, rate the roll half a stop
under, `roll.Rated(-0.5)`, and the card lands on the middle.

```go
roll = roll.Rated(1.5)  // a stop and a half over
roll = roll.Rated(-2)   // two stops under
```

What `Print` gives back is the light a projector would throw through the
frame, with a white lamp behind it. It is linear, and it is never more
than one. It is not bright: film at its clearest still holds some
density, and `pollen` passes less than a tenth of the light at its best.

Turn the projector up with `colour.Develop(printed, exposure)`, or a
`colour.NewTone`. The exposure that puts a white card on the screen's
white is about 2 for `soot`, `ash` and `smoke`, 4 for `dusk` and
`lantern`, and 16 for `pollen`.

Do not also give the picture the grain of `colour.Grained`. The gauge
of a roll comes with its grain turned off for that reason.

## Grain

You do not set it. It comes from three things:

- The stock's crystals: coarser ones are noisier
- How wide your picture is: grain is counted over the piece of film one
  pixel covers, so a 640-wide picture of a frame is far cleaner than a
  4096-wide one, as a small print is cleaner than an enlargement
- The density: a dark part of the frame holds more crystals

The picture's width is taken to span the gauge's width. A roll and a
frame number give the same grain for ever; the next frame's is new.

## When it refuses

Check loading and printing errors before using their results. The
following messages identify the failed condition and expected input.

| the error says | it means | do |
|---|---|---|
| `no stock is called "x"; the shelf holds [...]` | a name not on the shelf | use a name from the list it gives |
| `a roll comes from Roll, and is rated within what a number holds` | the roll is a zero value, or `Rated` was given something absurd | check the error from `Roll`; rate in stops, single digits |
| `a frame is printed from one glimpse or more` | `Print` was handed no frames | render the glimpses first |
| `a glimpse is a frame of one pixel or more, with a light for every pixel` | a nil or empty frame | the renderer failed; check its error |
| `the glimpses of a frame are all one size` | two renders of different sizes | one width and height for the whole reel |
| `a glimpse holds light that is not a number` | the renderer returned a NaN or an infinity | fix the scene; ninjafilm will not hide it |

## Model limits

- Sharpness: the frame is as sharp as your render
- Halation, the glow round a bright thing
- Tinted and toned prints
- A red-blind stock seeing a little of a broad red, as a real one did:
  here it sees none

doc/design.md says why, for each.
