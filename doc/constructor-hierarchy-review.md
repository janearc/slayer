# Constructor hierarchy for review

The proposal and historical inventory were written against Slayer checkout
`parts-compositions-taxonomy` at `fcaae65f`. The tree below is a
proposal, not a claim that all of its constructors or directories exist.
Names beginning with `New` in the proposed tree are working names. A common
boundary must do real shared work and earn its cost by reducing code.

## Current implementation and remaining work

Checked species books and constructors exist for seasonal plants, conifers,
cacti, ferns, kelp, ray-finned fish, birds, pinnipeds, whales, theropods,
reptile studies and squid. Their authored scenes and catalogue cards use
profile recipes. The former plant, alga and animal factory-kind aliases are
retired. Catalogue card identifiers and the private scavenged-cluster
population may retain historical labels; they select profile recipes and
are not factory kinds.

The squid path is `NewHumboldtSquid` -> the checked squid profile ->
`squid.NewSquid` -> `molluscs.NewMollusc`. Squid profiles supply mantle size,
five anatomy proportions, coat and mottle. The mantle, appendages and
illustrative pulse share one scaffold; the profiles are not validated
representations of every squid species.

Pinniped profiles select skeleton counts, posing and supported seal or walrus
outlines. The existing walrus head, limbs, tusks and hide, and the separate
drawn-seal studies, retain their own anatomy. Orca profiles supply body
stations, swim-cycle length, skin sheen and marking colours; fins, flukes,
face, marking placement and skeletal details remain tied to the orca study. `NewFish` currently supports
ray-finned forms; `NewBird` uses the existing bird rig. The theropod book
supports the Allosaurus study, and the reptile book selects the distinct
pliosaur, crocodylian and pterosaur studies. These are not universal bodies
for their larger clades.

Fern profiles now supply crown, stalk and leaflet proportions; their family
recipe and the actual depth-first limb growth engine live in `plants/ferns`
and `branching`. Seasonal plants use the separate bounded breadth-first
branching expansion, and conifer shoots retain their own indexed depth-first
planner. Their order and seed rules differ. Kelp is under algae, not plants.

The bird profile reader, skeleton, feathers, poses and geometry now live in
the birds family package, with a root adapter for factory kinds. The other
profile books have taxonomic paths, but much animal geometry and several plant
constructors and readers remain in the root `manufacture` package. The shared
vertebrate, mammal and other ancestor constructors in the proposal below are not yet implemented; they should be added only where real
common anatomy can replace code. This migration is not complete. Embedded
model books and shared construction policies still live in Slayer; checked
file-based readers and snapshot keys prepare for, but do not create, a
separate models repository. See [plant recipes](plant-profile-recipes.md),
[animal recipes](animal-profile-recipes.md), and
[alga recipes](alga-profile-recipes.md).

## The intended shape

A species constructor supplies its own data to a family constructor, and a
family supplies common anatomy to a broader constructor where there is a real
common operation. For example:

```text
NewHumboldtSquid(phase)
  -> NewSquid("humboldt-squid", phase)
       -> SquidProfiles.NewSquid(species, phase)
            -> shared squid construction
                 -> NewMollusc(soft body, pose, material)
```

The arrow means an actual call, not Go inheritance. Common constructors take
their measures and choices as data. They do not need a new Go file or builder
for every species. A parent must do real shared work; a function that merely
forwards to a child adds footprint without consolidating anything.

## Proposed biological hierarchy

