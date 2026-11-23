package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The salts of the draws a face's cyberpunk layer makes from one uniform
// stream, so that where a thing stands, which span it is on, how high it
// stands, and whether a cell is lit are four draws and not one.
const (
	spotAlong = 1 + iota
	spotUp
	spotSpan
	// spotCell is the first salt of the cells of a sign, which a later
	// time of day adds its step to for a screen, so that its pattern is
	// another at every step.
	spotCell = 100
)

// squaresPerCount is the area, in square metres, that the density of
// greebles counts a face in.
const squaresPerCount = 100

// cyberFace is the cyberpunk layer on one face, over its skin: a light
// band up its corner, greebles stuck to its wall, and stacks of signs and
// screens, whose pattern changes with the time. At the demo level the
// finer greebles are on it too.
func (job faceJob) cyberFace() []placedElement {
	parts := job.cornerBand()
	parts = append(parts, job.greebles()...)
	if job.demo {
		parts = append(parts, job.fineGreebles()...)
	}
	return append(parts, job.signs()...)
}

// neon is the material of a lit neon tube of a colour: dark in itself,
// and glowing.
func (job faceJob) neon(one option) *thing.Material {
	return &thing.Material{Name: "cities neon " + one.Name,
		Colour: maths.Grey(0.05), Glow: colourOf(one).Scale(
			job.drawing.fixed("cyber-neon-glow"))}
}

// cornerBand is the band of light up the corner at the start of a face,
// the same colour all the way up the shaft, if no other part of the plan
// covers that corner. It leans with the face.
func (job faceJob) cornerBand() []placedElement {
	frame, drawing := job.frame, job.drawing
	if len(frame.spans) == 0 ||
		frame.spans[0][0] > -frame.length/2+job.margin {
		return nil
	}
	one := drawing.pick("cyber-neon", drawing.options("cyber-neon"),
		place{Bay: job.shaft})
	return []placedElement{job.strip("corner band", -frame.length/2, 0,
		-frame.topLength/2, frame.slope, drawing.fixed("cyber-band-width"),
		job.recess+drawing.fixed("cyber-band-proud"), job.neon(one))}
}

// strip is a flat bar from a place on the face to another, each along the
// face and up its slope, as wide across its run as given and as deep as
// given from the body, standing on it.
func (job faceJob) strip(name string, fromAlong, fromUp, toAlong, toUp,
	wide, deep float64, look *thing.Material) placedElement {
	frame := job.frame
	run := frame.along.Scale(toAlong - fromAlong).Add(
		frame.up.Scale(toUp - fromUp))
	along := run.Unit()
	rise := frame.out.Cross(along).Unit()
	origin := frame.at(fromAlong, fromUp, -job.recess).Origin.Sub(
		rise.Scale(wide / 2))
	return placedElement{Element: solid{Name: name, Form: Box{run.Len(), wide,
		deep}, Material: look}, Place: maths.NewSubspace(origin,
		maths.Orientation{X: along, Y: rise, Z: frame.out}, 0)}
}

// within is whether a thing of a width and a height, its foot at a place
// up the face and its middle at a place along it, is wholly on the face:
// its foot not under the face's foot, nor its top over the face's top.
func (job faceJob) within(along, up, wide, tall float64) bool {
	frame := job.frame
	return up >= 0 && up+tall <= frame.slope && math.Abs(along)+wide/2 <=
		frame.half((up+tall)/frame.slope)
}

// somewhere is a place on one of the stretches of the face that nothing
// covers, drawn for a place: along the face and up it.
func (job faceJob) somewhere(at place) (along, up float64, ok bool) {
	frame, drawing := job.frame, job.drawing
	if len(frame.spans) == 0 {
		return 0, 0, false
	}
	span := frame.spans[int(drawing.spot(spotSpan, at)*
		float64(len(frame.spans)))]
	along = span[0] + (span[1]-span[0])*drawing.spot(spotAlong, at)
	return along, drawing.spot(spotUp, at) * frame.slope, true
}

