# Caprica system: one authored scene, many cameras

The near-term goal is one mergeable Slayer branch that can describe
the complete fictional system once and render named cameras throughout it.
The intended product is a reusable system constructor, not a hard-coded
Caprica showpiece. The brief calls for investor-compelling images; a passing parser or
a merely recognizable diagnostic is not the visual goal.

## System and artistic direction

- **Caprica** is an Earth-like world and circles **Fresno**, its molten lava
  drop. Fresno has an atmosphere in the story because comets inundated it
  over millions of years. That is authored history; existing haze/wake
  wisps do not simulate atmospheric deposition.
- **Amber** is the yellow gas giant, uncomfortably close to the larger,
  hotter primary star. The preserved candidate is not yet visually proved
  yellow. Its enormous beautiful auroral crown and escaping tail matter.
- **Azure** is the blue ice giant. The brief calls for roughly twelve Hatzegopteryx
  flying high over dark, brooding cloud layers, two distant suns, and a
  noctilucent-inspired contrast: light catches the animals while the lower
  cloudscape remains dark. They must be close enough to read as animals,
  not merely points of light. This is explicitly fantasy, including their
  flight and survival there.
- The binary has an older, cooler white dwarf accreting wind from a younger,
  hotter primary. The brief also calls for a vast persistent procedural comet
  population, infall and breakup, meaningful streams/wakes, and sunrise,
  sunset and Caprica–Fresno views. Do not silently turn static hand-placed
  fragments into a claim that the population or breakup is simulated.

The initial brief asks for procedural planetary/solar systems described by
constructors and viewed by placed/picked cameras; research and cite how
software works, clearly mark invented artistic choices, and make the images
beautiful enough for investors. This is not currently a demand for physical
realism. Publish render proofs through the configured local gallery.
The twelve plates use Ninjafilm
`film: {preset: half-portrait, rays: 3}`: 1080×960, nine samples/pixel,
with separately preserved PFM and developed PNG.

## Source of truth and what exists

- The original draft was preserved in `f4a32ef31` at
  `sets/caprica-binary/system.yaml`. The authored working variant
  is described below.

- `sets/caprica-binary/working-system.yaml`, with `WORKING.md` and
  `DIRECTION.md`, is an explicitly artistic, fixed-epoch edited system based
  on the preserved draft. Amber maps to the original hot-Jupiter body,
  approximately 0.29026 AU from the primary at the selected epoch; Azure
  is a new blue-band generated giant placed 20 AU east of the primary.
  Their locations are fixed authored placements, not implemented orbits.
  The body template's existing tail and spotlighted crown are artistic
  approximations; the crown is not a simulated aurora. Stellar wind/capture,
  comet pieces and Fresno's haze remain artistic or hand-staged as described
  in the source handoff. Do not present them as validated simulations.
- `doc/examples/system-book/plan.md`, `findings.md`, `manifest.json` and
  its scene/job files record system-book render inputs and outcomes. The
  full-system Caprica/Fresno relationship view showed both bodies but the
  Caprica disc was mostly dark and Fresno was small/clipped. A simplified
  local Amber portrait excluded many bodies and wisps and must not be
  represented as a complete-system render. Azure's exterior views showed a
  blue banded globe; cloud detail was soft. Treat the findings as receipts,
  not as evidence that every scene is presentation-ready.
- The in-atmosphere flock is a **separate camera-local artistic scene**:
  `sets/caprica-binary/azure-flock-twilight-v15.yaml`, its named job and
  `AZURE-TWILIGHT.md`; associated `hatz-*.yaml` profiles provide four
  static phase/bank choices. Its binary was translated from the snapshot,
  but Caprica, Amber and other distant geometry are omitted. Nearby glowing
  spheres are deliberately enlarged/displaced theatrical sun-disc cues,
  not the actual stars' apparent positions/sizes. The key/rim/fill lighting
  and blue cloud layer are authored. `giant_gas` opacity 0.0004 is an
  extreme artistic setting: the retained Azure gas may be visually
  negligible. The prescribed cloud deck is not simulated giant weather and
  may extend outside the nominal gas boundary.
- The V15 flock image and sampled companion correspond to the two
  `azure-flock-twilight-v15` jobs in `sets/caprica-binary/`. An
  independent observer counted twelve separated forms and two cues. Higher
  sampling made animal forms more continuous, but regular membrane lines,
  gaps and a blurred, flat-looking cloud bank remained. This is a still
  observation, not a claim of motion stability or a finished hero.

The retained detailed model derives from the detailed pterosaur study (source
commit `e760470a3`, adapted as `ee10c3d09`); it is a speculative anatomy
study, not an approved reconstruction. `detailed-silver` is an invented pale
**reflective**, non-emissive material choice. The GPU-oriented variant omits
posed membrane rest-space paint because that path refuses it; CPU's
`detailed` retains it. Don't silently call the two identical.

## Why one YAML is not yet one reliable system

The set reader already admits named cameras, a binary, world bodies, lights,
recipe-backed animals and finite cloud decks in one document. The missing
work is coherent staging across scales, feature combinations and time, not
merely concatenating those YAML blocks.

1. **Coordinates and frames.** The original draft uses `unit: 1000000`; the
   Azure animal shot uses metre units. `session/world_half.go`'s `rebasing`
   only shifts terrain-backed scenes; ungrounded space is unshifted. At the
   global Azure placement, float32 spacing is far larger than an 11 m
   animal. Translating only the camera or keeping planets in a different
   separately authored scene does not solve it. Camera shot resolution in
   `set/shots.go` and `session/scene.go` supports absolute or placed-thing
   anchors, not a general body-local frame. The current finite
   `world.cloud-decks` route also explicitly requires metre units.
