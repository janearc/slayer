# ninjafilm’s design

A renderer supplies linear light. ninjafilm gives that light the response
of photographic film: limited spectral sensitivity, a characteristic
exposure curve, and grain. It uses Slayer’s maths and libtheme’s colour
primitives and does not render geometry.

## Film for the hills

The stocks evoke the cameras carried into the hills: blue-sensitive plates,
red-sensitive black-and-white film, dyed mosaics, two-colour prints, and
later slides. They share one model:

1. Choose a stock by name and load a seeded roll.
2. Render the camera’s view at instants during each shutter opening.
3. Give those renders to the roll to integrate and develop.

[Filming](filming.md) covers the calls. This document explains the model
behind them, including its approximations.

## Primitives

Each primitive has a package; [layout.md](layout.md) maps them.

| Primitive | Meaning | Basis |
|---|---|---|
| span | Wavelength interval in whole nanometres | libtheme wavelengths |
| crystal | Developed grain, specified by diameter | Nutting’s law |
| curve | Density as a function of exposure | Hurter–Driffield response |
| record | Sensitive span, image span, crystal, and curve | The preceding primitives |
| lay | Stacked records or a side-by-side mosaic | Seeded site selection |
| stock | Records and layout on a film gauge | `data/gauges.csv` |
| roll | A seeded stock with an exposure rating | Stock data and frame number |

Stock names select table rows, not separate rendering algorithms. Adding a
stock changes data, while adding a new physical process may require code.

## Spectral sensitivity

A record responds to a wavelength span. The current RGB representation
approximates each channel as one monochromatic source at its dominant
wavelength: 611, 549, and 464 nm. These are computed from the intersection
of a ray from white through the primary with libtheme’s spectral locus.

A channel exposes a record only when its wavelength lies inside that
record’s span. Real reflected red light spans many wavelengths, so a stock
blind above 600 nm might still record some of it. This model records none.
Spectral-band input would allow the spans to replace that approximation.

An earlier design spread each RGB channel across the spectrum using its
share of flat white. Tests rejected it: yellow contributed to red, slides
printed red as brown, and mosaics printed it as yellow.

## Exposure to density

The Hurter–Driffield curve relates density to logarithmic exposure, with a
roughly straight middle and curved ends. Here it is a logistic in log
exposure. Its middle slope and minimum and maximum densities come from
the stock data.

The model represents the projected positive: density falls as exposure
increases. For negative-and-print workflows, one effective positive curve
uses the combined slope, while grain follows the camera stock.

Reference exposure uses the grey halfway along libtheme’s lightness scale.
When that grey is lit to reproduce itself, it falls at the curve’s midpoint.
This is distinct from an 18% photographic grey card.

## Density to grain

Nutting’s law (1913) relates density D over area A to N grains, each casting
shadow area a:

```text
D = log10(e) × N × a / A
```

For a pixel’s film area, the model draws a grain count from a normal
distribution with mean N and standard deviation √N, then converts it back
to density. This approximates a Poisson count when N is sufficiently large.

Grain needs no separate strength control. Larger crystals or smaller film
areas per pixel produce fewer grains and greater variation. Roll seed,
site, and frame number make the variation reproducible.

The normal approximation becomes poor below roughly ten expected grains.
For crystals wider than a micrometre and pixels narrower than three
micrometres this matters; the supplied gauges approach that regime at
roughly 8,000 pixels across.

## Developed density to light

Each developed record attenuates light by `10^-density` within its image
span. Silver blocks all three RGB channels; dyes block only channels
within their spans. Transmission through stacked records multiplies.

For a mosaic, each site contains one record selected by the seeded hash.
Only channels within its window expose it, at full strength: the stock’s
rating already includes input filtering. Output passes through that same
window, where the plate’s transmission loss is applied.

Pixels larger than one mosaic element average the sites they cover.
Element width and plate transmission derive from two published parameters:
elements per square inch and the fraction of light transmitted.

## Public API

| Call | Result | Errors |
|---|---|---|
| `Stocks()` | Available stock names | Unreadable stock tables |
| `Roll(name, seed)` | Seeded roll at its rating | Unknown stock or invalid tables |
| `roll.Rated(stops)` | Adjusted exposure rating | No immediate error |
| `roll.Gauge()` | Film gauge for the lens | None |
| `roll.Print(which, glimpses)` | Developed frame | Invalid roll, absent or mismatched frames, empty frames, nonfinite light |

Obtain a usable roll through `Roll` and check its error. The zero value is
not a loaded stock. Invalid rating values are reported when the roll is
used for printing.

## Model limits

- Toe and shoulder use the logistic shape, not measured stock-specific curves.
- Halation in the film base is omitted.
- Mosaic lampblack is represented only through overall transmission loss.
- Spatial sharpness follows the input render; no film MTF is applied.
- Tinted and toned prints are omitted because suitable published dye
  densities were not established.

[Sources](../data/SOURCES.md) distinguishes fetched, search-derived, recalled,
and chosen values. Those categories should not be treated as equally verified.

## Validation

External tests check functions and use Rapid for properties over ranges.
The recorded statement-coverage target is 100%.

`game mutants` exercises operators, integer constants, and booleans. A
surviving mutation needs either a stronger test or an entry in `.mutants`
explaining why it is equivalent. See [checking.md](checking.md) for limits
and resource controls.
