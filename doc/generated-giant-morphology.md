# Generated giant morphology: a builder slice

2026-11-27. Research and design, not a claim that this feature has been built.

## Aim

A system constructor should make distinct giant worlds from a description and
a seed, then let cameras find them from any side. The first slice makes the
existing gas-giant generator express different arrangements of bands and
storms; it is not a simulation of atmospheric fluid dynamics. Its immediate
purpose is to make memorable, art-directable images for investor meetings.

## What is already there

At rc-v0.1.9, `scene/generate/make.go` constructs gas through `stripes`:
eleven equally spaced cream and rust/ochre paints, eight to twelve jets and
fixed roil, fine noise, mottle, shear and grain. The seed selects two dark
colours and jet count. It supplies no spots or streets. This is a narrow
generator choice, not a limitation inherent in `texture.Banded`:
`scene/texture/banded.go` already supports seeded roil, fine detail, mottle,
latitude jets, explicit oval storms and seeded streets of storms, evaluated
in the body's frame. `bodies.NewGas` holds this texture in a giant medium.
The first builder should follow the existing texture through the native draw
path before promising identical rendering capabilities in either backend.

## Evidence and what it justifies

* NASA/JPL's Juno close-flyby image shows belts, zones and visibly structured
  clouds, and identifies the image as processed to enhance detail and colour.
  It is a reference for differences in scale, shape and organisation, not a
  source for a universal palette or the appearance of every giant:
  https://science.nasa.gov/photojournal/nasas-juno-mission-images-jupiters-belts-and-zones/
* Michelic, Kenzel, Haubenwallner, Kerbl and Steinberger, *Real-time
  Rendering of Procedural Planets at Arbitrary Altitudes* (I3D 2019 poster),
  consider terrain, ocean, clouds and atmosphere together and transitions
  between ground and space. This motivates multiple camera distances and
  coherent world-space features, not a claim that their method is implemented:
  https://www.cg.tuwien.ac.at/research/publications/2019/kerbl_2019_planet_poster/
* Schneider and Vos, *The Real-time Volumetric Cloudscapes of Horizon Zero
  Dawn* (SIGGRAPH 2015), separate low-frequency mass and high-frequency
  detail, and use coverage and cloud-type controls with artistic overrides.
  This motivates separate large storm structure and fine detail; their Earth
  sky model is not a Jovian dynamics model:
  https://www.guerrilla-games.com/read/the-real-time-volumetric-cloudscapes-of-horizon-zero-dawn

## Proposed first slice

Extend the constructor's giant appearance description with optional,
independent controls for a band paint sequence, seed, roil/grain/shear/fine
and mottle, jet count, and explicit storms and storm streets using the
semantics already represented by `texture.Banded`. Allow zero storms, several
explicit storms, or streets, without inferring them from temperature or mass.
Keep old behavior when omitted. Distinguish a fixed, authored spot from a
seeded street in the description and the draw's provenance. Validate finite
values and meaningful ranges at the same boundary as existing body inputs.
Use existing constructor and Pzm conventions rather than adding another parser.

This is intentionally a bounded feature: it exposes and composes existing
capabilities first. A later slice may generate irregular band spacing, stronger
interacting vortices, polar structures or cloud relief, but none is promised
by merely changing the palette. In particular, the existing `Spot` swirls a
paint coordinate and tints it; it does not calculate a moving fluid vortex or
guarantee geometric relief. Labels must say so.

All chosen palettes, counts, dimensions, turn strengths and arrangements
for imagined planets are **made up** artistic choices. The reference informs
the visual vocabulary; it does not supply those numbers. Keep the seed,
parameters and version with the set so the same body is seen from each camera.
Do not randomize independently for each camera. A changed seed should alter
the intended seeded structures while preserving explicitly placed storms.

## What the researcher should look for

Render at least two independently described giants, using the same general
constructor: one with broad contrasting belts and a dominant storm plus a
street; another with a different number of bands, colours and multiple smaller
storms. Include a full disc and a closer oblique view of the same generated
body, and a second longitude showing that the storm placements belong to the
world. Inspect every frame at full size. Ask whether storms read as swirls
rather than flat tinted ovals, bands have variation at more than one scale,
and the camera transition preserves positions without seams or surprising
changes. Record deficiencies and send the builder back; passing an automated
check does not establish visual quality. These are proposed visual exercises,
not completed renders or a claim of scientific accuracy.

## Sources and retrieval

`sources.yaml` is Slayer's source manifest; its header describes the existing
`game run sources-fetch` shelf workflow and checksum pinning. Before building
from these references, add source entries only after checking the actual
retrievable bytes, terms and license, and record the resulting SHA-256, byte
count and fetch date. Prefer the author's stable PDF for the cloud presentation
and the official NASA page or its individually identified image product; the
NASA image credits both the mission and its named processor. Do not assert
redistribution rights from the presence of a public image. Reuse the existing
fetcher and its polite behavior if applicable; inspect it before asserting
specific rate limiting or conditional-request behavior. This note has not
fetched or pinned those files.
