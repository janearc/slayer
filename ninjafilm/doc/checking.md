# Check ninjafilm

From the Slayer root with [Game configured](../../doc/developing.md):

```sh
export GOMAXPROCS=2 GOFLAGS='-p=1 -parallel=1 -timeout=60m'
game test -- ./ninjafilm/...
game test -- -race ./ninjafilm/...
```

Use `game check` and `game cover` for the full project. To select a package
for mutation checks, pass its file-path prefix to `--only`:

```sh
game mutants --workers 2 --only '^ninjafilm/lay/'
```

Run `game run mocks` to regenerate mocks.

## Add a stock

Add the stock to `data/stocks.csv`, its records to `data/records.csv`, and
its provenance to [Sources](../data/SOURCES.md). Update the expected stock
names in `TestTheShelfHoldsTheStocksOfTheHills`.
