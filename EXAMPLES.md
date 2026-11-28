# what Slayer draws

start with [a small still](doc/examples/jobs/first-picture.yaml) or
[a short film with music](doc/examples/jobs/first-film.yaml). both use
Slayer’s CPU renderer and a built-in scene. MP4 encoding uses AVFoundation
on macOS.
see [getting started](doc/getting-started.md) for setup.

```sh
game build slayer
game run slayer -- draw doc/examples/jobs/first-picture.yaml
game run slayer -- draw doc/examples/jobs/first-film.yaml
```

look for `out/first-picture.png`, `out/first-picture.pfm` and
`out/first-film.mp4`. a PFM is a raw image that can be
[developed again](door/README.md) with different print settings.

## local model scene edits

The [local model fixture](contrib/local-model/BRIEF.md) turns a text
request into a bounded film and camera edit and renders a preview
with Slayer.

## scenes to try

| scene | what it shows | input |
| --- | --- | --- |
| lantern | a star lights haze through the gaps in a rotating swarm of panels | [lantern portrait job](doc/examples/jobs/lantern-portrait.yaml) |
| orbital | two inhabited rings turn in opposite directions around a beamed star | [orbital portrait job](doc/examples/jobs/orbital-portrait.yaml) |
| Webb at L2 | seeded models of Webb and the Larson with a distant sun and the Larson’s own lights | [Webb set](sets/webb-at-l2.yaml) |
| shoebox | a house, its materials and its cameras built from YAML | [shoebox set](sets/shoebox.yaml) |
| hillside | an observatory and a seeded forest on surveyed volcanic ground | [observatory set and setup](doc/observatory-ecosystem.md) |
| macrostructures | a gallery of seeded arcologies, spires and bridges | [macrostructures job](doc/examples/jobs/macrostructures.yaml) |

both sets and jobs can go straight to `slayer draw`. check a file and its
render setup without tracing a frame with `slayer assess FILE`. the
portrait jobs request Metal. the observatory also needs its elevation
model. see [renderer modes](doc/modes.md) and [set authoring](doc/sets.md).

```sh
game run slayer -- assess sets/webb-at-l2.yaml --width 320 --height 180 --rays 1
game run slayer -- draw sets/webb-at-l2.yaml --width 320 --height 180 --rays 1 --out out/webb.png
```

## smoke scenes

Slayer’s numbered scenes isolate rendering behaviour, including light
and shadows, volumes, optics, astronomical scale and bands. list the
scenes and groups available in your build, or assess a particular scene:

```sh
game run slayer -- list smokes
game run slayer -- assess --smokes 9 --out out/smoke-nine
```

`--out` copies the pictures and the comparison result. use `--fresh`
to redraw a cached result, or `--small` for a smaller frame with fewer
rays. see the [source](scene/smoke) for a scene’s assumptions and datasets.

## scenes by name

```sh
game run slayer -- list scenes
game run slayer -- list kinds
```

`list scenes` describes procedural scenes and the parameters accepted
under `scene.with` in a job. the names include `swarm`, `orbital`, `ring`,
`moonrise` and the `weather` map renderer. `list kinds` shows the kinds a
set can place. see [example jobs](doc/examples/jobs/README.md) and the
[spacecraft walkthrough](doc/model/spacecraft/guide/README.md) for more
files to open or draw.
