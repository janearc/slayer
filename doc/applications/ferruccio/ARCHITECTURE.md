# Ferruccio inside Slayer

Ferruccio is a separate macOS executable. Its draft module keeps YAML as the
authoritative scene, validates it against Slayer's schemas and makes edits
through one undo history. Selection identities refer to authored source
paths and survive placement changes.

The native window uses GoGPU for drawing and input. A separate staging
helper runs Slayer's scene preparation and returns shared meshes and
placements. The working view uses those meshes for drawing and picking.
[Working view](WORKING-VIEW.md) describes its proxies and interaction limits.

Slayer produces the offline image. The Ferruccio preview module freezes
the request, [submits it to Azazel](AZAZEL-CLIENT.md) and accepts the result
only after checking job identity and output. The UI thread owns visible
state. Background staging and rendering report results tagged with the
source generation so an older result cannot replace new work.

[Build and checks](BUILD.md) lists the app and disk image targets.
