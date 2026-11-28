# Scene constructors and their authoring surfaces

This page connects the scene entry points to the geometry and world-building
APIs they use. It is an index, not a promise that every Go value has YAML
syntax. A set file and a named scene job are different authoring surfaces:
the set reader accepts its documented fields, while a named scene accepts
only the parameters its registered builder declares.

## Entry points

| Entry | What it constructs | Source and fuller guide |
|---|---|---|
| `slayer draw SET` or `slayer draw JOB` with `scene.set` | reads and stages the set, then draws it with the chosen renderer. | [sets](sets.md), [jobs](../job/README.md), [`set.Read`](../set/keys.go) |
| `slayer assess FILE` | runs the preparation checks and reports what drawing would write. | [`assess`](../door/assess.go) |
| `slayer draw JOB` with `scene.named` | Looks up a scene registered in the running Slayer process, validates its `with` map, then builds its world or map output. | [`scene/named`](../scene/named/README.md), [`job.Scene`](../job/scene.go), [`door`](../door/README.md) |
| `slayer list scenes` | Lists the runtime registry; it is the authority for names present in that binary. | [`scene/named.All`](../scene/named/named.go), [`door`](../door/README.md) |

The built-in traced registry currently contains `swarm`, `zappy`, `orbital`,
`ring`, `moonrise`, and `magnetar`.

Their individual constructors and parameters are in
[`scene/named`](../scene/named/README.md), with implementations in
[`swarm.go`](../scene/named/swarm.go), [`zappy.go`](../scene/named/zappy.go),
[`orbital.go`](../scene/named/orbital.go), [`ring.go`](../scene/named/ring.go),
[`moonrise.go`](../scene/named/moonrise.go), and
[`magnetar.go`](../scene/named/magnetar.go).

`weather` is the separate map-producing scene, built by
[`weather.go`](../scene/named/weather.go) and described in the named-scene
guide; it is not a `thing.World` constructor.

The Slayer command registers the application scenes `heard` and `heard-film`
at startup through [`cmd/slayer/main.go`](../cmd/slayer/main.go),
[`slayer_scene.go`](../improv/searching/slayer_scene.go), and
[`slayer_film.go`](../improv/searching/slayer_film.go). `heard` takes a checked
stage plan; its stage kinds and scene-specific geometry live under
[`improv/searching`](../improv/searching/README.md) and `stage_*.go`.

`heard-film` takes a screenplay path and composes those stages with other
registered named scenes. These are Go-registered scenes with validated job
parameters, not set YAML recipes. The general registration hook is
[`named.RegisterAuthored`](../scene/named/authored.go); applications must
register builders before the job can use them.

## Set YAML: recipes and direct world pieces

The complete current set reader is the authority for accepted YAML. Its
top-level scene vocabulary is declared in [`set/scene.go`](../set/scene.go),
and its world-piece keys are declared in
[`set/pieces.go`](../set/pieces.go). The field-by-field guide is
[`sets.md`](sets.md); schemas describe document structure, while the reader
and stage determine whether a validated feature is actually drawable.

| YAML surface | What it can build | Source |
|---|---|---|
| `scene.recipes`, `scene.set.things`, `scene.set.stands` | reusable manufactory recipes and placed catalogue kinds. | [`set/recipes.go`](../set/recipes.go), [`set/world.go`](../set/world.go), [constructor reference](constructor-reference.md) |
| `scene.set.bounded_box`, `scene.set.assets`, `scene.set.texts` | bounds, declared assets and scene text. | [`set/scene.go`](../set/scene.go), [`set/text.go`](../set/text.go) |
| `scene.world.materials`, `boxes`, `spheres` | Named materials plus the built-in box and sphere pieces; references are checked by the reader. These are not arbitrary `thing.Thing` literals. | [`set/pieces.go`](../set/pieces.go), [`set/shapes.go`](../set/shapes.go) |
| `scene.world.imports` | Checked external geometry placements with explicit material bindings, provenance, and license; mesh faces are not authored inline in YAML. | [`set/imports.go`](../set/imports.go), [`oakland`](../oakland/api.go) |
| `scene.world.lamps`, `spots`, `floods`, `suns`, `sky` | The set format's light emitters and sky descriptions. | [`set/pieces.go`](../set/pieces.go), [`set/spots.go`](../set/spots.go) |
| `scene.world.wisps`, `fizzes`, `vapor-cavities` | Set-level gas, rising fizz, and bounded five-tip propeller-vapor density fields. The propeller form requires explicit visual onset and remains a visual approximation. | [`set/shapes.go`](../set/shapes.go), [`scene/medium/propeller_cavitation.go`](../scene/medium/propeller_cavitation.go), [`medium`](../scene/medium/medium.go) |
| `scene.world.bodies`, `motions` | Template-based celestial bodies and the set reader's named keyframe motions. | [`set/pieces.go`](../set/pieces.go), [`set/motion.go`](../set/motion.go), [`set/space_things.go`](../set/space_things.go) |
| `scene.terrain`, `air`, `atmosphere`, `weather`, `ecosystems`, `fire-fronts`, `fuel-beds`, `lasers`, and `planet` | Higher-level terrain, climate, ecology, fire, optics, and planetary adapters. These compose geometry and media; they are not generic shape constructors. | [`set/scene.go`](../set/scene.go), [`set/atmosphere.go`](../set/atmosphere.go), [`set/ecosystems.go`](../set/ecosystems.go), [`set/fire_front.go`](../set/fire_front.go), [`set/fuel_bed.go`](../set/fuel_bed.go), [`set/lasers.go`](../set/lasers.go), [`set/planet.go`](../set/planet.go) |

