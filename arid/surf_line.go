// This file is what the surf did to a sea, counted for the draw's line.

package arid

// Surfline is what a coast's surf did to a sea, counted at the ground's
// samples, after what its shoreline did.
type Surfline struct {
	Shoreline
	// Calm is whether the sea had no wind or no waves, and so no surf.
	Calm bool
	// Gain is the most of their open height the waves stand at.
	Gain float64
	// Capped is how many samples the depth cut the shoaled waves at, and
	// CappedFrom the deepest of them, in metres.
	Capped     int
	CappedFrom float64
	// Spilling, Plunging and Surging count the waterline's samples by the
	// kind of breaker their slope makes.
	Spilling, Plunging, Surging int
	// RunupLeast and RunupMost are the least and most run-up two in a
	// hundred pass, over the land beside the waterline, in metres.
	RunupLeast, RunupMost float64
	// Swashed is how many samples of land the swash wets.
	Swashed int
	// Whitecaps is the share of the open sea the wind whitens.
	Whitecaps float64
	// Rounds is how many rounds of sweeps each band's bending took.
	Rounds []int
}
