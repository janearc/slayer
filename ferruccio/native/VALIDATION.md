# Native frontend validation

GoGPU pins: gogpu 0.54.1, ui 0.1.55, gg 0.52.7. Game5 build 511afb9,
Go 1.27.3, Cgo disabled, two processors and Go parallelism one. This is a
reviewable prototype; no release or independent acceptance is claimed.

## Checks

Game check28 passes formatting, vet and all four native package suites: 27 named
tests and two executable examples. Statement coverage is 49.0 percent.

The native .game declares the pixel-test packages so Game takes the tests and GPU
lanes in order. An earlier attempt to wrap check in the GPU lane was refused
before tests, because its tests lane must come first; it is not a passing gate.
Native callbacks and dialogs remain substantial
coverage gaps. No coverage threshold was moved or exception added.

Tests cover Unicode editing, selection, focus, clipboard routing, undo limits,
large insertions, bounded scrolling and type size; failed open preservation,
source-path/hash render identity, independent example copies and actual image
pixels/aspect fitting. Runtime paths are resolved before changing the renderer’s
working directory. Example saving requires a chosen destination. Puzzlemouth
examples, dry runs and argument refusals run without creating a native window.

Document identity and runtime-location tests first failed against empty stubs.
The editor’s first red result was a missing-symbol compile failure; a deliberate
insertion fault later produced an assertion failure and was restored. Initial
appearance checks were cache-blocked; those are not claimed as assertion-red.
Its final widget-size regressions draw real pixels, without a handwritten fake.
All 51 palette foreground/surface pairs retain the fixed contrast and hue rules.

## Actual native checks

The isolated Metal probe selected Apple M3 Max/Metal and opened its window.
Its first exported PNG was black: calling RenderToImage inside the active draw
routed drawing to the window instead of the offscreen target. Moving the probe’s
capture to OnClose fixed this. The second probe asserts violet and red pixels.
No third-party module source was patched.

The first full-workspace attempt called StartAnimation before GoGPU initialized
its animation controller and crashed without producing an image. Moving it into
the first update fixed the proof. Capture03 then opened a Metal window and
exported the real widget tree with an existing first-picture YAML file.

Capture04 invoked the same StartPreview handler as the visible button, ran the
existing clean Slayer checkpoint 07186e10 through Game’s shared GPU lane,
validated its fresh PNG and displayed it. The final view was visually inspected:
it shows the actual rendered sphere, not the earlier Venus mission reference.

Workspace PNGs are explicitly labelled offscreen GoGPU UI widget-tree exports;
they are not OS-compositor screenshots. The actual native process independently
selected Metal, ran the UI loop and exited normally. All saved images, including
the first blank backend probe, are retained and labelled in renders.

Capture05 rendered the existing Fizz scene, and exposed a missing status footer
after the busy indicator was added. The Box container uses constructor children
for layout, independently of WidgetBase.AddChild. Building the footer children
before constructing the HBox fixed it. The actual visible-tree test failed first.
GoGPU buttons lack a public accessibility-label method in this version, so their
stable widget IDs identify the log control in that regression test.

The packaged app is stamped a3d8456. Capture06 ran its actual binary with its
own bundled Fizz YAML, Slayer and Game5, including paths containing spaces.
It selected Metal, completed the real preview and displayed it with the restored
status and log controls. The final image was visually inspected. Bundle plist,
shell syntax and identical frontend/engine binary checks passed.

## Limits

Preview quality is deliberately 640 × 360 with one ray per side. The bundled
engine is an existing clean checkpoint, not a claim that this GUI includes the
latest Slayer RC. A supplied Webb example currently yields a dark crop at this
preview configuration; it is a scene/framing limitation, not a working showcase.

The YAML editor is deliberately small. It does not provide syntax completion,
full grapheme-cluster editing or a language server. Type size applies to the
current window and is not persisted. Save/open confirmation and native dialogs
need more interactive automation. Film/video, motion planning and music are
reserved; no fake preview or fabricated render percentage substitutes for them.

## Catalogue settings repair

User testing exposed an unsupported format: the Subaru catalogue settings file
was being validated as a job. The repair is frontend ac79e24. Draft and preview
regressions failed first, then passed with catalogue recognition and routing.
An explicit lost-local-lights fault failed the complete snapshot-mapping
assertion; the exact source was restored and the focused test passed again.

