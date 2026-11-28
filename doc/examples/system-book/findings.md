# System candidate findings

The unchanged working-system snapshot and preview jobs produced 595–601
at 640×400, one ray. The transfer preview showed a clipped light and a
weak repeated stream; capture showed a bright white patch in brown haze.
The crown shows a cropped banded giant with soft blue/violet wisps, but has
not yet shown the requested yellow Amber. The tail was a smooth gradient,
Azure was black, Fresno was clipped, and Fresno from Caprica was a bright
small point beside a dark sliver. Independent inspection did not find a
complete system story in these frames.

Separate development of existing raw made 603's transfer a row of glowing
discs, 604's capture entirely white, 605's Azure a dark green/brown cropped
disc and 606's Fresno bright with clipped crust. The development settings
are in the separately named jobs; these are diagnostics, not accepted views.
609's wider Fresno camera with −24.8-stop print showed a small dark red
globe and bright seams, but its haze was not legible. A −20-stop print of
the same raw, 613, exposed brighter red/orange plates but clipped the seams.

Amber 607 and 616 used the same camera position but changed only the camera
lens's `wide` and job exposure. Their raw PFMs have identical SHA-256
`d57cd271c51d235d46db85bac311aedb7e38fec2554144942568aba70fddc656`;
their prints differ. Both logs report tracing rows. In the world-only path,
`session/world_only.go` takes lens width from the film and does not call
the `lensOf(shot)` logic used by the other scene path. This is evidence for
why changing only camera width did not change those raw images, not a proof
about all caching. The later jobs set `film.wide` explicitly. 619 then fits
Amber's disc, but is dark with a blown rim and dense white speckles.

For a bounded visibility control, 628 removed only `wind-core-01`,
`wind-sheath-01` and `wind-sheath-02` from a copy of 619's scene. Their
bounding spheres contain the nearby Amber camera; all other scene settings
were retained. Independent inspection found much less violet background haze
at right, but the dark globe, blown rim and dense speckle persist, so removing
these three did not resolve the portrait. The raw
PFM hashes differ, so no raw identity is claimed. This does not identify
the remaining cause. Earlier 627 tried zero thickness but was refused by
the existing positive-thickness rule; its input was preserved.

The primary-facing camera in 614 shows the actual Azure as a cropped blue,
softly banded globe. 620 uses a farther camera and explicit 55° job width;
the whole blue globe is visible but small. Its source scene and still epoch
are unchanged. All outputs have raw PFM except developed print-only frames,
which retain links to their original raw sources in their queue requests.

The simplified Amber comparison 631/633 omits all wisps, auxiliary lamps,
other planets and comet pieces in both scenes, retaining Amber and the
binary/Fresno. The second scene translates the camera, body and drop anchor
by `[+6862930.645479282, -2228182.686697275, 0]` million metres. Its Amber
center is the global center divided by 10000, preserving the template tail's
nominal direction away from the origin. Decimal inputs are exact to their
written digits; numerical and lighting equivalence are not established.
The initial local version at exact origin was refused because the tail had
no direction. Default prints were clipped; matched −18-stop developments
634/635 revealed similar cream/tan banded globes with bright halos. This is
one visual comparison, not renderer parity. 636 uses the simplified local
scene at 800×800, two rays, a 45° job lens and −18-stop print. Independent
inspection found a complete, well-framed globe with soft bands and halo,
usable as an identification view with its omissions disclosed.

637 is a new camera in the full working system, looking at the actual
constructor-placed Caprica and Fresno. Both bodies are visible and separated;
independent inspection found Caprica's upper crescent and faint right-side
clouds, but most of its disc is very dark. Fresno is small and orange-edged,
with a clipped white interior and no readable lava texture. This is a useful
provisional relationship study, not a detailed portrait. Positive-pixel raw
luminance has median about 0.0505 and maximum about 2.66×10⁷, indicating the
extreme range a global print must handle. The candidate brighter print is a
separate development of the retained raw, not a new illumination model.
