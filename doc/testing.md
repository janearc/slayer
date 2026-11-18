# Testing Slayer

Start with the smallest check that exercises your change. Slayer includes
quick host tests, real image/audio/device tests, an independent physics
harness and geometry records. Running every package with every draw enabled
is a deliberate expensive gate, not the first command for a contributor.

Complete [build setup](building.md) first.

## Draw tests

Draw tests open devices, stage scenes, render images/audio/movies, use the
source shelf or perform other slow work. Files use `//go:build draw`,
combined with platform tags where needed. Some package-wide GPU suites
have no tests in their plain build. Do not set `-tags=draw` globally: it
enables expensive tests for every Go invocation that inherits it.

## Run a real GPU sample

Metal needs macOS, the Xcode command-line tools and a usable device.
Native queries need their supported ray-tracing API too. This small
example exercises both software and native compositor passes:

```sh
game test -- -tags=draw -count=1 -parallel=1 -timeout=5m -v ./gpu -run '^TestCompositePassTransport$'
```

## Caches

The smoke gallery has its own executable/input cache. `SMOKES_FRESH=1`
requests real smoke work; do this deliberately for expensive scenes.
Film resume and section reuse also save work independently of Go's cache.
Inspect actual trace counters when measuring a speedup. See [caches](caches.md).