Native Game check34 passes all four packages, including receipt-selected
specimen loading, standalone-plan selection, relative executable paths,
wrapped diagnostic pages and real-canvas font measurements. Root focused
checks53/56 pass. The previous 49.0% coverage figure belongs to the earlier
frontend, not a new measurement of this change. No thresholds were changed.

Packaged proof61 opened the exact Subaru settings from the reported render
folder, recovered subaru from its receipt and completed a fresh native Metal
catalogue render. Capture07 displays the actual new specimen image beside its
unchanged YAML. The original source was compared byte for byte afterward.
The frontend and bundled executable copies, launcher and plist were checked.
The output is an offscreen widget-tree export, not an OS window screenshot.

Catalogue is Game-stamped a89af6be, SHA-256
882c0216ab5d3735d5323338f2e3b3962da6b58541ee3029465953ea3aeb8f2f.
Its declared Celestial replacement is local and unversioned; see the catalogue notes.
The all-command Slayer build was interrupted after catalogue was stamped;
this records a usable catalogue binary, not a successful all-command build.

Packaged proof63 also renders the existing first-picture job through Slayer and
displays its fresh sphere image in capture08. Both new views were inspected and
published with their receipts. Root lint62 passes with Cgo enabled for Fyne;
native lint35 uses Cgo disabled for GoGPU. The full native Game check34 includes
the final dialog code. There was no independent review, as requested.

## Current integration checkpoint

Native Game check86 passed all four native package suites. Root check87 passed
the draft, preview and assets packages. Native lint check88 found two missing
function comments; those comments were fixed. Native lint89 and root lint90
then exited zero.

The earlier packaged orbit proof, capture09, used incorrect coordinate deltas
and does not establish correct orbit behavior. Capture10's zoom proof passed.
The first undo proof failed to restore the pre-gesture draft; the undo path has
since been repaired. Fresh packaged proofs below supersede those failures;
the failed artifacts remain available. Gesture proofs are separate from
the passing package tests.

The editor's real-canvas theme regression first failed on both Vaporwave and
Fermi Long because its background and text ignored the active Chroma theme.

After the editor switched to current appearance roles, both-theme background
and readable-ink checks passed, as did the checked selection-pair regression.
The initial assertion-red was observed in the console but was not saved as a
log; no red-log path is claimed. The final native check91, including all four
packages and the editor regressions, is `native/checks/91-chroma-editor-check.log`.
Native lint92 is `native/checks/92-chroma-editor-lint.log` and reports clean.

## Packaged Ferruccio and Oakland, 15 November 2026

Game build93 produced frontend 540a635. The bundle contains Slayer c6a3002f
and catalogue 0b0d207e. Exact hashes are in `BUILD-PROVENANCE.md`. The final
lighting YAML is an external resource; it is recorded separately from the
executable stamps. No source code changed after check91 and lint92.

Fresh native-window proofs exercise the packaged executable and real renders:

- Capture12: Observe, Vaporwave. A 100 by 40 pixel drag produces yaw -35
  and pitch 14 degrees; the rendered image changes in 104,575 pixels.
- Capture13: Compose, Fermi Midnight. Scrolling changes field of view
  from 50 to 37.040911034085894 degrees; 135,946 pixels change.
- Capture14: Command-Z with the image focused restores the exact draft,
  automatically renders it, and restores identical decoded image pixels.
- Capture15: Observe, Vaporwave. The bundled Larson study renders with
  the two-metre import, matching source hashes and an import receipt.
- Capture16: Compose, Vaporwave. A 100 by 40 pixel drag changes the set
  camera from [18,-24,12] to
  [1.1418646093800149,-26.117649200335975,18.81809366964031];
  158,783 rendered pixels change. The original 1,793 source bytes and
  bundled geometry resources remain unchanged.

These are widget-tree exports from actual GoGPU Metal windows. UI backend
selection does not establish which backend a Slayer job chose. A separate
explicit CPU/Metal render pair verifies the imported study on both paths.
Its mean absolute RGB byte-channel difference is 1.05865 and RMSE 3.09620
at 640 by 360; these are developed-image metrics, not radiance equivalence.

All captures, raw renderer outputs, receipts and failed earlier attempts
are retained in the shared renders gallery. The native tests and these
interaction proofs passed; complete UI coverage and independent review
are not claimed. Film, music, USD and object handles remain outside this slice.
