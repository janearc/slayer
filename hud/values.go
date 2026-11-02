package hud

import (
	"fmt"
	"time"
)

// Gauge is a number a hud shows that may change from frame to frame, as
// a thing's motion does: asked at each frame's time.
type Gauge interface {
	At(t time.Duration) float64
}

// Fixed is a gauge that is always its own value.
type Fixed float64

// At is the value, whatever the time.
func (value Fixed) At(time.Duration) float64 { return float64(value) }

// Ramp is a gauge that runs evenly from one value to another: From until
// Start, To from Start and Over on, and the straight line between.
type Ramp struct {
	From, To    float64
	Start, Over time.Duration
}

// At is the ramp's value at t.
func (made Ramp) At(t time.Duration) float64 {
	if made.Over <= 0 {
		if t < made.Start {
			return made.From
		}
		return made.To
	}
	share := float64(t-made.Start) / float64(made.Over)
	share = max(0, min(1, share))
	return made.From + (made.To-made.From)*share
}

// Wording is the words a hud shows that may change from frame to frame.
type Wording interface {
	At(t time.Duration) string
}

// Words is wording that is always its own words.
type Words string

// At is the words, whatever the time.
func (words Words) At(time.Duration) string { return string(words) }

// Count is wording that is a gauge written out by a format, as fmt
// writes a float64: "%.0f%%" for a percentage, "%06.1f" for a clock.
type Count struct {
	Of     Gauge
	Format string
}

// At is the gauge's value at t, written out.
func (made Count) At(t time.Duration) string {
	return fmt.Sprintf(made.Format, made.Of.At(t))
}
