# Release checks

From a clean Slayer checkout, run:

```sh
game run ship-ready
```

This checks the commit at HEAD. Set `GAME` to select a different Game
executable. The summary marks dependent stages skipped if their inputs
failed. A failed required stage produces exit status 1.

1. The tree checks run Game coverage, signature generation and sweep.
   The summary reports lint and Shuriken counts without making them
   failure gates. If a package reaches the Go test timeout without a failed
   assertion, it is marked `slow` and its later tests remain unrun.
2. Workflow checks run actionlint and shellcheck, and compare each
   Ferruccio build module's Go requirement with the workflow toolchain.
3. The clean build archives HEAD with the distribution omissions, then
   builds the disk image with an empty module cache, the public Go proxy,
   a fresh Game config and no local source shelf. The local Go build
   cache is reused. The same image checker runs in the GitHub workflow.

## Optional GitHub runner

`game run ship-ready -- --runner OWNER/REPOSITORY-dryrun` also runs the
Ferruccio workflow on GitHub's Mac. It force-pushes the archived copy over
`main` in the selected repository. Use a designated disposable repository
whose name ends in `-dryrun`. The command waits for the workflow result.

## Results

Read `bin/ship-ready/<time>/summary.txt` and the logs beside it. A failed
clean-build or runner stage keeps the temporary checkout and module cache
for inspection. Signature generation can update
`.game5/metrics/analysis.md` in the original checkout.