```text
animals/
  molluscs/
    NewMollusc                 soft body, pose, material, attachment points
    squid/
      NewSquid                 mantle, fins, arms, tentacles, eyes, motion
      NewHumboldtSquid         Humboldt dimensions, appearance and behavior
  vertebrates/
    NewVertebrate              shared skeletal construction and posing
    fishes/
      NewFish                  shared fish construction where anatomy permits
      jawless/                 living and extinct jawless fish
      early-jawed/             extinct placoderms and other early forms
      cartilaginous/           sharks, rays, chimaeras and extinct relatives
      bony/
        ray-finned/            tuna, sardine and other current fish profiles
        lobe-finned/           coelacanths and lungfish
    archosaurs/
      dinosaurs/
        NewDinosaur            shared dinosaur construction where justified
        saurischians/
          theropods/
            non-avian/         Allosaurus and other non-avian theropods
            avian/
              NewBird          skeleton, wings, plumage and stance
              species          heron, egret, gull and other bird profiles
          sauropodomorphs/     sauropods and their relatives
        ornithischians/        armored, horned, dome-headed and other lineages
      pterosaurs/              Hatzegopteryx; a separate archosaur branch
      crocodilians/            Purussaurus and other crocodilian relatives
    mammals/
      NewMammal                only the skeletal and surface work really shared
      whales/
        NewWhale               common cetacean body and rig
        NewOrca                orca profile and poses
      pinnipeds/
        NewPinniped            common pinniped anatomy and posing
        seals/
          NewSeal              seal body and coat
          NewHarborSeal        harbor seal profile and poses
          NewHawaiianMonkSeal  monk seal profile and poses
        walruses/
          NewWalrus            walrus profile, tusks and poses
```

This tree must be able to hold living and extinct dinosaur and fish lineages,
not just the kinds already present. Extinct forms occur throughout these
branches; an extinct flag is profile data, not a single taxonomic family.
Birds are living avian dinosaurs within theropods. Allosaurus is a non-avian theropod. Hatzegopteryx is a pterosaur,
not a dinosaur; Purussaurus is a crocodylian relative, not a dinosaur. Pliosaurs
and mosasaurs are marine reptiles on other branches, not dinosaurs or fish.

`fishes` is a practical model grouping, not a single exclusive clade: in
evolutionary classification tetrapods are also within the lobe-finned lineage.
Jawless, cartilaginous and bony fish have distinct skeletons and anatomy. A
shared `NewFish` must accommodate that variety or delegate to different
builders; the current ray-finned builder cannot simply be assumed to fit all.
The named ancestors are proposed: a taxonomic folder does not automatically
require an ancestor constructor with no actual shared work.

Species-specific anatomy must stay expressible: a walrus has tusks, a
different skeleton and different poses from a seal. `NewPinniped` must accept
those differences rather than force the walrus through a seal's shape.
`NewVertebrate` and `NewMammal` are proposed boundaries to prove from real
shared behavior, not instructions to put unrelated animals into one rig.

## Historical dinosaur and fish inventory at `fcaae65f`

This inventory records the reviewed commit, not the current factory vocabulary
or a limit on the future model repository. Species should normally be profiles
with a small named constructor; adding a species should not require a separate
duplicated construction system. The kind names below were present at that
commit; the migrated species labels are no longer public kind aliases.

### Fish: nine profiles at the reviewed commit

At `fcaae65f`, all nine rows in `manufacture/fish.yaml` used the same bony,
ray-finned fish maker: `fishRows` -> `fishRow` -> `planFish` -> `buildFish`.
Each had a species kind and `-glide` and `-bones` kinds. Schools were present
only where listed; their numbers were not new species.

| Profile / historical base kind | Historical school kinds |
|---|---|
| `pacific-bluefin-tuna` | none |
| `california-yellowtail` | none |
| `california-barracuda` | none |
| `pacific-sardine` | `pacific-sardine-school-500`, `pacific-sardine-school-5000` |
| `convict-tang` | none |
| `dusky-grouper` | none |
| `mediterranean-chromis` | `mediterranean-chromis-school-60` |
| `european-perch` | none |
| `flathead-grey-mullet` | `flathead-grey-mullet-school-120` |

No shark, ray, chimaera, jawless fish, coelacanth or lungfish profile was found
in this maker. Their branches above are future coverage, not existing kinds.
The common builder's fin rays, bony skeleton and caudal assumptions must not
be imposed on those other forms.

### Dinosaurs: non-avian and living birds at the reviewed commit

- **Allosaurus fragilis:** `AllosaurusStudy(seconds, at)` in
  `manufacture/allosaurus_bioform.go` is a walking non-avian theropod study.
  It is reached at phase zero through `gargoyle_studies.go` and the
  `gargoyle-allosaurus-granite` and `gargoyle-allosaurus-concrete` kinds.
  At that commit it was not a data-driven `NewDinosaur` / `NewTheropod` constructor.
