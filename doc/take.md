# Takes: moving a camera through time

A take carries a camera along a path while the shutter opens for successive
frames. It lives in `take/`, moved from ninjafilm on 2026-10-23 because it
coordinates Slayer's scene and tracer. Slayer imports ninjafilm; the
reverse dependency is deliberately excluded.

## Timing and motion blur

A take combines a path, pace, and duration. The path implements
`thing.Motion` and returns the camera's subspace at a time. Tracks, orbits,
and caller-defined motions all fit that contract.

Frame n opens at `n / rate`. The shutter angle sets the fraction of that
frame interval spent open. Glimpses sample the open interval at evenly
spaced instants while the camera keeps moving.

Slayer renders each glimpse. A ninjafilm roll integrates their light and
develops the frame, producing motion blur. Frame times derive directly
from the rate rather than accumulating per-frame delays, avoiding drift.

## API

- **`Cranked(rate, shutter, glimpses)`:** Result: Pace; Validation: Positive
  finite rate and shutter; shutter at most one turn; at least one glimpse

- **`Take(path, pace, length)`:** Result: Take; Validation: Non-nil path,
  constructed pace, positive duration

- **`take.Frames()`:** Result: Number of complete frames; Validation: No error

- **`take.Glimpses(number)`:** Result: Samples for a frame; Validation: No
  error

- **`glimpse.At()`, `glimpse.Eye()`:** Result: Scene time and camera
  transform; Validation: No error


## Choosing a pace

- **rate:** Meaning: Frames per second; Historical reference: Hand-cranked
  film often ran near 16, with roughly 10-26 possible

- **shutter:** Meaning: Opening angle, in radians; Historical reference: 170
  degrees: `170*maths.Degree`

- **glimpses:** Meaning: Renders per opening; Historical reference: Choose for
  motion and rendering cost


Every glimpse costs a render. One freezes motion regardless of speed;
three or more begin to capture blur, though fast motion may need more.
A stationary camera and scene need only one.

Time starts at zero, when the first frame opens. Use each glimpse's `At()`
for both scene evaluation and camera placement, so moving objects and camera
share the same clock.

## Example

`take/smoke_test.go` films two seconds of a ball on grey ground using every
stock. `cmd/hillside` uses a one-frame take to sample its rig's shutter.
The test's structure is:

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

- **`take: a camera is cranked at a rate above nought, ...`:** Cause: Invalid
  rate, shutter, or glimpse count; Check: Shutter uses radians and is at most
  one full turn

- **`take: a take has a path, a pace from Cranked, and a length above
  nought`:** Cause: Nil path, invalid pace, or nonpositive duration; Check:
  Handle `Cranked` errors before constructing the take
