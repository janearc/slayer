package searching

// lanternShellClock choreographs eight groups without a mutable playhead.
// All turn, settle and hold; groups wake from inner to outer, share a run,
// settle together, hold, then reverse. Returning integrated motion keeps
// position continuous at each start and stop and makes seeking repeatable.
func lanternShellClock(progress float64, group int) float64 {
	initial := shellRampArea(progress, 0, .03) - shellRampArea(progress, .07, .05)
	start := .32 + float64(group)*.046
	grouped := shellRampArea(progress, start, .065) - shellRampArea(progress, .73, .06)
	reverse := shellRampArea(progress, .91, .12)
	// A three-second thinking hold precedes the slower unlock. The later
	// hold and tentative reverse make recognition deliberate, not a loop.
	return 32 * (initial + grouped - reverse)
}

// shellRampArea is the integral of a smoothstep velocity entrance.
func shellRampArea(progress, start, width float64) float64 {
	elapsed := progress - start
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= width {
		return elapsed - width/2
	}
	share := elapsed / width
	return width * (share*share*share - share*share*share*share/2)
}