- **Birds:** the eight profiles in `manufacture/birds.yaml` are
  `black-faced-spoonbill`, `little-egret`, `grey-heron`, `pacific-reef-heron`,
  `yellow-legged-gull`, `white-tern`, `great-frigatebird` and
  `white-tailed-tropicbird`. Each then had four kinds of the form
  `bird-<species>-standing`, `-gliding`, `-bones` and `-outline`. The shared
  path at that commit was `birdKinds` -> `birdKind` -> `birdThings` ->
  `newBirdBody`; it did not call a generic dinosaur or vertebrate constructor.
- No other non-avian dinosaur species constructor was found by name in the
  checked-out code. Sauropodomorph and ornithischian branches are proposed,
  not implemented. They need genuinely different size, support, posture,
  ornament and motion profiles; an Allosaurus-shaped default is not sufficient.

Other prehistoric models must stay in scope without being mislabeled dinosaurs:
the Hatzegopteryx composite study and gargoyle are pterosaurs;
`PliosaurStudy`/`PliosaurFresh5` were pliosaur marine reptiles;
`PurussaurusFresh5` was a crocodylian relative. The latter two constructors
are now named `PliosaurContinuousSkin` and `PurussaurusStudy`. The checkout also contains
`study/mosasaurus-four-pass` plans and reviews for *Mosasaurus hoffmannii*;
no Mosasaurus-named Go or YAML constructor was found in this checkout.
These are separate branches, not fish merely because some swam.

## Proposed plant and alga hierarchy

```text
plants/
  NewPlant                     shared growth/placement only where it fits
  cacti/
    NewCactus                  stems, branching and spines from a profile
    NewSaguaro                 saguaro profile
  conifers/
    NewConifer                 trunk, branches and foliage from a profile
    NewSpruce                  spruce profile
    NewNorwaySpruce            Norway spruce profile
    redwoods/
      NewCoastRedwood          coast redwood profile
      NewGiantSequoia          giant sequoia profile
  broadleaf-and-other-plants/
    shared profile builder     oaks, mangroves, shrubs and other existing rows
  ferns/
    NewFern                    frond growth and profile
algae/
  kelp/
    NewKelp                    shared kelp growth and forest placement
```

The `broadleaf-and-other-plants` label is a placeholder, not a botanical
clade. Joshua trees and pandanus, for example, should not be classified as
broadleaf trees just to match an existing implementation. Taxonomic directories
and a reusable growth engine need not have identical boundaries. Kelp is an
alga, not a plant. A shared `NewPlant` should not force all plants into a
single branching or foliage model.

## Historical construction paths at `fcaae65f`

At the reviewed commit, the usual route for a registered kind was `KindNamed` -> registry `table` ->
the selected maker -> `Make` or `MakeWithComplexity`. This is dispatch: it
does not mean every kind calls one biological parent constructor.

| Family | Shared path at `fcaae65f` | Gap at that commit |
|---|---|---|
| Squid | `NewHumboldtSquid` -> `NewSquid` -> `NewMollusc`; the gargoyle study calls the Humboldt entry point. | `NewSquid` scales the Humboldt-authored field, pose and look; it is not yet a general species-profile squid builder. `NewMollusc` supplies a posed mesh and anchors, not complete mollusc anatomy. |
| Fish | `fishRows` -> `fishRow` -> `planFish` -> `buildFish`; species and school choices come from `fish.yaml`. | Shared builder exists, but no exported taxonomic `NewFish`/`NewVertebrate` chain. |
| Birds | `birdKinds` -> `birdKind` -> `birdThings` -> `newBirdBody`; species data comes from `birds.yaml`. | Shared bird builder exists; no `NewVertebrate` parent. |
| Seals | The lofted harbor and Hawaiian monk seals share `sealLoftedKind` and `sealForm`. Other harbor and bay seal variants use separate rig paths. | Their variants are not all under one `NewSeal`; no shared pinniped constructor with walrus. |
| Walrus | `walrusFresh5Kind` selects the maintained body, skeleton, volume or pose path. | Separate skeleton and skin from seals; no `NewPinniped`. |
| Orca | `orcaClean05Kind` calls the maintained orca builder; separate variants provide bones and spy-hop. | No `NewWhale` or `NewMammal`. |
| Profile plants | `vegetationRows` -> `seasonalPlant` -> profile plan, trunk and foliage; 18 kinds share profile data. | This is a growth-engine boundary across several plant groups, not a botanical family. |
| Conifers | Spruce, Norway spruce, coast redwood and giant sequoia share a conifer scaffold, with different species settings and foliage. | No exported `NewConifer` or universal plant constructor. |
| Saguaro | `saguaro` -> its body, branches and spines; stem geometry is in `plants/cacti`. | The cactus package currently holds stem geometry, not a complete `NewCactus` builder. |
| Fern and kelp | Fern has its own growth path; kelp kinds share `kelpRow` and `kelpPlant.grow`. | Neither is under a universal plant constructor; kelp should remain outside plants. |

