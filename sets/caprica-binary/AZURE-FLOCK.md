# Azure's local flock: first composition

This is an unrendered composition, not a claim that the twelve silhouettes,
both stars or clouds are visible. It requires the detailed Hatzegopteryx
recipe adapted at `ee10c3d09`; the detailed painted membranes
currently require CPU. The twelve animals have different positions, headings
and seeds but share the built-in fixed wingbeat phase. Flight in Azure's
atmosphere, formation, cloud banks, colour and exposure are artistic choices.
The 55-degree camera and staggered 82–150 m distances aim for separately
readable animals and include the two stellar directions; the first preview
must establish whether that composition succeeds.

The primary, white dwarf, Fresno and Azure use the same authored snapshot
as `working-system.yaml`, with the coordinate translation documented in
`AZURE-SHOT.md`. The camera is 30 m inside the nominal generated gas
volume, whose density is almost zero at that boundary. The two added wisps
are made-up local cloud proxies, not simulated giant weather. Neither their
appearance nor star visibility is established before rendering. Caprica,
Amber, comets, wake and aurora are intentionally omitted from this local
composition to avoid very distant reflecting geometry disrupting metre-scale
ray offsets. The binary constructor still includes Fresno. This does not
establish that omitted bodies would have no effect on global illumination.

Local animal coordinates avoid one source of large-coordinate precision
loss, but Azure's centre remains roughly 46.7 million m away and the stars
are trillions of metres away. Boundary precision, star visibility, medium
lighting, animal pose and exposure need observation. The renderer's minimum
stellar image size may enlarge distant stars; this is not a physical angular
size guarantee. No moving orbit, physical flight or complete rebasing is
claimed. Preserve the exact original system draft separately.

`azure-flock-plain-gpu.yaml` is a separate GPU diagnostic with identical
composition and `detailed-plain` animals: their membrane paint is omitted,
while the detailed model and its hide paint are retained. The corresponding
job uses GPU-native rather than CPU. Backend lighting, sampling and numerical
behaviour can differ, so the pictures are not an isolated comparison of
membrane paint. The detailed CPU reference remains intact and unsubmitted.

The first GPU preview (job 565) showed a crowded star field, tiny dark
animals and a smooth blue lower band; neither individual stars nor twelve
animals could be identified. `azure-flock-plain-gpu-v2.yaml` is a new,
unrendered attempt: one animal is 25 m from the eye and eleven are 52–75 m
away; the wider 65-degree composition is rearranged, the artificial star
field is much sparser, and the two local cloud wisps are moved nearer and
thickened. An invented directional key and cool fill light the flock and
clouds as photographic art direction; they do not represent radiation
from the binary. The generated binary and its translated positions are
unchanged. Both actual stars may remain invisible or indistinguishable;
the preview must show whether these changes work. The GPU membrane-paint
omission and other backend limitations still apply.

Job 566 (v2) clipped most of the image white, including foreground animals;
the log reported no renderer failure. The raw PFM median channel was 7.69
and its 90th percentile 144.84, versus 0.0048 and 0.0985 for job 565.
The much stronger photographic lights and denser, nearer wisps were changed
together, so the image does not isolate the contribution of either. V3 keeps
the camera, formation, exposure and star geometry, reduces key and fill by
100 times, and quarters wisp extinction; it is an unrendered artistic
correction, not evidence of a physically modeled atmosphere or light.

For the v2/v3 960×600 camera, a separate pinhole projection of the
translated directions puts the primary at approximately (337, 344) and
the companion at (620, 344), using a top-left image origin. Both are within
the field, absent occlusion. `session/stars.go` applies a minimum two-pixel
drawn radius and reduces the glow to conserve light; the companion's authored
0.0122-solar radius and 7400 K imply about 0.00040 solar luminosities,
much fainter than the 50.63-solar primary at comparable distances. The
clipped v2 frame does not resolve either star's actual visibility or cause
of any absence; projection alone does not demonstrate that it is rendered.

Job 567 brought the animals into view, but an independent observer counted
only 11 and described rigid rows, a flat pale lower half and two tiny points
that do not read as suns. V4 is a new artistic composition: a closer 18 m
foreground animal; individually varied depth, heading and height; an
invented blue gradient and two smaller, patchier cloud wisps. Heading is
yaw only: all animals retain the same fixed wing phase and this does not
implement banking. The generated binary remains in place, but two separate
nearby emissive spheres align approximately with its projected directions as
deliberately enlarged theatrical sun-disc cues. They may occlude the actual
stellar images; their brightness and angular size are invented, and they do
not establish actual stellar visibility. No visible cloud detail, twelve
distinct silhouettes, or final aesthetic result is claimed before rendering.

Job 568 reduced the grid but remained small and repetitive, with a smooth
blue background and two white cue dots on the horizon. V5 moves the camera
20 m upward, so it is about 10 m inside the nominal giant boundary, and
places a near animal 12 m forward with the other eleven in a staggered
diagonal at varied heights and headings. Those poses still share a phase
and cannot bank. The exaggerated artistic cue spheres move upward away from
the physical stars' directions and use different sizes and warm/cool colours;
the binary itself is unchanged. They are theatrical placements, not actual
apparent positions, sizes or brightnesses. Small authored wisps are denser,
and another is added high in frame; any resulting cloud detail is unverified.
