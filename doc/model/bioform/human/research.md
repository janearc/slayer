# Cinematic bioforms: reference and direction

## Scope

The goal is a native, procedural, anatomically complete adult human for
artistic and cinematic stills and films. A figure that reads at scene scale
is one use, not the acceptance criterion. The shared construction, rigging,
surface and grooming machinery should also serve other bioforms. A human
constructor is an application of those tools, not a template that dictates
whale, bird, fish or dinosaur anatomy.

The existing portrait study uses an imported body. It is useful research,
but is not a native Slayer human. The native torso study is partial; its
shape, transitions and pose response have not been established as
anatomically plausible. Neither demonstrates a finished human constructor.

## Character-model precedents

Baldur's Gate 3 and Cyberpunk 2077 are useful artistic precedents for
character detail, customization and use in narrative scenes. Ordinary adult
anatomy belongs in a complete model and does not make the modeling task
sexual. These games also include sexual content in other contexts; that is
not a characterization of this project's purpose.

- Larian describes the identity and appearance choices in Baldur's Gate 3
  and their place in the player's story. An interview with character artist
  Alena Dubrovina describes its design and reports both praise and criticism
  of constraints on body and face customization.
  [Larian](https://blog.playstation.com/2023/09/14/how-your-choices-in-baldurs-gate-3s-character-creator-carry-through-your-entire-playthrough/),
  [Game Developer](https://www.gamedeveloper.com/art/building-an-inclusive-character-creator-for-the-fantasy-world-of-baldur-s-gate-3)
- GameSpot's review praises the freedom of Baldur's Gate 3, including
  character creation. It is a review of the game, not a measurement of
  anatomical accuracy.
  [GameSpot](https://www.gamespot.com/reviews/baldurs-gate-3-review-let-freedom-reign/1900-6418131/)
- The ESRB describes Cyberpunk 2077's adult body customization. A
  pre-release PlayStation hands-on praised its customization; a later
  Tom's Guide review criticized it as shallow. Reception was not uniform,
  and neither establishes anatomical fidelity.
  [ESRB](https://www.esrb.org/ratings/36949/cyberpunk-2077/),
  [PlayStation](https://blog.playstation.com/2020/06/25/cyberpunk-2077-hands-on-night-city-is-a-playground-of-customization-and-player-choice/),
  [Tom's Guide](https://www.tomsguide.com/reviews/cyberpunk-2077)

These are production and artistic references, not licensed geometry or a
scientific validation dataset. Their reviews do not establish head-to-toe
anatomical accuracy. Assess Slayer against anatomical references and actual
renders, not a claim inferred from either game's reception.

## Technical reference points

- [SMPL](https://files.is.tue.mpg.de/black/papers/SMPL2015.pdf) describes
  shape- and pose-dependent deformation and makes the limitations of plain
  blended skeletal transforms visible. It is a research reference, not
  permission to substitute an imported body for Slayer geometry.
- [Efficient Rendering of Human Skin](https://research.nvidia.com/publication/2007-06_efficient-rendering-human-skin)
  explains a diffusion approximation. Surface diffusion is not by itself
  a measured, layered, through-thickness skin transport model.
- [A Practical and Controllable Hair and Fur Model for Production Path Tracing](https://media.disneyanimation.com/uploads/production/publication_asset/147/asset/siggraph2015Fur.pdf)
  is a rendering reference for fiber scattering. Geometric roots alone do
  not provide that light transport or hair dynamics.

## Evidence to gather when requested

Inspect adult figures in front, side, back and three-quarter views, with
consistent scale and neutral, raking and back lighting. Examine anatomical
landmarks, surface continuity and silhouettes separately from material.
Include variations in adult proportions and skin tones, plus raised arms,
twists and bending. Show deforming surfaces and attachments through motion,
including planted feet where applicable. Inspect hair roots, hairline,
parting, coverage and how the groom follows deformation. Inspect skin at
both close and scene scales for pigmentation, roughness and translucency.

These are inspection directions, not measured pass thresholds. No test,
review or render in this document asserts that the current work meets them.
Document actual approximations and unsupported features with each result.
The project's renders belong in the shared gallery when produced.

## Measurement and transport references

The [NHANES 2015–2018 anthropometry report](https://stacks.cdc.gov/view/cdc/100478)
provides distributions of measured body lengths and circumferences in a US
sample. It is useful for broad variation, not a dense map of the human
surface or a global population model. Marginal percentiles should not be
combined as if they described a measured individual; correlated measurements
need appropriate subject-level data.

[Jensen et al.](https://graphics.stanford.edu/~srm/publications/SG01-subsurf-abstract.html)
describe single scattering and a diffusion approximation for multiple
scattering, including comparison to measured materials. A colour tint alone
is not that transport. [Physically Based Rendering's hair chapter](https://www.pbr-book.org/4ed/Reflection_Models/Scattering_from_Hair)
describes a pigmented fiber core and anisotropic scattering; geometry with
ordinary surface shading does not implement that model. Different species
need their own anatomical and material reference data, even where they share
interfaces and algorithms.
