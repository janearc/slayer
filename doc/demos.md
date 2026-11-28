# Slayer demos

From a checkout with Go and Game installed, list the available scenes:

```sh
game run demos
```

Choose a numbered smoke or a named scene. These commands draw preview images
under `out/demos/`:

```sh
game run demo -- 1
game run demo -- orbital
game run demo -- camera
```

The camera demo also writes a short sequence of frames. A demo renders the
scene; it is not the reference comparison performed by `slayer assess --smokes`
or a test verdict. Some scenes need additional data, as noted in the menu.
For shared-machine rendering, follow the [Azazel guide](azazel.md).

Use `--dry-run` to inspect a demo without drawing it. For supported stills,
you can choose an output size or display a terminal preview:

```sh
game run demo -- orbital --dry-run
game run demo -- 9 --width 320 --height 180 --out out/first-smoke
game run demo -- 1 --terminal --cells 80x24
```

The demo's options and scene descriptions are in
[the catalogue](../cmd/demo/catalogue.yaml). House and camera use their own
scene sizes and refuse size overrides. See [camera motion](camera-motion.md)
for frame sequences and encoding.
