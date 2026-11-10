# What the overnight checks established

The frame, patch and streaming drafts initially deferred execution checks.
Focused checks have now run on Go 1.27.3 through Game 5 RC d4fac338, one
job at a time. This records those scopes, not a completed release gate.

## Geometry and source caches

At 18b7dc8, independent frame and pinhole witnesses passed. They cover
cardinal and relative-origin geometry, rational projection expectations,
frustum boundaries and refusal cases. The pinhole YAML remains an authored
expectation book; the Go tests read it instead of recording their own output.

At c4cee0f, all eight terrain source/cache witnesses passed. Tiny generated
TIFFs exercise byte and entry limits, LRU eviction after reuse, source-file
loss, absence and malformed coordinates, stricter allowances on cache
hits, equivalent side requests and grids held across Clear and Close.

These checks do not establish mesh selection, source projection or height
datum conversion, seam handling, moving-view residency or a world renderer.
The tile cache and patch cache remain separate from consumer-held memory.

## Streamed sample finishing

At c578590, three host-finishing tests and their 46 pixel cases passed.
An independent fixture supplies 3x3 samples across a 64-row boundary and
a short final band, with a poisoned reusable buffer, two texture records,
sky, coverage, nearest depth and four distinct diagnostic totals.

The expected colours are closed-form arithmetic. No old averaging call
provides the answer, and NaN cannot pass the numeric comparison. Diagnostic
counts and payload sizes here are authored fixture values, not observations
of native queries or whole-process memory.

Matched shader fields, device-buffer lifetime and failure cleanup still
need their own checks. No timing improvement or lower RSS follows from
the host-finishing result.

## Selected renderer checks

At 77c1540, the requested packages compiled and selected CPU/Metal scopes
passed. The observatory mesh/material comparison and the guide beam through
a level mist deck executed on Metal without skips. This establishes those
specific scenes, not every new factory or atmosphere combination.

At a3a7aba, the 123 hard prose findings were cleared. Game lint still exited
1 for 1,853 review findings. Sweep, Shuriken, broader coverage, mutations
and integration remain open; none is waived by the focused results.

The crowded-summit YAMLs are still [draft study inputs](quality-summit.md).
Their Moon and weather are authored. No new PNG, movie, 4K timing or visual
acceptance is claimed here.

## Repeat a scope through Game

Run each command separately under the machine's resource policy:

```sh
game test -- -count=1 -p=1 -parallel=1 ./scene/frame
```

```sh
game test -- -count=1 -p=1 -parallel=1 ./scene/footprint
```

```sh
game test -- -count=1 -p=1 -parallel=1 ./terrain -run '^Test(Streaming|PatchCache)'
```

```sh
game test -- -count=1 -p=1 -parallel=1 ./gpu -run '^TestReadback'
```

The last scope is host-only inside the Darwin GPU package; it does not
open a kernel or require a device skip. A full check or native render is
a separate job. The original run logs retain their exact source pins;
later comment and documentation composition does not invent a new run.
