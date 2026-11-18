# Local review bundle, 15 November 2026

Portable provenance note (15 November 2026): personal identifiers are roles
and private locations are documentary environment bindings. Source commits,
hashes, measurements and test failures are unchanged. Resolve bindings from
the original retained shared-render recipes; this document does not rename
evidence directories or claim that a generalized path is an original path.


Open `bin/Ferruccio.app` from the parent directory. This bundle contains
the tested frontend and the new Oakland renderer. It is local review work;
no distribution release or independent review is claimed.

| Component | Game build stamp |
| --- | --- |
| Ferruccio frontend | 540a635 |
| Slayer with Oakland | c6a3002f |
| Catalogue with editable orbit | 0b0d207e |
| Game5 | 511afb9 |

SHA-256 of the packaged executables, in the same order:

```text
9100f1c78eaea9b67a9b414307c25893d54e45c94924be70002919f780dcb710
5c7ab569e03d3123bf6888c94eac94fc2847e56d2f2011f4fa016cd7e89ae43f
cee0a127dd7694d3499f08fedcd5c21237d1f8be954cec4d9ec48eda00ff8834
42461cc5016ee519cdc9ec713c48f36257dbee9637d542284dcd7bc2ba823b72
```

Each bundled executable matches its staged input byte for byte. The plist
passes `plutil -lint`; the launcher passes `sh -n`. The launcher supplies
the Puzzlemouth `open` verb and explicit engine, examples and Game paths.
The application is separate from Grissom.

The final scene lighting was adjusted after building the executables.
Examples are external bundle resources, not compiled into the frontend.
The source and packaged copies have these SHA-256 values:

```text
ced732dbb44600e46dba0c5be1458c852ac80af49f2860a001892641fc5bccea  set.yaml
a36af6c635ef8acf17e9f1365a6a20f78dda2cfa8a0ccbdda904b3bd09acea57  two-metre-human.gltf
39581a2eb0f4bf177a2cace98f29302e5491ab3053de23dd9ce4d4a15e1aeb8c  two-metre-human.bin
809a98ed936f2036ba5f8d4d277c9a3d3c768dce44e3a90af0d70fa2b9b81aa1  LICENSE.txt
```

The fixture is original CC0 geometry from Oakland core c76ca61a.
The scene uses explicit native matte material and two directional studio
lights. Its dimensions and appearance are authored, not physical readings.

Chroma is pinned to 284aab1761ca9202052caee4e3b65e4e9c0a456f.
Its exact CSS, licence and role choices are recorded under
`internal/appearance/themes/`. The module was made available through a
local file proxy because that source revision was unpublished. Slayer
retains its existing local Celestial replacement. Neither prerequisite
is a claim of a complete published dependency closure.

Builds used Go 1.27.3, `GOWORK=off`, `GOMAXPROCS=2` and Go parallelism one.
Native GoGPU uses `CGO_ENABLED=0`; Slayer uses `CGO_ENABLED=1` on macOS.
Game build93 stamped the frontend. Oakland build34 passed its source
verification and all 17 command builds. Later documentation and external
scene commits do not change those executable stamps.

The shared evidence starts at:

```text
${FERRUCCIO_ATELIER_EVIDENCE}/index.html
```

That path is relative to the vorrh. Captures12–16 show the current packaged
interaction proofs; `oakland/index.html` contains the explicit CPU/Metal
comparison. Earlier unsuccessful attempts remain labelled and available.
Read [validation](VALIDATION.md) for checks, measurements and remaining gaps,
and [the architecture](../FERRUCCIO-OAKLAND.md) for the source ownership map.
