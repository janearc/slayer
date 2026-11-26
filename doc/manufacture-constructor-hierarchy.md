# Manufacture constructor hierarchy

This is a map of the **implemented** constructor paths on
`parts-compositions-taxonomy`. The migration is still in progress;
the map is a snapshot for review, not a claim that the tree is finished. It
covers the current biological families in detail and the main entry points
for other manufactured things.

Indentation groups related constructors and data. An arrow means a real call
or dispatch, not inheritance. A species name under a family is usually a row
in a checked profile book, not its own construction algorithm. The `New...`
functions below return either a reusable `Kind` recipe or constructed parts,
as noted; `Kind` recipes are realized later with a seed and detail level.

## Scene and factory entry points

```text
KindNamed(name)                         registered non-profile kind
ReadPlantBook(family, file, bytes)
  -> PlantBook.New(options)             checked plant Kind
ReadAnimalBook(family, file, bytes)
  -> AnimalBook.New(options)            checked animal Kind
ReadKelpProfiles(file, bytes)
  -> KelpProfiles.NewKelp(options)      checked alga Kind

Kind + Sow(seed)
  -> Make / MakeWithComplexity          manufactured good
  -> MakeAtTime                         time-dependent good, where supported
  -> good.Things(placement)             scene things
```

`EmbeddedPlantBook`, `EmbeddedAnimalBook` and `EmbeddedAlgaBook` supply copies
of the in-repository defaults. File-backed recipes can instead supply their
own checked books. See [plant recipes](plant-profile-recipes.md),
[animal recipes](animal-profile-recipes.md), and
[alga recipes](alga-profile-recipes.md).

## Plants and algae

```text
PlantBook.New
├── seasonal -> PlantProfiles.NewPlant(species) -> seasonal.Book.Get
│   ├── Kind realization -> seasonal.Profile.Plan -> branching.Expand
│   ├── optional WithSeason -> summer / autumn / winter foliage
│   ├── NewValleyOak, NewAlder, NewPoplar, NewManzanita,
│   │   NewChaparralBrush, NewJoshuaTree
│   ├── NewKandeliaObovata, NewAegicerasCorniculatum,
│   │   NewAvicenniaMarina, NewBruguieraGymnorhiza,
│   │   NewSonneratiaApetala
│   └── NewPandanusTectorius, NewScaevolaTaccada,
│       NewMetrosiderosPolymorpha, NewBarringtoniaAsiatica,
│       NewQuercusIlex, NewEricaArborea, NewSalixAlba
│       -> shared seasonal plant planning, branching, bark and foliage
├── conifer -> ConiferProfiles.NewConifer(species)
│   ├── conifers.Book.Select -> Kind realization
│   │   -> conifers.Architecture.Grown -> branching growth
│   ├── NewSpruce, NewNorwaySpruce
│   ├── NewCoastRedwood, NewGiantSequoia
│   └── NewConiferShoot(species) -> conifers.PlanShoot
│       ├── NewCoastRedwoodShoot
│       └── NewGiantSequoiaShoot
├── fern -> FernProfiles.NewFern(species)
│   └── NewGenericFern -> ferns.Growth -> branching.NewGrowth
└── cactus -> CactusProfiles.NewCactus(species) -> cacti.Book.New
    └── NewSaguaro -> cacti.Recipe.Columns / Surface -> cacti.Stem / spines

NewGiantKelp
└── KelpProfiles.NewKelp(options) -> kelp.Profiles.New -> kelp specimen / forest
```

Seasonal plants, conifers, ferns and cacti have separate profile books and
construction paths. The seasonal grouping is a shared growth implementation,
not a claim that all its species form one botanical family. The seasonal family book owns its checked profiles, growth plan and foliage
cell cache. The fern profile controls its crown, stalk and leaflet proportions. The cactus family book
and recipe own the species profile, seeded columns and stem dimensions; the
root kind adapts them to manufacture. Kelp is an alga; its water condition
and forest selection are recipe options.

The general-purpose [`branching`](../manufacture/branching) package supplies
`NewGrowth`, `NewLimb`, `NewSleeve`, `NewLeaf`, `NewRing` and `NewWhorls`,
as well as the breadth-first expansion used by seasonal growth. It is not a
plant-only taxonomy. Conifer shoots use the family’s indexed depth-first planner;
kelp uses a different iterative growth path.

## Animals