The main entry points are in `manufacture/kinds.go`, `offered.go` and
`complexity.go`. Animal paths are in `humboldt_squid.go`, `mollusc.go`,
`fish.go`, `bird_kind.go`, `harbor_seal_lofted.go`,
`walrus_fresh5_kind.go` and `orca.go`. Plant paths are in
`vegetation.go`, `physical_spruce.go`, `redwoods.go`, `saguaro.go`,
`plants/cacti/stem.go` and `kelp.go`. These are the paths at the reviewed
commit; `redwoods.go` has since been removed in favor of the checked conifer
profile book and shared constructor.

## Other manufactured things

The same principle applies outside biology, but their taxonomy is by what a
thing is and how it is assembled, rather than by a biological clade.

```text
structures/
  parts                       body, leg, storeys, windows, deck, stays, road
  composition                 parts and connections read from YAML
  buildings/                  tower, condominium, house
  bridges/                    deck, supports, approaches and inhabited parts
spacecraft/
  shared components           bus, power, antenna, instruments, structure
  observatories/              Spitzer, Hubble, Webb, PRIMA, Roman
  probes/                     Voyager, Cassini, SOHO, Mariner, New Horizons
vehicles/
  road/                       trucks and other wheeled vehicles
  water/                      boats and submarines, with distinct hull needs
```

Today the new composition builder, the older bridge/tower makers and
`Building` are distinct paths. The composition uses reusable city geometry;
it is not yet a parent of the older kinds. `Spacecraft` shares a builder for
several observatories; Voyager 1 and 2 share another; several probes still have
separate makers. Boats share data-driven construction and trucks share a
maker, but a boat and the Alfa submarine do not share a general vessel
constructor. `Building` and `SpacecraftFrom` already share assembly primitives
such as boxes, drums, rods and dishes. A common part is useful when it removes
repeated construction, not merely because two things have similar names.

## Review decisions

- Species should almost always be profile data with a small named constructor.
- Allow the shared pinniped skeleton and pose to express differences such as
  species-specific skin and tusks.
- Do not retain old kind names as aliases: change names and update references.
  The intended destination for the YAML model data is a separate models
  repository, and this work prepares that boundary.
- Start with plants, then vertebrates.
- Recursive branching should be a general constructor, usable by plants and
  by other forms such as rubble and mountains. Species profiles supply their
  own branching rules and dimensions; the shared mechanism is not a plant.
  Keep this general mechanism separate from the taxonomic profiles that use it.

## Original questions

1. Is the proposed biological tree the level of taxonomy you want in the
   directories and constructor names? Should species usually be profile data
   with a small named convenience constructor only where it adds meaning?
2. For seals and walrus, should the first shared boundary be a data-driven
   pinniped skeleton and pose, with skin and tusks remaining species-specific?
3. Should growth engines such as `seasonalPlant` live separately from the
   taxonomic directories, with species profiles pointing to them?
4. Should old set kind names continue as aliases during consolidation, even
   where internal constructors and files are renamed?
5. Which family should be consolidated first after review: pinnipeds,
   conifers/cacti, or the rest of the squid/mollusc boundary?

The work recorded in `structure-compositions-progress.md` is still in
progress. This map does not certify identical geometry, full test coverage or
completion of the wider taxonomy migration.
