# quickstart

## i'm sorry

there isn't really a quick way to get started with slayer. it's just not how
slayer is built.

## the least explanation possible

define your environment:

```sh
# where will slayer live
$ export SROOT=${HOME}/slayer
$ mkdir -p ${SROOT}/{shelf,render/{cache,frames,animation}}
# where your supporting info is
$ export SLAYER_SHELF=${SROOT}/shelf
# where the cache for your renders will live
$ export SLAYER_CACHE=${SROOT}/render/cache
```

install `game` version 0.5 or later

```sh
$ go install github.com/janearc/game@latest
```

get your sources (this is necessary because slayer uses derived data for
everything; without this data, slayer doesn't know how the world works and is
afraid to use data and equations it can't verify).

```sh
$ go run ./cmd/sources verify
# ... downloading ensues
```

get some binaries.

```sh
# builds slayer
$ game build
# runs the test suite (this can take quite a while)
$ game check
```

draw something.

```sh
# verifies something can be drawn
$ ./bin/slayer assess doc/examples/jobs/first-picture.yaml
# draws the thing, this one is a clay ball
$ ./bin/slayer draw doc/examples/jobs/first-picture.yaml --out ${SROOT}/render/frames/first-picture.png
# very simple set with a basic house, some lamps, and a planet
$ ./bin/slayer draw sets/shoebox.yaml --out ${SROOT}/render/frames/shoebox.png
```

you can get a list of scenes slayer knows about with `slayer list scenes`.