// greebles are the things stuck to a wall in a dense city, as many as the
// culture's density gives the face: pipes, vents, utility boxes and
// gantries, each of metal drawn for its place, wherever it fits wholly on
// the face.
func (job faceJob) greebles() []placedElement {
	frame, drawing := job.frame, job.drawing
	area := (frame.length + frame.topLength) / 2 * frame.slope
	count := int(math.Round(area / squaresPerCount *
		drawing.fixed("cyber-greeble-density")))
	var parts []placedElement
	for index := range count {
		at := place{Bay: job.shaft, Storey: job.segment,
			Opening: job.face*faceStride + index}
		along, up, ok := job.somewhere(at)
		if !ok {
			continue
		}
		look := paint("greeble", drawing.pick("cyber-greeble-colour",
			drawing.options("cyber-greeble-colour"), at))
		kind := drawing.pick("cyber-greeble", drawing.options(
			"cyber-greeble"), at).Name
		parts = append(parts, job.greeble(kind, at, along, up, look)...)
	}
	return parts
}

// greeble is one greeble of a kind at a place, or nothing if it does not
// fit on the face there.
func (job faceJob) greeble(kind string, at place, along, up float64,
	look *thing.Material) []placedElement {
	drawing, frame := job.drawing, job.frame
	switch kind {
	case "pipe":
		radius := drawing.fixed("cyber-pipe-radius")
		length := math.Min(drawing.number("cyber-pipe-length", at),
			frame.slope-up)
		round := footprintShape{Plan: planPolygon, Width: 2 * radius,
			Sides: drawing.count("cyber-pipe-sides", at)}.polygon()
		if !job.within(along, up, 2*radius, length) {
			return nil
		}
		return []placedElement{{Element: solid{Name: "pipe", Form: prism{
			Bottom: polygon{round}, Top: polygon{round}, Height: length},
			Material: look}, Place: frame.at(along, up, radius/2)}}
	case "vent":
		wide, tall := drawing.fixed("cyber-vent-width"),
			drawing.fixed("cyber-vent-height")
		if !job.within(along, up, wide, tall) {
			return nil
		}
		return []placedElement{frame.onFace("vent", Box{wide, tall, job.recess +
			drawing.fixed("cyber-vent-depth")}, look, along-wide/2, up,
			-job.recess)}
	case "box":
		wide, tall := drawing.number("cyber-box-width", at),
			drawing.number("cyber-box-height", at)
		if !job.within(along, up, wide, tall) {
			return nil
		}
		return []placedElement{frame.onFace("utility box", Box{wide, tall,
			job.recess + drawing.fixed("cyber-box-depth")}, look,
			along-wide/2, up, -job.recess)}
	}
	return job.gantry(along, up, look)
}

// gantry is a catwalk stuck to the wall, a floor and a rail along its
// outer edge, with its middle at a place along the face.
func (job faceJob) gantry(along, up float64,
	look *thing.Material) []placedElement {
	drawing, frame := job.drawing, job.frame
	long, deep := drawing.fixed("cyber-gantry-length"),
		drawing.fixed("cyber-gantry-width")
	thick, rail := drawing.fixed("cyber-gantry-thick"),
		drawing.fixed("cyber-rail-height")
	post := drawing.fixed("cyber-rail-thick")
	if !job.within(along, up, long, thick+rail) {
		return nil
	}
	return []placedElement{frame.onFace("gantry", Box{long, thick, job.recess +
		deep}, look, along-long/2, up, -job.recess),
		frame.onFace("gantry rail", Box{long, rail, post}, look,
			along-long/2, up+thick, deep-post)}
}

// signs are a stack of signs and screens up one face, if the culture gives
// the face any, one above the next with a gap between, from the start the
// culture draws, as far up as the face holds them.
func (job faceJob) signs() []placedElement {
	frame, drawing := job.frame, job.drawing
	face := place{Bay: job.shaft, Storey: job.segment,
		Opening: job.face * faceStride}
	if !drawing.chance("cyber-sign-share", face) {
		return nil
	}
	edge, gap := drawing.fixed("cyber-sign-edge"),
		drawing.fixed("cyber-sign-gap")
	up := drawing.number("cyber-sign-start", face) * frame.slope
	var parts []placedElement
	for index := range drawing.count("cyber-sign-count", face) {
		at := face
		at.Opening += index
		kind := drawing.pick("cyber-sign-kind",
			drawing.options("cyber-sign-kind"), at).Name
		wide, tall := drawing.number("cyber-"+kind+"-width", at),
			drawing.number("cyber-"+kind+"-height", at)
		along := (drawing.spot(spotAlong, face) - 0.5) *
			(frame.length - 2*edge - wide)
		if !job.within(along, up, wide, tall) {
			break
		}
		parts = append(parts, job.sign(at, kind, along, up, wide, tall)...)
		up += tall + gap
	}
	return parts
}

