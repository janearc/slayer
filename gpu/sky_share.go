package gpu

// Fast light: the sky's looks, shared.
//
// The exact tracer asks, at every surface point a ray meets, how much
// of the sky the point sees: sixteen looks round its normal, a shadow
// ray along each (trace/skylight.go, the kernel's skylit). It asks at
// every sample, so a pixel of four samples takes sixty-four looks. On
// the cloud sea those shadow rays are about half the frame.
//
// Shared, the sixteen looks are dealt round a pixel's samples as cards
// are. With four samples each takes four and the pixel takes every look
// once; with sixteen each takes one. A pixel still sees the sky along
// all sixteen looks, from points at most a pixel apart, and sends a
// quarter or a sixteenth of the shadow rays.
//
// What it gets wrong: one sample's sky light is rougher, as rough as
// the exact tracer's would be with that many looks, so sky-lit shade is
// a little grainier across a pixel's width. With one sample a pixel
// there is nothing to share and nothing changes.
//
// It is drawn with the lamp store's mode and is off without it.

// skyLooks is the looks the tracer takes of the sky from a surface
// point: trace's own skyLooks, so the two look alike. Defined there.
const skyLooks = 16

// shareSkyFor shares the sky's looks among a pixel's samples for a
// mode that asks it, at so many samples along a pixel's side.
//
// The looks are shared only where they are dealt evenly: where a pixel's
// samples divide sixteen (one, two or four along a side), or sixteen
// divides them (eight, sixteen). At three a pixel would take nine of the
// sixteen in a row, and at five some twice, so there every sample takes
// all sixteen, as the exact tracer does (review of 2026-11-12).
func (encoded *scene) shareSkyFor(mode traversal, side int) {
	if mode == fasterTraversal && encoded.reach != nil &&
		sharesEvenly(side) {
		encoded.reach.shared = max(1, side)
	}
}

// sharesEvenly is whether a pixel of side by side samples takes every
// one of the sky's looks the same number of times when they are dealt.
func sharesEvenly(side int) bool {
	count := side * side
	return side >= 1 && (skyLooks%count == 0 || count%skyLooks == 0)
}
