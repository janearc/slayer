# Parameter sources and verification status

This record traces values in `stocks.csv`, `records.csv`, and the code
to their cited origins. It preserves the original collection status;
this documentation edit does not newly verify the sources.

- **Fetched:** read from the source on 2026-10-06 by the implementation author.
- **Found:** obtained from a web-search summary on 2026-10-06. The underlying
  page was not read.
- **Recalled:** entered from memory of the cited work and not found online.
  Verify these before relying on them quantitatively.

Code and tables use fictional stock names. Real products and photographic
processes remain here where they identify a citation.

## The `source` column

| key | the stock is | rows |
|---|---|---|
| cine | a silver image on cine stock, printed to a positive | soot, ash, smoke |
| pair | two colour records, each dyed and laid back to back | dusk |
| mosaic | a scatter of dyed elements over one silver record | pollen |
| still | three dye records one under another, developed to a positive | lantern |

## What each record sees and blocks: the spans

Lamb's light is three lamps. libtheme puts their hues at 611, 549 and
464 nanometres, and a span either holds a lamp or does not. So a span's
ends matter only for which side of those three wavelengths they fall,
and every figure below has room on both sides.

| span | used for | from | how |
|---|---|---|---|
| 360 to 495 | soot: silver bromide alone sees no further | "sensitive up to around 495 nm" | found |
| 360 to 600 | ash: blue and green, not red | Wikipedia, "Orthochromasia", citing Mees, *The Photography of Colored Objects* (1919): "sensitive to blue and green light but not red light" | fetched for what it sees; the 600 is the thirds below |
| 360 to 830 | smoke, and every silver image: all of it | libtheme's observer, `swatch.Observed()` | computed |
| 360 to 500, 501 to 600, 601 to 830 | the thirds: blue, green, red | the usual division of the visible into three, as in Hunt, *The Reproduction of Colour* | recalled |
| 601 to 830 and 501 to 600 | dusk: a red-filtered record and a green-filtered one; the red record dyed green-blue, the green record dyed red-orange | George Eastman Museum, "Two-Color Process" | found |

The same museum page is why dusk cannot print blue: "incapable of
replicating accurate blues, purples, and yellows".

## The crystals: `crystal_um`

| value | stock | from | how |
|---|---|---|---|
| 0.2 to 1.5 | the range everything here lies in | US patents on silver halide emulsions, among them statutory invention registration H1450: mean grain size 0.2 to 1.5 micrometres, fine grain 0.2 or less | found |
| 0.20 | lantern | the fine end of that range | found |
| 0.30 | soot, pollen | Mees and James, *The Theory of the Photographic Process*, 3rd ed. (1966), table of mean grain diameters: motion picture positive | recalled |
| 0.88 | ash | the same table: portrait film | recalled |
| 1.09 | smoke, dusk | the same table: high-speed roll film | recalled |

Assigning those grain sizes to fictional stocks is a modelling choice,
not a measurement of the stocks themselves.

## The curves: `slope`, `least`, `most`

| value | stock | from | how |
|---|---|---|---|
| slope 1.5 | all | Wikipedia, "Sensitometry": the gamma of a projection transparency is "approximately 1.5" | fetched |
| 0.3 to 2.0 | soot, ash, smoke | the same article's one worked example of a positive film: least 0.3, most 2.0 | fetched; one example, not a survey |
| 0.2 to 3.2 | lantern, dusk | US patent 5,563,027, on a colour reversal film: measured from a density of 0.20 to 3.20 | found; applied to dusk for want of a figure of its own |
| 0 to 2.0 | pollen | least is zero because the plate's own dimness is in `passes`; most is the silver positive's above | derived |

## The mosaic: `per_square_inch`, `passes`

| value | from | how |
|---|---|---|
| 4,000,000 | Wikipedia, "Autochrome Lumière": "approximately 4,000,000 grains per square inch" | fetched |
| 0.075 | the plate "transmits 7.5 per cent of the incident light"; the search's summary did not say which of its pages said so | found |

The same article gives the three dyes as red-orange, green and
blue-violet, the emulsion as panchromatic, and the plate as developed to
a positive, all fetched. The element's width is worked out in code from
the count: 12.7 micrometres.

## The constants in the code

| constant | value | from |
|---|---|---|
| `inch` | 25400 micrometres | the international yard and pound agreement, 1959: an inch is 25.4 mm |
| `perMillimetre` | 1000 | the metric prefixes |
| `halfway` | 0.5 | half of libtheme's lightness scale, by definition of a middle |
| `midway` | 0.5 | the middle of a step, by definition |
| `turn` | 2 pi | a whole turn, in radians |
| the 4 in `curve.density` | 4 | a logistic's steepest slope is a quarter of its height; derived in the comment |
| the 4 in `crystal.shadow` | 4 | a disc's area is pi times its width squared over four |
| `chance`'s 2 and 2 pi | | Box and Muller, "A note on the generation of random normal deviates", *Annals of Mathematical Statistics* 29 (1958) |

## The laws

| law | from | how |
|---|---|---|
| density is a count of grains | Nutting, "On the absorption of light in heterogeneous media", *Philosophical Magazine* 26 (1913) | recalled; it also follows from grains shadowing an area at random |
| density against log exposure | Hurter and Driffield, "Photochemical investigations and a new method of determination of the sensitiveness of photographic plates" (1890) | fetched, by way of Wikipedia, "Sensitometry" |
| granularity falls with the square root of the area | Selwyn, *Photographic Journal* 75 (1935); Siedentopf, *Physikalische Zeitschrift* 38 (1937) | Selwyn's law fetched from Wikipedia, "Film grain"; the two citations recalled |

## For the guides, not the code

| figure | from | how |
|---|---|---|
| cranked at about 16 frames a second, and anywhere from 10 to 26 | cinematographers' own accounts of the 1910s and 1920s | found |
| a studio camera's shutter fully open at 170 degrees | a description of a camera of 1912 | found |
| the first stock that saw red reached cine cameras in 1913, was a regular stock in 1922, and passed the red-blind stock in use in 1926 | Wikipedia, "Panchromatic film", citing Koszarski (1994) | fetched |

## The gauges

`gauges.csv` carries forward Lamb’s frame dimensions unchanged, in
millimetres. They represent published film gauges; this records their
inheritance rather than a new measurement.

The inherited grain setting is visual tuning: a clump around 46–47 µm
expressed relative to frame width. It is not a measured stock property.
Each row’s source column records where ninjafilm obtained it.
