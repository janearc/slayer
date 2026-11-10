# Palomar, both Kecks, and Subaru

The first ground-family slice is on `feat/ground-observatory-families`.
The models landed at `5ea0659`; `a24e394` adds construction-error and budget
contracts exposed by mutation testing. Work remains local.

These are scenery templates. The [family design](observatory-family-design.md)
separates source dimensions from picked supports, pointing and coatings.
Keck I and II deliberately share one template for separate site placement.

## Checked behavior

All four details have finite closed meshes, stable physical bounds and the
expected primary aperture. Both Keck primaries have 36 connected panels,
counted independently of their triangulation. Both scene readers accept
the names. The earlier 176 geometry rows remain byte-identical.

The manufacture, Set and catalogue suites passed for the model checkpoint.
After the error-handling cleanup, catalogue and ground contracts, all 208
geometry records, and the native/software GPU comparisons passed again.
The replay also ran unchanged manufacture and catalogue suites first.

Actual Apple M3 Max comparisons used CPU, software Metal and native closest-hit
rendering. The largest native ground-view relative RGB error was 0.000548662,
against the existing 0.005 budget. Those small comparison views had zero
native fallbacks; the larger specimen images have their own counts.

Eight 800 by 600 native specimens, at four samples per pixel, were rendered
and individually inspected. The output is in the local home under
`scratch/renders/2026-11-07-ground-observatories-01/`. Each PNG retains its
settings, hashes, native outcomes and executing-binary hash.

Those pictures came from the model draft before its commit. The compiled
revision from `go run` is explicitly unknown. `inspection.json` records this
limit; it does not substitute a nearby checkout hash for the binary's identity.
The enclosures hide some mounting detail in these views.

## Mutation receipts

The complete original 166-case run at `5ea0659` caught 152 faults, left 11
survivors and had three unbuildable cases. None timed out. The raw report is
retained without rewriting its results.

Fresh mutations at `a24e394` cover every changed construction/preparation
method, the data reader, camera elevation, maker error handling and slit-array
constants. This 42-case run caught 38 faults, with one equivalent change,
three unbuildable cases and no survivors or timeouts.

The strict mapping requires each of the ten real original survivors to be
caught by its intended new regression test. The original scope resolves to
162 caught faults, one equivalent change and three unbuildable cases.
This combines two actual runs; it is not a second full 166-case run.

The equivalence adds a fifth array slot to four named corner roles. That slot
is never read or emitted as a vertex. No reachable error guard is waived.
Parsing failures use malformed source bytes; tests do not change global
error flags to manufacture coverage.

Raw reports and the mapping are under `out/ground-review/`. The local home
also keeps a copy under `scratch-tools/slayer-ground-proof-2026-11-07/`,
with checksums for the preserved receipts.

## Remaining gates and features

Fresh Shuriken has zero new findings against the pre-catalogue baseline.
Game lint has zero hard findings and 1,292 existing review notes. Game sweep
still reports seven older files. The whole Game check, broader catalogue
mutation/coverage work and release integration remain open.

Rubin, ELT, Griffith and historical OWL still need distinct assemblies.
Radio dishes and arrays, radomes, wooded sites and the mountain flyby remain
in the [feature backlog](feature-backlog.md).