```text
Coral(CoralSpec) -> coral.Spec.Grow -> coral.Skin / Normals
                   -> a seeded branching cnidarian colony

AnimalBook.New
├── squid -> SquidProfiles.NewSquidKind(species, phase)
│   └── squid.Book.Get -> Kind realization -> squid.Recipe.Build
│       -> squid.NewSquid -> molluscs.NewMollusc
│   Direct: NewHumboldtSquid -> NewSquid("humboldt-squid")
│       -> SquidProfiles.NewSquid -> squid.Book.Get -> squid.Recipe.Build
│
├── fish -> FishProfiles.NewFish(species, options)
│   └── fishes.FishProfiles.NewFish -> fishes.Recipe.Build
│       ├── Pacific bluefin tuna, California yellowtail, California barracuda
│       ├── Pacific sardine, convict tang, dusky grouper
│       └── Mediterranean chromis, European perch, flathead grey mullet
│           -> single fish / glide / bones / supported schools
│
├── bird -> BirdProfiles.NewBird(species, stance)
│   └── birds.Book.New -> birds.Specimen.Build
│       ├── black-faced spoonbill, little egret, grey heron, Pacific reef heron
│       └── yellow-legged gull, white tern, great frigatebird,
│           white-tailed tropicbird
│           -> supported standing / gliding / bones / outline forms
│
├── theropod -> TheropodProfiles.NewTheropod(species, seconds)
│   └── NewAllosaurus -> theropods.TheropodProfiles.NewTheropod
│       -> Allosaurus walking study
│
├── sauropod -> SauropodProfiles.NewSauropod(species)
│   ├── sauropods.Book.Profile -> Kind realization -> sauropods.Build
│   └── NewAlamosaurus, NewDiplodocus, NewApatosaurus, NewBrachiosaurus
│
├── pinniped -> PinnipedProfiles.NewPinniped(species, pose)
│   ├── pinnipeds.Book.Get -> checked species profile
│   ├── NewHarborSeal        -> pinnipeds.NewLoft -> Loft.Build; hauled or swimming
│   ├── NewHawaiianMonkSeal  -> pinnipeds.NewLoft -> Loft.Build; hauled
│   ├── NewWalrus            -> pinnipeds.NewWalrus -> Walrus.Build; lying, hauled-out or swimming
│   ├── NewPinnipedRepresentation -> pinnipeds.Bones or Walrus.Build for bones/volumes
│   └── NewPinnipedStudy     -> pinnipeds.Detailed or pinnipeds.Bay
│
├── whale -> WhaleProfiles.NewWhale(species, pose, representation)
│   └── NewOrca -> whales.Profile.Swim / SpyHop / Bones
│
├── snake -> SnakeProfiles.NewSnake(species)
│   ├── snakes.Book.Profile -> Kind realization -> snakes.Build
│   ├── NewPalaeophisColossaeus, NewTitanoboaCerrejonensis, NewMadtsoiaBai
│   ├── NewEunectesMurinus, NewBoaConstrictorOccidentalis
│   ├── NewMalayopythonReticulatus, NewPythonBivittatus, NewPythonSebae
│   └── NewBitisGabonica
│
└── reptile -> ReptileProfiles.NewReptile(species, study)
    ├── NewPliosaur
    │   ├── pliosaurs.NewComposite
    │   └── pliosaurs.NewContinuous
    ├── NewPurussaurus -> crocodylians.NewCrocodylian
    └── NewHatzegopteryx
        ├── hatzegopteryx.NewComposite
        ├── hatzegopteryx.NewCage
        └── hatzegopteryx.NewDensity
```

Snake profiles select one static specimen from a shared body and head builder.
Their recorded dimensions and colours are illustrative, and fossil soft tissues
are reconstructions. They do not yet supply a snake skeleton or animated pose.

The root squid convenience path is an actual chain:
`NewHumboldtSquid` -> `NewSquid` -> `SquidProfiles.NewSquid` ->
`squid.Book.Get` -> `squid.Recipe.Build` -> `squid.NewSquid` ->
`molluscs.NewMollusc`. The squid family book owns its checked profile and
private cached rest surface. `NewMollusc` provides the posed
soft body and attachment points; it is not a complete universal mollusc.
The squid proportions, material and motion are illustrative.

The fish builder covers the existing bony, ray-finned fish, not sharks,
rays, jawless fish or every fish lineage. Birds are living dinosaurs within
theropods; Allosaurus is a non-avian theropod. Hatzegopteryx is a pterosaur,
Purussaurus a crocodilian relative, and pliosaurs are marine reptiles; none
is being classified as a fish. The reptile studies use separate scaffolds.

The pinniped book carries skeleton counts, poses, bounded seal/walrus
outline variation, and the lofted seals’ pelt and anatomical-source data.
The pinniped family owns the seal and walrus rigs. The drawn seal studies
share a fixed outline distinct from the lofted seal profiles. Seal and walrus
have distinct skins and anatomy; the walrus retains its tusks and limbs. A
retained walrus rig can cache a ground surface and a swimming surface, each
retaining about 17.5 MB at the default grid once used. These surfaces go away
with the rig; output paint retains only the frame and ray data it needs, so
keeping output does not retain the cache.

The whale book owns the orca’s trunk stations, timing, satin and palette.
Its skeleton and marking placement remain specific to the orca scaffold;
it does not yet describe all cetaceans. `mammals.AxialHold` is a shared
bone-binding operation; there is no implemented universal `NewVertebrate`
or `NewMammal` constructor.

Sauropod profiles select one static reconstruction through a shared body and
limb scaffold. Proportions combine fossil evidence with inferred soft tissue;
pose, colour and missing skeletal parts are illustrative, not measurements.
The species do not yet supply animated poses.

## Other manufactured things

```text
Composition(cities.Composition) -> validated parts and connections -> Kind
Building(BuildingSpec)          -> building Kind
SpacecraftFrom(SpacecraftSpec)   -> spacecraft Kind
cities.NewTower / NewBridge / NewBareBridge / NewHubBridge
cities.NewVillage / NewSuperblock / NewLashings
                                -> city designs and assemblies
KindNamed                       -> registered makers such as stone, fire,
                                   vehicles, spacecraft, buildings and bridges
```

These are existing construction paths, not a claim that every structure or
spacecraft already shares a single parent constructor. The earlier
[proposal and historical inventory](constructor-hierarchy-review.md) records
additional desired boundaries and the reasons they still need work.
