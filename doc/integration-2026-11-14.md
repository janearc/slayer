# Integrated development tips on 14 November 2026

The documentation describes the local `feat/sextant-throughput` branch
after the gravity prototype and current renderer/music work were combined.
No push or release is implied by this integration.

## Source revisions

| Source | Integrated revision |
|---|---|
| original throughput/gravity branch | `2c34ff9f` |
| release candidate | `b3d592f8` |
| gravity/RC merge | `7653449a` |
| the renderer and music-film tip | `adc069d1` |
| merge of that tip | `50b26e2c` |
| the publication checkout tip | `10aff2fc`, already in RC ancestry |
| the summit profiling checkout | `b3d592f8`, already included |

All branch heads from the publication repository and the beats/shoot
repository were fetched and checked against the integrated tip. No further
production branch was missing. Twelve divergent branches there are
deliberate fault witnesses, retaining broken operations for counterexample
checks. They are not implementation candidates. A separate review
queue has unrelated history containing reviews/tasks, not Slayer sources.

## Focused validation

The renderer merge was conflict-free. Selected host packages passed:
beats, cuesheet, job, hillside, Omnipositor, searching and cmd/beats.
The independent harness history rules passed without modification.
The new real-Metal surface/volume pass test and gravity RC guards passed
as a focused sample. The earlier gravity/RC sample is recorded separately
in [gravity integration](gravity-rc-integration.md).

Source-shelf verification passed. A temporary single-command build checked
CLI help and starter-job dry runs; it has no release provenance stamp.
Both beginner jobs rendered: a CPU PNG/PFM still and eighteen CPU frames
with synthesized music. The resulting three-second movie probes as H.264
with AAC audio at six frames per second. Its 90-row image is padded to 96
rows by the encoder.

Every library tune passed the loading checker; this
is not listening acceptance.

The integrated inventory contains 6285 test names: all 6269 names from
the gravity/RC merge plus sixteen additions, with none lost. The complete
tier listing check passes with zero problems: 5021 non-draw and 1264
draw-only names.

The full GPU suites were not rerun: a sample was asked for after the
earlier full run. Repository lint still reports inherited findings;
new guide prose was checked against its paragraph rules. Documentation
links and patch whitespace were checked separately. Timing claims in the
guides retain their original scene and measurement context.

## Contributor and bootstrap follow-up

The mathematical-model guide's actual Go fragment compiled and constructed
the landed Larson. The four selected Larson/shoot construction tests and
the documented Azazel budget/engine/command tests passed without rendering.

Bootstrap checks passed with controlled command boundaries: no-write
inspection, missing local-pin refusal, setup ordering/reuse, module/source
failure stops, unmanaged-config preservation, path aliases and local Game
installation/reuse. The real prerequisite-only check passed without making
a setup directory; the generated Game config shape also parsed with the
real runner. An actual network installation/full bootstrap was not run.

Shell syntax, local links and whitespace were checked. New guides have no
lint findings; repository-wide inherited lint findings remain. No full
draw suite or release was run for this follow-up.
