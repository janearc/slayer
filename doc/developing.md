# Working on Slayer with Game

[Building Slayer](building.md) covers the required tools, source shelf,
builds and checks. Run Game commands from the checkout root; project runs
and targets are declared in `.game5/`, while your Game configuration holds
machine-specific paths.

A named run uses the built binary. Rebuild after changing its code, and put
application arguments after `--`:

```sh
game build
game run slayer -- --help
game run slayer -- draw doc/examples/jobs/first-picture.yaml --dry-run
```

A run entry in `.game5/runs/` is an argument list, not a shell script: it
does not expand shell variables or execute pipelines. Current Game versions
forward arguments after `--` to the application.

For development, start with the affected test package. `game check` runs
formatting, vet and the non-draw tests; `game sweep` checks the committed
tree for distribution. [Testing](testing.md) explains the test tiers,
resource limits and real-device checks. [Azazel](azazel.md) covers shared
render work.