// sign is one sign or screen with its middle at a place along the face
// and its foot a place up it: a dark board that is also a light, and on it
// cells, each lit or dark. A sign is one colour and its pattern scrolls up
// at each step of the time; a screen's cells are each a colour of their own,
// and its pattern is another at each step.
func (job faceJob) sign(at place, kind string, along, up, wide,
	tall float64) []placedElement {
	drawing, frame := job.drawing, job.frame
	board := paint("cyber board", drawing.pick("cyber-board-colour",
		drawing.options("cyber-board-colour"), at))
	lamp := &Lamp{Heat: drawing.fixed("cyber-sign-heat"),
		Luminance: drawing.fixed("cyber-sign-luminance")}
	depth := drawing.fixed("cyber-board-depth")
	parts := []placedElement{{Element: solid{Name: "sign board", Form: Box{wide,
		tall, depth}, Material: board, Lamp: lamp},
		Place: frame.at(along-wide/2, up, -job.recess)}}
	colours := drawing.options("cyber-neon")
	fill := drawing.fixed("cyber-" + kind + "-fill")
	step := int(math.Floor(job.seconds / drawing.fixed("cyber-sign-period")))
	cell, gap := drawing.fixed("cyber-cell"), drawing.fixed("cyber-cell-gap")
	proud := drawing.fixed("cyber-cell-proud")
	neons := map[string]*thing.Material{}
	for row := range int(tall / cell) {
		for column := range int(wide / cell) {
			spot, salt := place{Bay: job.shaft, Storey: column,
				Opening: at.Opening*faceStride + row + step}, spotCell
			if kind == "screen" {
				spot.Opening, salt = at.Opening*faceStride+row, spotCell+step
			}
			if !(drawing.spot(salt, spot) < fill) {
				continue
			}
			one := drawing.pick("cyber-neon", colours, at)
			if kind == "screen" {
				one = drawing.pick("cyber-neon", colours, spot)
			}
			if neons[one.Name] == nil {
				neons[one.Name] = job.neon(one)
			}
			parts = append(parts, placedElement{Element: solid{Name: kind + " cell",
				Form:     Box{cell - gap, cell - gap, proud},
				Material: neons[one.Name]}, Place: frame.at(
				along-wide/2+float64(column)*cell+gap/2,
				up+float64(row)*cell+gap/2, -job.recess+depth)})
		}
	}
	return parts
}

// searchlights are the lit heads of the searchlights on the roof of a
// shaft: a dark housing at some corners of its top plan, drawn in from the
// corner, and on each a lens that glows and is a light. No beam is drawn:
// the haze that would show one is the scene's, not the tower's.
func (tower *Tower) searchlights(index int, looks towerLooks) []placedElement {
	drawing, shaft := tower.drawing, tower.Shafts[index]
	_, ends, err := shaft.layer(len(shaft.Segments) - 1)
	drawing.fail(err)
	if err != nil {
		return nil
	}
	corners, roof := ends[0], tower.height(index)
	size, lens := drawing.fixed("cyber-searchlight-size"),
		drawing.fixed("cyber-searchlight-lens")
	glass := &thing.Material{Name: "cities searchlight lens",
		Colour: maths.Grey(0.05), Glow: colourOf(drawing.pick(
			"cyber-searchlight-colour", drawing.options(
				"cyber-searchlight-colour"), place{})).Scale(
			drawing.fixed("cyber-searchlight-glow"))}
	lamp := &Lamp{Heat: drawing.fixed("cyber-sign-heat"),
		Luminance: drawing.fixed("cyber-searchlight-luminance")}
	count := int(drawing.fixed("cyber-searchlights"))
	middle := centroid(corners)
	var parts []placedElement
	for each := range count {
		corner := corners[each*len(corners)/count]
		away := math.Hypot(middle.X-corner.X, middle.Z-corner.Z)
		share := drawing.fixed("cyber-searchlight-inset") / away
		x := corner.X + (middle.X-corner.X)*share
		z := corner.Z + (middle.Z-corner.Z)*share
		parts = append(parts, put("searchlight", Box{size, size, size},
			looks.plant, maths.V(x-size/2, roof, z-size/2)),
			placedElement{Element: solid{Name: "searchlight lens", Form: Box{lens,
				lens, lens}, Material: glass, Lamp: lamp},
				Place: standingAt(maths.V(x-lens/2, roof+size, z-lens/2))})
	}
	return parts
}
