# Checking ninjafilm

Run from the Slayer root with the [Game setup](../../doc/developing.md).
Keep heavy checks serial, including race checks and mutation runs:

```sh
export GOMAXPROCS=3 GOFLAGS='-p=1 -parallel=1 -timeout=60m'
game run check-film
game check
game cover
```

The recorded statement-coverage target is 100% in each package.

## Mutation testing

Run one package tree at a time, with at most three workers and a separate cache.
The selector is relative to the Slayer root:

```sh
game mutants --workers 3 --only '^ninjafilm/lay/'
```

The tool changes one operator, integer, or `true` value and runs the tests.
Tests must catch each meaningful change. Equivalent survivors belong in
`.mutants` with an explanation.

Some generated mutations cannot compile: subtracting strings after changing
`+` to `-`, or removing a statement required for a return path. These are
invalid mutations rather than evidence of a Go compiler fault.

Generate the mocks reproducibly with `game run mocks`.

## Race checks

```sh
game run check-film-race
```

The named run allows ten minutes. Race instrumentation adds substantial cost;
use `game cover` for the coverage report.

Recorded timings on the optics branch were about 235 seconds for optics and
88-94 seconds for phase on a busy machine. These are historical measurements,
not a timing guarantee for the current checkout.

## Runaway guards

Package `main_test.go` files install a guard that reports all goroutine
stacks and stops a test process when either limit is reached:

- Three quarters of the test timeout: 450 seconds for the race command above.
- The heap ceiling recorded in that package's `testdata/numbers.yaml`.

Heap ceilings were set at about four times measured test usage. If a valid
new test needs more, measure the new usage and record who changed the
ceiling and when.

Mutation testing especially needs this guard. A changed loop can allocate
indefinitely, and stopping `go test` may leave its test binary behind.
One such process reached 7.5 GB. Inspect orphaned test processes after a run:

```sh
ps -Ao pid,ppid,command | awk '$2==1 && /\.test /'
```

Before stopping a process, verify that its executable belongs to the
current run's temporary directory.

## Adding a stock

Add the stock to `data/stocks.csv`, its records to `data/records.csv`, and
its provenance to `data/SOURCES.md`. Existing stock models need no code
change. Update the expected names in `TestTheShelfHoldsTheStocksOfTheHills`.