The remaining scene-level keys—film, cameras, geographic location and time,
light choice, moon/sun placement, fantasy overrides, and HUD—set viewing,
lighting, or presentation. They do not add a general shape constructor; their
details are in [`sets.md`](sets.md) and the readers in [`set/scene.go`](../set/scene.go),
[`set/world.go`](../set/world.go),
[`set/fantasy.go`](../set/fantasy.go), and [`set/hud.go`](../set/hud.go).

Set YAML has no inline constructor for `shape.Tube`, arbitrary triangles,
`shape.MeshGroup`, or general `thing.Thing`/`thing.Volume` literals. Use a
registered recipe, the checked import path, or a Go-authored scene as
appropriate.

see [sets in space](sets.md#a-scene-in-space) for the supported placements
and [`session/afloat.go`](../session/afloat.go) for the stage. the reader's
reserved features are checked by [`session/undrawn.go`](../session/undrawn.go).

## Direct Go surface geometry

These are values or constructors for a scene builder. They are not exposed as
general inline set YAML shapes.

| Go API | Geometry and boundary | Source |
|---|---|---|
| `shape.Sphere`, `shape.Plane` | Analytic ball and unbounded plane values. | [`scene/shape/shape.go`](../scene/shape/shape.go) |
| `shape.Tube` | Marchable hollow cylindrical shell value. | [`scene/shape/tube.go`](../scene/shape/tube.go) |
| `mesh.New` | Validated immutable indexed triangle mesh; reuse the returned prototype for repeated placements. | [`mesh/mesh.go`](../mesh/mesh.go) |
| `shape.NewMesh` | Validated reference mesh whose exported geometry must stay unchanged after its acceleration tree is built. | [`scene/shape/mesh.go`](../scene/shape/mesh.go) |
| `shape.NewMeshGroup` | Hierarchy of placed mesh members with transforms; it does not create set-YAML syntax. | [`scene/shape/group.go`](../scene/shape/group.go) |
| `shape.NewHeightField` | Finite sampled height surface, retaining its supplied height data. | [`scene/shape/heightfield.go`](../scene/shape/heightfield.go) |

Manufactory outputs and custom Go builders can also create indexed meshes;
their recipes and limits are covered in the
[constructor reference](constructor-reference.md). A returned `shape.Shape`
does not itself carry a material or world placement.

## Direct Go worlds, things, materials, and motion

`thing.World` is the assembly point. `NewWorld`, `World.Add`, `World.AddVolume`,
`World.AddField`, and `World.Lamp` add scene parts in Go.

A `Thing` value joins a shape, material, optional motion, and visibility
attributes; a `Volume` joins a density medium, material, optional motion, and
attributes. Neither value has a generic YAML-literal counterpart. See
[`scene/thing/thing.go`](../scene/thing/thing.go) and
[`scene/thing/field.go`](../scene/thing/field.go).

| Go API | Role | Set YAML counterpart |
|---|---|---|
| `thing.Diffuse`, `thing.Polished`, or `thing.Material` literals | Surface reflectance and optical response. | Named `world.materials` has its own colour/glow form; it does not serialize every `Material` field. |
| `thing.Made`, `thing.FinishOf`, `thing.Painted` | Build materials from the material/finish catalogues or a painted finish. | Set materials use `world.materials`; there is no generic finish expression in a set material. |
| `thing.NewVolume` or `thing.Volume` literals | Couple a `medium.Medium` to material and optional motion. | `world.wisps`, `world.fizzes`, and higher-level set adapters expose selected volume forms only. |
| `thing.Still`, `thing.Orbit`, `thing.Nested`, `thing.Spin`, `thing.Growing`, `thing.Keyed` | Fixed and composed time-dependent placements. | `world.motions` supports the reader's named keyframe/easing vocabulary; it does not expose every Go motion type. |
| `thing.Sun`, `thing.Lamp`, `World.Lamp` | Directional solar emitter or a world emitter. | `world.suns`, `lamps`, `spots`, and `floods` are separate checked YAML forms. |
| `thing.NewVolumeIllumination` | Checked world-level volume-light sampling configuration. | `scene.volume-illumination` is the set reader's corresponding setting, with its own choices. |
| `thing.Field` / `World.AddField` | Go-authored gravitational field that bends rays. | Not a general set-YAML world piece. |

The lower-level media APIs are Go inputs to `thing.Volume` and other scene
constructors. Named field values include:

- [`Air`](../scene/medium/air.go), [`Aurora`](../scene/medium/aurora.go),
  [`Clouds`](../scene/medium/clouds.go), and [`Disc`](../scene/medium/disc.go).

- [`Fizz`](../scene/medium/fizz.go), [`Giant`](../scene/medium/medium.go),
  [`Jet`](../scene/medium/jet.go), and [`Pillars`](../scene/medium/pillars.go).

- [`Plume`](../scene/medium/plume.go), [`Puff`](../scene/medium/medium.go),
  [`Tail`](../scene/medium/tail.go), and [`Wispy`](../scene/medium/medium.go).

Validated builders include [`NewWeather`](../scene/medium/mist_weather.go),
[`NewGrove`](../scene/medium/grove.go),
[`NewLayer` and `NewDeck`](../scene/medium/layer.go),
[`NewShellWeather`](../scene/medium/shell_weather.go), and the
[`NewPath`, `NewGrain`, and `NewWake`](../scene/medium/wake.go) family.

Only selected forms have a set reader: notably `wisps` and `fizzes`, plus the
higher-level atmosphere, ecology, fire, and planet adapters linked above. There
is no single YAML constructor for every exported `medium.Medium`.

## Where the truth lies

The set reader remains the source of truth where the prose guide and the actual
schema/stage support may differ. Named-scene parameters remain owned by each
scene's `Parameters` and `Check` implementation. Private helper functions and
every internal shape assembled by a scene are intentionally not presented as
stable public constructors.
