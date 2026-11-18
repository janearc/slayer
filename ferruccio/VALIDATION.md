# Earlier Fyne prototype validation

Current GoGPU evidence is in [native/VALIDATION.md](native/VALIDATION.md).
The sections below preserve the earlier Fyne work and its measured limits.

Game5 build 511afb9, Go 1.27.3, Fyne 2.8.2. Processor limit 2; Go build
parallelism 1. Shared Game lanes were used for tests, builds and captures.
No reviewer agent was requested. The smaller test agent wrote focused
preservation, widget and command tests.

## Automated checks

The latest complete `game5 check --cover` passes formatting, vet and all six
package suites. There are 27 named tests and three executable Puzzlemouth
examples. Game lint is clean with the configured rules unchanged.

The checks exercise comment preservation, reciprocal YAML/settings changes,
invalid drafts, duplicate and unknown keys, refusal without mutation, saving and
reloading unfinished work, camera/sound/timing preservation, navigation without
draft loss, omitted settings, bounded document loading, screenshot overwrite
refusals, command dry runs and invalid command input.

Initial document and UI tests failed against their stubs before implementation.
Schema guidance and capture-file preservation also have actual failing stub
runs. Exact local records are retained in `checks/`; subsequent regression tests
extend those contracts. No setup failure is counted as an assertion-red result.

Statement coverage is 62.3% overall, 89.2% for the document implementation and
100% for the theme. Interactive file-dialog code is 8.0%; dialog callbacks and
native event-loop paths need further automated coverage before a release.
Coverage is not represented as complete.

## Visual checks

The palette gate checks prose, disabled text, placeholders, links and selected
tab text on background, input, menu, overlay, header, button, hover, selection,
disabled-button and focus surfaces, plus selected navigation. All 51 pairs pass
the fixed 10–12.5 contrast band and the hue rule, using libtheme's existing colour
and contrast implementation. Body type uses 24 with regular weight; code uses 20,
captions 18 and the smallest explanation 16. Fyne's normal scale setting applies.

Actual Fyne widget trees were captured at 1800×1120 and 1440×1000. The first
compact image exposed an overly small YAML pane; source and settings were
separated into tabs. A later compact image exposed horizontal clipping in the
settings grid; settings now use one vertical column. Earlier images remain in
the shared gallery.

The first native smoke crashed in Fyne's explicit monitor-centering call.
Removing that optional call allowed the native macOS GLFW window to open,
capture its own canvas and exit successfully. Its receipt records native
output at 3600×2196 on the actual display. Software captures are labelled as
software captures and are not used to claim native-window success.

## Acceptance boundary

This is a reviewable interface prototype on its own road branch. It has not
been merged or released. Full mutation, property, Shuriken, Okapi and independent
acceptance work is not claimed. Agent review was explicitly deferred until
after the morning reading. See `MORNING.md` for the working interactions and
the reserved renderer, set-authoring, motion and music integrations.

## Renderer connection

The next phase adds actual job/set loading and still previews. A cheaper test
agent first ran the Open/Prepare contracts against empty stubs and observed
assertion failures. After implementation, all package tests passed in check34;
focused widget tests also exercise existing files, render gating and adjustable
type without launching the renderer.

Native proof36 opened the unchanged first-picture YAML, invoked the same
Render preview handler as the button, waited for Slayer through Game, decoded
the returned PNG and captured the updated native window. It passed at source
c37bfab. The displayed image is an actual 640 by 360 Slayer result, not mission
reference imagery. The bundled renderer is the existing clean build 07186e109;
Game is the requested 511afb9. Neither executable was installed on the host.

The first Vaporwave render capture remains available as an intermediate view.
The supplied desktop reference then guided quieter lavender body text, keeping
pink for actions/headings and cyan for accents. The original full palette gate
remains in place, including focus backgrounds and primary text. Text size is
adjustable; code follows the chosen size with a minimum of 16.
