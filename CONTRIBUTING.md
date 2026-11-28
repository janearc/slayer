# Contributing to Slayer

Welcome. Slayer builds worlds from mathematics: shapes, growth rules,
materials, motions and light. A contribution can be a new model, a better
physical calculation, a renderer improvement, a useful scene, music or
clearer documentation. You do not need to understand the whole repository
to improve one well-defined part.

## Get oriented

Start with [building](doc/building.md) and [testing](doc/testing.md).
The [beginner guide](doc/getting-started.md) makes two pictures.
Read [package responsibilities](PACKAGES.md) before choosing where code goes,
and repository [agent guidance](AGENTS.md) if you use a coding agent.

Use Go from `go.mod`, compatible Game and your own source shelf/config.
The building guide covers the tools and source shelf needed before building.

Work on a branch and keep the change small enough to review. State the
behavior you want to change before touching shared geometry or transport.
For a large new physical model, explain the equations, source data, units,
approximation and intended renderer support before adding a large API.

## Make models through the manufactory

Slayer's objects are constructed by code and checked numeric recipes.
Contributing a model means contributing the mathematics that makes it,
not importing an ordinary mesh asset. Triangles are generated output.
Use the [modeling guide](doc/making-models.md) for the public API, a new-kind
workflow, the MCRN Larson example and how trees are built.

Reuse a construction rule when the new object is another member of its
family. Add a new primitive or architecture when the shape actually needs
one. Keep geometry in local coordinates, retain material distinctions,
describe attached lights and expose a checked recipe through the factory.
Scene placement, camera and development belong to scene/job adapters.

Named dimensions and counts need provenance. Distinguish measured values,
mathematical derivations and authored choices beside their declarations.
The Larson's saucer proportions are authored fiction; needle length ranges
have named botanical sources. Both are valid, with different claims.

## Develop with evidence

Write the behavioral test and its counterexample before implementation.
Use an independent expected result: a formula, an authored geometric
constraint or a separately computed reference. A test that repeats the
implementation's algorithm can repeat its mistake.

Positive tests need an anti-test that demonstrates the distinction matters.
Mark it `// anti-test of TestName` on its own line. Prove important guards
can fail with a bounded counterexample or mutation; a marker alone is not
proof. [Testing](doc/testing.md) explains tiers, real-device checks and
how to update the inventory without hiding lost tests.

Keep generated output deterministic under the declared inputs. Use explicit
seeds instead of time or a shared random stream. Validate invalid dimensions,
counts and placements at construction boundaries. Preserve the first error
and return no successful partial specimen. Do not weaken refusals to make
an unsupported backend appear to work.

Aim for complete statement coverage of changed Go code, and report the
actual scope. Go coverage does not instrument Metal. Keep generated mocks,
commands and defensive paths visible rather than excluding them to raise
the number. The testing guide describes the remaining instrumentation limits.

## Review a model or renderer change

Check construction without a render first, then use a small CPU picture.
For GPU support, compare actual Metal output and unsupported-input behavior.
Inspect the silhouette, winding, normals, surface gaps, material boundaries,
shadows and camera motion at the representation you intend to ship.

Compare the same scene, seed, time, samples and development. Record the
source revision and device. Report raw-light differences separately from
developed-image differences and temporal acceptance. A faster smaller
picture does not establish a quality-preserving optimization.

Geometry signatures in `manufacture/testdata/` protect authored specimens.
Treat an intentional shape change as a reviewable change with before/after
renders and a reason for new references. Do not regenerate fixtures merely
because a refactor fails them. Instrumentation can itself change vertex
bits; keep that failure distinct from ordinary geometry acceptance.

## Prepare the contribution

Explain the specific problem and resulting behavior in the change
description. Include focused test commands and outcomes, skips, measured
coverage scope, source/provenance changes and relevant image comparisons.
If a feature is a prototype, identify the remaining acceptance work.

Keep `burning-trousers/` changes in commits separate from renderer and
other production changes, including merge resolutions. Preserve numbered
smokes; changing an existing smoke is a separate commit. Use pinned mock
generation where interfaces need doubles. Do not replace real physics or
device comparisons with mocks.

Update guides and examples with an interface change. Use Game for builds,
checks, lanes and publication workflows. Run lint and inspect its findings;
existing repository debt is not permission to add more. Publication and
release are separate maintainer actions; a local passing branch is not a
release. [Azazel](doc/azazel.md) explains the current queue boundary.
