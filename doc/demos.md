# A first look at Slayer

From the checkout root, with Go and Game installed:

```sh
game run demos
game run demo 1
game run demo orbital
game run demo camera
```

`demos` lists the stable smoke numbers and the named scenes. `demo` draws one
of them. Numbered smokes use the actual scene factories in `scene/smoke`. They
are previews, without smokecheck's reference comparison or a test verdict.

Named demos are `orbital`, `swarm`, `ring`, `hillside`, `moonrise`, `house`,
and `camera`. These run the actual scene commands with modest first-look
settings.

The menu is checked from [catalogue.yaml](../cmd/demo/catalogue.yaml), where
the preview choices are identified as authored. Their defaults do not replace
the commands' own full-size settings.

Images go to `out/demos/`. Moonrise also keeps its linear `.pfm` raw. Camera
writes a poster and four seconds of numbered frames; its YAML plan describes
the move. See [camera motion](camera-motion.md) for frames, timing and
encoding.

Repeated camera demos resume only when their input and renderer hashes match.
If code or a plan changes, use a new output directory to keep films separate.

The named command is built into the output directory's `.bin/` first, so a
source checkout does not need a full `game build` before showing a demo. This
needs the module dependencies declared in `go.mod`.

The current development tree still names private review versions; those must
be available from the review proxy until the dependencies are released. No
demo silently downloads a survey or treats missing external data as a
successful picture.

## Terminal output and smaller previews

Put `--` before arguments that belong to the demo rather than Game:

```sh
game run demo -- orbital --terminal
game run demo -- 9 --width 320 --height 180 --out out/first-smoke
game run demo -- 1 --terminal --cells 80x24
game run demo -- camera --dry-run
```

Terminal mode keeps the PNG and shows a fitted version through Slayer's
existing sextant-cell renderer. Its fallback cell aspect is the usual 3/4
pixel shape; `--cells` controls the dimensions. It streams a single view and
does not take over the keyboard or install an alternate terminal screen.

The first-look settings are 640x360 and one sample along each pixel axis. A
numbered smoke retains a higher sampling requirement if its recipe asks for
one. House uses its set's film size, and camera uses its sequence plan; they
refuse size overrides instead of ignoring them.

The preview budget is 8,294,400 supersampled pixels. It bounds the image
scratch allocation; scene geometry and native allocations have their own
costs. Smokes 36 and 37 keep 9x9 samples, so use a smaller first look:

```sh
game run demo -- 36 --width 320 --height 180
```

Most basic demos need no external data. Smoke 38 needs the Blue Marble file
named in the menu. The larger terrain and astronomical-data examples still
have their own [setup instructions](grissom-setup.md).

## Game dependency

The Game version installed when this work began discarded arguments after the
run name. That makes `game run demo N` lose N. The companion local branch
`fix/run-demo-arguments` forwards literal arguments to the last configured run
step, leaving earlier setup steps alone.

It also passes application flags after `--`. Use a Game build containing that
fix for the selector examples.

On older builds, these fixed runs work without forwarding:

```sh
game run demos
game run demo-one
```

The Game fix is a separate change. No global binary or configuration is
replaced by Slayer's demo command.
