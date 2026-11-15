# Keeping the harness independent

burning-trousers began as a separate repository so that a single change
could not alter both the tracer and the test used to validate it. It now
lives in Slayer, with three rules preserving that separation.

1. Use only Slayer's exported APIs. Do not import internal packages.
   The tests in `rules/` enforce this boundary.
2. Compute expected values independently, from physical laws, published
   tables, or the harness's own arithmetic. Reviewers must check this;
   an import test cannot establish the origin of an expected value.
3. Keep harness changes in separate commits. Such commits may touch only
   `burning-trousers/`, plus necessary `go.mod` and `go.sum` changes.

The module-file exception excludes lines naming the project's libraries: those
versions pin the physics used by the renderer. `rules/` examines every
commit since integration, including merges and renames, and reports
violations of either boundary.

To inspect harness commits manually:

```sh
git log --stat -- burning-trousers
```

Check that each commit respects the path and dependency restrictions.

History checks start at integration commit `03d5d44`. Bring work forward
by fast-forward or merge. Rebasing that history changes the integration
commit's identity and causes the rules test to fail.
