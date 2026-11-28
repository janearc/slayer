# Testing Slayer

complete [build setup](building.md), then choose the check that
exercises your change.

| command | scope |
|---|---|
| `game test -- ./job ./camera` | focused non-draw tests |
| `game check` | formatting, vet and non-draw tests |
| `game run test-draw` | draw-tier tests, one package at a time |
| `game run test-all` | both tiers, with real draws |

the [physics harness](../burning-trousers/README.md) checks the
renderer against independent expectations.

## Draw tests

the draw tier includes device tests, image and audio output, and
scenes that need the source shelf. pass `-tags=draw` to the selected
`game test` command. some suites also need a supported platform.

## Run a real GPU sample

Metal needs macOS, the Xcode command-line tools and a usable
device. native queries need Metal ray-tracing support. this
compositor test exercises both paths.

```sh
game test -- -tags=draw -count=1 -parallel=1 -timeout=5m -v ./gpu -run '^TestCompositePassTransport$'
```

## Caches

The smoke gallery has its own executable/input cache. `SMOKES_FRESH=1`
requests real smoke work; do this deliberately for expensive scenes.
Film resume and section reuse also save work independently of Go's cache.
Inspect actual trace counters when measuring a speedup. See [caches](caches.md).
