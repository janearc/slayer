package cities

import (
	"math"

	"github.com/janearc/slayer/scene/thing"
)

// What the cyberpunk layer adds at the demo level, said once: air
// conditioners with a fan's grille, runs of duct with flanges and an
// elbow down, cables sagging between two hooks, and boxes bolted to the
// wall with a conduit running down from each.
//
// They are steel, so the
// world's rust finds them, and each has a flat underside and top, so the
// world's grime runs down the wall below it. The quick level's pipes,
// vents, boxes and gantries are drawn as they were.

// fineOpening is where the demo greebles of a face start counting their
// places, past any the quick level's greebles take.
const fineOpening = faceStride / 2

// fineGreebles are the demo level's greebles on a face, as many as the
// culture's density gives it, each where it fits wholly on the face.
func (job faceJob) fineGreebles() []placedElement {
	frame, drawing := job.frame, job.drawing
	area := (frame.length + frame.topLength) / 2 * frame.slope
	count := int(math.Round(area / squaresPerCount *
		drawing.fixed("cyber-fine-density")))
	var parts []placedElement
	for index := range count {
		at := place{Bay: job.shaft, Storey: job.segment,
			Opening: job.face*faceStride + fineOpening + index}
		along, up, ok := job.somewhere(at)
		if !ok {
			continue
		}
		look := paint("greeble steel", drawing.pick("cyber-greeble-colour",
			drawing.options("cyber-greeble-colour"), at))
		kind := drawing.pick("cyber-fine-greeble", drawing.options(
			"cyber-fine-greeble"), at).Name
		parts = append(parts, job.fineGreeble(kind, at, along, up, look)...)
	}
	return parts
}

// fineGreeble is one demo greeble of a kind at a place, or nothing if it
// does not fit on the face there.
func (job faceJob) fineGreeble(kind string, at place, along, up float64,
	look *thing.Material) []placedElement {
	switch kind {
	case "air":
		return job.conditioner(along, up, look)
	case "duct":
		return job.duct(at, along, up, look)
	case "cable":
		return job.sagging(at, along, up, look)
	}
	return job.bolted(along, up, look)
}

// conditioner is an air conditioner hung on the wall: a box standing out
// of the face, and on its front the dark disc of its fan's grille.
func (job faceJob) conditioner(along, up float64,
	look *thing.Material) []placedElement {
	drawing, frame := job.drawing, job.frame
	wide, tall := drawing.fixed("cyber-air-width"),
		drawing.fixed("cyber-air-height")
	deep, radius := drawing.fixed("cyber-air-depth"),
		drawing.fixed("cyber-air-fan")
	if !job.within(along, up, wide, tall) {
		return nil
	}
	grille := paint("greeble grille", option{Name: "dark",
		Colour: []float64{fanDark, fanDark, fanDark}})
	round := circle(radius, int(drawing.fixed("cyber-air-sides")))
	return []placedElement{frame.onFace("air conditioner", Box{wide, tall,
		job.recess + deep}, look, along-wide/2, up, -job.recess),
		{Element: solid{Name: "fan grille", Form: prism{Bottom: polygon{
			round}, Top: polygon{round}, Height: drawing.fixed(
			"cyber-air-grille")}, Material: grille},
			Place: frame.facing(along, up+tall/2, deep)}}
}

// fanDark is the light a fan's grille gives back: the dark of the fan
// behind it. picked: near black.
const fanDark = 0.02

// duct is a run of duct along the face with a flange every so often and
// an elbow at its end that runs down the face.
func (job faceJob) duct(at place, along, up float64,
	look *thing.Material) []placedElement {
	drawing, frame := job.drawing, job.frame
	side, long := drawing.fixed("cyber-duct-side"),
		drawing.number("cyber-duct-length", at)
	drop := drawing.fixed("cyber-duct-drop")
	if !job.within(along, up, long+side, side) ||
		!job.within(along+long/2, up-drop, side, drop) {
		return nil
	}
	left := along - long/2
	parts := []placedElement{
		frame.onFace("duct", Box{long, side, job.recess + side}, look, left,
			up, -job.recess),
		frame.onFace("duct", Box{side, drop, job.recess + side}, look,
			left+long-side, up-drop, -job.recess)}
	flange, pitch := drawing.fixed("cyber-duct-flange"),
		drawing.fixed("cyber-duct-pitch")
	for x := left + pitch; x < left+long-side; x += pitch {
		parts = append(parts, frame.onFace("duct flange", Box{flange,
			side + 2*flange, job.recess + side + flange}, look, x, up-flange,
			-job.recess))
	}
	return parts
}

// sagging is a cable hung between two hooks on the face at one height,
// sagging between them in short straight runs.
func (job faceJob) sagging(at place, along, up float64,
	look *thing.Material) []placedElement {
	drawing := job.drawing
	long, sag := drawing.number("cyber-cable-length", at),
		drawing.fixed("cyber-cable-sag")
	runs := int(drawing.fixed("cyber-cable-runs"))
	if !job.within(along, up-sag, long, sag) {
		return nil
	}
	thick, off := drawing.fixed("cyber-cable-thick"),
		drawing.fixed("cyber-cable-off")
	height := func(share float64) float64 {
		return up - sag*4*share*(1-share)
	}
	var parts []placedElement
	for each := range runs {
		from, to := float64(each)/float64(runs), float64(each+1)/float64(runs)
		parts = append(parts, job.strip("hung cable", along-long/2+long*from,
			height(from), along-long/2+long*to, height(to), thick,
			job.recess+off, look))
	}
	return parts
}

// bolted is a box bolted to the wall, a bolt head at each corner of its
// front, and a conduit running down the face from its foot.
func (job faceJob) bolted(along, up float64, look *thing.Material) []placedElement {
	drawing, frame := job.drawing, job.frame
	wide, tall := drawing.fixed("cyber-bolted-width"),
		drawing.fixed("cyber-bolted-height")
	deep, run := drawing.fixed("cyber-bolted-depth"),
		drawing.fixed("cyber-conduit-length")
	conduit := drawing.fixed("cyber-conduit")
	if !job.within(along, up-run, wide, tall+run) {
		return nil
	}
	parts := []placedElement{
		frame.onFace("bolted box", Box{wide, tall, job.recess + deep}, look,
			along-wide/2, up, -job.recess),
		frame.onFace("conduit", Box{conduit, run, job.recess + conduit},
			look, along-conduit/2, up-run, -job.recess)}
	bolt, inset := drawing.fixed("bolt-head"), drawing.fixed("bolt-inset")
	for _, corner := range [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		x := along - wide/2 + inset + corner[0]*(wide-2*inset-bolt)
		y := up + inset + corner[1]*(tall-2*inset-bolt)
		parts = append(parts, frame.onFace("bolt head", Box{bolt, bolt,
			bolt}, look, x, y, deep))
	}
	return parts
}
