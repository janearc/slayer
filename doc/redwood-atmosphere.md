# Redwood groves and their atmosphere

These references ground the connection between a grove and its atmosphere.
They supply no runtime constants yet. The [observatory ecosystem](observatory-ecosystem.md)
now generates coast redwoods and prescribed grove mist. Its optical values
are authored appearance controls; these references do not calibrate them.

The species here is coast redwood, Sequoia sempervirens. These references
do not establish the same behaviour for giant sequoia, Sequoiadendron
giganteum, or for every conifer.

## Transpiration and retained moisture

California State Parks' [Survivors Through Time](https://www.parks.ca.gov/?page_id=24728),
under "Amazing Survivors and Helpful Neighbors" and "Components of an
Ancient Forest", describes transpiration contributing to fog and the
forest canopy retaining atmospheric moisture.

This is an interpretive account. Its per-tree daily water figure is
not a calibrated rate for arbitrary specimens. No number from that
page has been adopted as a simulation parameter.

Source key: `ca-parks-redwood-transpiration` in `sources.yaml`.
The page was read on 2026-11-02.

## Fog also arrives from outside the grove

The National Park Service's [About the Trees](https://www.nps.gov/redw/learn/nature/about-the-trees.htm),
under "Role of Fog" and "Root System", describes marine fog reaching
the forest, interception by foliage, water dripping to the ground,
and transpiration returning water to the air.

This supports modelling exchange with the surroundings. It does not
give a rule that turns a tree count into visible mist density.

Source key: `nps-redwood-fog`. The page was read on 2026-11-02;
its displayed update date is 2015-02-28.

## Field evidence and its limits

Petreshen, Dymond, Keppeler, Allen, and Wagenbrenner (2025),
*Fog presence and ecosystem responses in a managed coast redwood forest*,
Agricultural and Forest Meteorology 368, article 110525.
[DOI](https://doi.org/10.1016/j.agrformet.2025.110525).

The [Forest Service publication record](https://research.fs.usda.gov/treesearch/69118)
and its abstract were read on 2026-11-02. The abstract reports Caspar
Creek observations across the 2020-2022 fog seasons, comparing shoulder
and ridge positions in harvested and unharvested third-growth forest.

It reports reduced transpiration during fog, variable soil-moisture
responses, and effects influenced by topography but not apparently
forest density at those study sites. This is evidence about responses
to fog; it does not demonstrate that more trees produce more fog.

Source key: `petreshen-2025-redwood-fog`. The PDF is archived separately
on the source shelf; its first page was checked for publication and
rights information. The full paper has not yet been reviewed for
equations, measured rates, or parameter extraction.

## What the recipe may choose

A grove atmosphere may use canopy structure, ground moisture, air
temperature, humidity, wind, and terrain as inputs. That is a proposed
model. These references alone do not determine its equations, mist
extinction, droplet sizes, or moisture source rates.

Keep geometry complexity separate from the grove's physical description:
a change from foliage clumps to needles must not change the assumed
canopy or water exchange merely because it changes the triangle count.
This is a software requirement, not a finding of the cited studies.

Before a measured value enters code, record its units, species, site,
conditions, source key, page, and supporting text in
`research/research-derived-data.yml`, following that file's convention.

Derived values need their derivation; visual choices need their author
and rationale. A plausible-looking fog field is not a measurement.