2. **All participants must transform together.** The selected camera's
   local frame must consistently carry body centres/orientations, binary
   stars and lamps, planets and gas media, animals, clouds, optical paths,
   shadow occluders and relevant references. Translation must occur at
   sufficient precision **before** float32 GPU encoding. Do not silently
   drop remote bodies to avoid numerical problems and then call the result
   the complete scene. The existing `scene/thing/ray_reference.go` and
   `trace/stage.go` supply an internal ray-origin reference; they are not a
   complete user-facing multiscale or camera-relative solution. Extremely
   large geometry can otherwise influence ray offsets and shadowing.
3. **Actual illumination versus authored cues.** The binary constructor
   emits actual star geometry and lamps, but the current flock shot adds
   directional lights and false cue discs for composition. Preserve those
   distinctions in the schema and captions. A local frame must preserve
   real directions, relative positions and illumination; extra art lights
   must be named, attributable and optionally disabled. Do not claim that
   observed theatrical discs prove the modeled stars were visible.
4. **Flowing giants.** `session/flowing_bodies.go` currently refuses a
   binary, additional lights, non-metre world units and several other
   combinations; it also refuses film-evolving weather. Existing flowing
   giant shading is precomputed for one light and has explicitly bounded
   shadow support. Do not remove guards to make a scene parse or claim
   multi-star, multi-body or time-coherent volumetric lighting without
   implementing it. The current Azure uses generated painted bands, not
   the single-sun flowing bridge.
5. **Time and moving cameras.** The current working system contains a
   fixed snapshot, not a complete dynamic system. Define body orientation,
   orbital position, local children and camera attachment at a shared epoch;
   where motion is unsupported, say fixed. Recompute per frame rather than
   freezing a transform while calling it an orbit. `session/scene.go`
   refuses a moving job camera with an explicitly declared initial medium;
   keep that refusal until camera-medium membership is tracked correctly.
6. **Medium and resolution limits.** `scene/medium/cloud.go` offers a
   prescribed noise field, not a fluid or physically solved clouds. The
   local cloud-deck route and optional box feather are real 3-D media; their
   GPU generic-volume field is baked onto a bounded grid, which can blur
   large-area detail. A large local layer is an artistic cloud proxy; avoid
   calling it an Azure-wide simulated atmosphere. A proposed opt-in edge
   detail candidate is recorded in commits
   `f3b72a4a7` and `04ce31a2f`, unrendered and not to be assumed integrated.

## Suggested deliverables and acceptance

1. **Schema and authoring:** a single complete system scene names bodies,
   actual stars, local frames, child placements and all desired cameras;
   separate labels for fixed/dynamic and for artistic overrides. Preserve
   the exact original draft in history. Document coordinate units, frame
   hierarchy, numerical precision and every invented material/light.
2. **Camera-relative stage:** implement one bounded fixed-epoch route that
   evaluates a selected local frame and transforms **all** relevant scene
   participants consistently. Preserve visibility and shadows, not just
   subject coordinates. Keep existing refusals for unsupported cases.
3. **Time and lighting:** extend per-body/frame motion and multi-star
   illumination incrementally, with separate supported behavior for static
   snapshots, posed animals, evolving weather, initial media and eclipses.
   A uniform-disc source or artistic fill must not be called a fully
   physical penumbra. Avoid duplicating Melancholia's existing weather
   constructor merely to produce a different palette.
4. **Visual acceptance:** render complementary named
   cameras from the **same authored scene**: system relationship, Caprica
   looking toward Fresno, Amber near the primary, Azure exterior, and the
   Azure flock. Verify that world features retain position across overlapping
   cameras; distinguish artificial markers from weather evidence. In the
   flock view, seek twelve individually legible animals, both actual suns or
   clearly labelled theatrical cues, convincing dark layered clouds below,
   no severe crop/overlap and readable light on bodies. Compare CPU/GPU
   only within each path's stated limitations. Preserve the exact job,
   binary, source scene, distinct unmodified PFM and developed PNG.

## Sources and attribution

- NASA's [Polar Mesospheric Clouds](https://science.nasa.gov/earth/earth-observatory/polar-mesospheric-clouds-south-pacific-ocean-80217/)
  and [Nocturnal Wonders](https://science.nasa.gov/resource/nocturnal-wonders/)
  describe high Earth ice clouds catching sunlight after it has set for a
  lower observer. The Azure shot borrows contrast and light direction as
  art; it is not an Earth cloud model or a physical survival claim.
- [Guerrilla's Nubis 2017 presentation](https://www.advances.realtimerendering.com/s2017/Nubis%20-%20Authoring%20Realtime%20Volumetric%20Cloudscapes%20with%20the%20Decima%20Engine%20-%20Final%20.pdf)
  describes multiscale noise erosion of cloud shapes; the proposed optional
  Slayer detail is an independently invented interpretation, not Nubis.
  The presentation is pinned as `schneider-vos-2017-nubis` in `sources.yaml`;
  use Slayer's source adapter for reproducible retrieval. Prefer the same
  approach for any new source once verified and pinned.
- Existing system and anatomy provenance lives beside the set in
  `sets/caprica-binary/DIRECTION.md`, `AZURE-TWILIGHT.md`,
  `doc/model/bioform/hatzegopteryx/`, and the repository's source manifests.
  Cite the relevant implementation and source; do not convert an artistic
  choice or an unverified render into a scientific claim.
