# Takes: moving a camera through time

A take carries a camera along a path while the shutter opens for successive
frames. It lives in `take/`, moved from ninjafilm on 2026-10-23 because it
coordinates Slayer’s scene and tracer. Slayer imports ninjafilm; the
reverse dependency is deliberately excluded.

## Timing and motion blur

A take combines a path, pace, and duration. The path implements
`thing.Motion` and returns the camera’s subspace at a time. Tracks, orbits,
and caller-defined motions all fit that contract.

Frame n opens at `n / rate`. The shutter angle sets the fraction of that
frame interval spent open. Glimpses sample the open interval at evenly
spaced instants while the camera keeps moving.

Slayer renders each glimpse. A ninjafilm roll integrates their light and
develops the frame, producing motion blur. Frame times derive directly
from the rate rather than accumulating per-frame delays, avoiding drift.

## API

| Call | Result | Validation |
|---|---|---|
| `Cranked(rate, shutter, glimpses)` | Pace | Positive finite rate and shutter; shutter at most one turn; at least one glimpse |
| `Take(path, pace, length)` | Take | Non-nil path, constructed pace, positive duration |
| `take.Frames()` | Number of complete frames | No error |
| `take.Glimpses(number)` | Samples for a frame | No error |
| `glimpse.At()`, `glimpse.Eye()` | Scene time and camera transform | No error |

## Choosing a pace

| Argument | Meaning | Historical reference |
|---|---|---|
| rate | Frames per second | Hand-cranked film often ran near 16, with roughly 10–26 possible |
| shutter | Opening angle, in radians | 170 degrees: `170*maths.Degree` |
| glimpses | Renders per opening | Choose for motion and rendering cost |

Every glimpse costs a render. One freezes motion regardless of speed;
three or more begin to capture blur, though fast motion may need more.
A stationary camera and scene need only one.

Time starts at zero, when the first frame opens. Use each glimpse’s `At()`
for both scene evaluation and camera placement, so moving objects and camera
share the same clock.

## Example

`take/smoke_test.go` films two seconds of a ball on grey ground using every
stock. `cmd/hillside` uses a one-frame take to sample its rig’s shutter.
The test’s structure is:

```go
roll, err := ninjafilm.Roll(stock, 1921)
pace, err := take.Cranked(16, 170*maths.Degree, 3)
shot, err := take.Take(path, pace, 2*time.Second)
for number := range shot.Frames() {
    for _, glimpse := range shot.Glimpses(number) {
        // Render the world at glimpse.At() from glimpse.Eye().
    }
    // Develop this frame with roll.Print(number, glimpses).
}
```

Error handling is omitted in this outline. Check each error before using
the resulting value.

## Errors

| Message prefix | Cause | Check |
|---|---|---|
| `take: a camera is cranked at a rate above nought, ...` | Invalid rate, shutter, or glimpse count | Shutter uses radians and is at most one full turn |
| `take: a take has a path, a pace from Cranked, and a length above nought` | Nil path, invalid pace, or nonpositive duration | Handle `Cranked` errors before constructing the take |
