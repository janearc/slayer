# Ferruccio sprint status

Historical checkpoint from 25 November 2026 at road revision `f9b1965d`.
For current commands and capabilities, see the [Ferruccio guide](README.md).

## What works in tests

- Opening does not start an offline render. Command-O and unsaved-document
  protection are implemented; explicit renders use the configured transport.
- The working view stages Slayer scenes in a separate helper. It shares meshes,
  identifies supported authored placements, and reports omissions and proxies.
- Supported space and terrain objects can be added, moved, turned and scaled.
  A generated terrain heading can be replaced deliberately with an authored
  angle. Some generated stands expose an editable region; members are not
  individual authored objects. Other placement forms remain unsupported.
- A test added, moved, scaled, undid, redid, saved and reopened a fish, fern
  and city. Two small Azazel CPU renders used that saved scene and its profile
  books. The native-window authoring workflow has not been accepted.
- Full-render setup offers producer output sizes, cameras and CPU/GPU choice.
  Existing physical rigs can change frame and eligible jobs can change stock.
  Exact arbitrary samples per pixel is a CPU primitive, not an integrated
  job, GPU, queue, receipt or UI feature.

## What still needs work

Futian has produced headless working-view frames, but native-window loading
and interaction are not yet accepted. Its sea is a mean-level proxy, and
volumes and lights are omitted with notices. All-scene coverage, full
authoring semantics and native interaction remain open. A rigged queued
render and exact-count CPU/GPU parity remain unverified.

The last complete workspace test run had two text-width assertion failures
and a full-render button timeout; the timeout also failed in isolation. The
draft suite and the focused heading, undo and staged-geometry tests passed.
Independent review remains open.

## Distribution and CI

`game build ferruccio` and `game build ferruccio-dmg` build an app and a disk
image for the host architecture. The app includes Ferruccio, its staging
helper, Slayer, Azazel, Game and examples. A queue must still be configured
for rendering, and some scenes need external data. A clean-user install,
runtime dependencies, data redistribution, signing, notarization and minimum
macOS version have not been accepted. This is not yet the self-contained
distribution described in the sprint plan.

The manually started GitHub Actions workflow `Ferruccio Apple silicon build`
is intended to fetch and verify the source shelf, build through Game on an
Apple silicon runner, check the image and upload it as an Actions artifact.
It does not publish a GitHub release. Its first run is still required to
establish that the clean runner can obtain all sources and build successfully.
An artifact is a build candidate, not proof of portability or release approval.
