package weathering

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// What a writer leaves, said once. A wall is cut into cells Grain long
// and a squash of that tall, and a cell that is marked holds one mark: a
// tag, a stroke that wanders along with strokes down through it; a
// throw-up, a few fat bubbles filled in one paint and outlined; or a
// piece, more bubbles in two paints under one outline. Each is drawn as
// shapes in paint, and none is a letter, so no font is needed.

// the numbers a mark is drawn with
const (
	// tagShare and throwShare are the shares of marks that are tags and
	// throw-ups; the rest are pieces. picked: tags are most of what is
	// on a wall, and pieces the fewest.
	tagShare   = 0.55
	throwShare = 0.35
	// markMiddle is how far up its cell a mark's middle line is, as a
	// share of the cell's height, so that its drips have the room below
	// it. picked: a little over the middle.
	markMiddle = 0.6
	// markMargin is how far in from its cell's ends a mark keeps, as a
	// share of the cell. picked: marks that do not run into each other.
	markMargin = 0.08
	// strokeWidth is how thick a tag's stroke is, as a share of its
	// cell's height. picked: a marker's line.
	strokeWidth = 0.11
	// strokeWave is how far a tag's stroke strays from its middle line,
	// as a share of the cell's height, and strokeTurns how many times at
	// the most it turns along the tag. picked: a hand moving fast.
	strokeWave  = 0.18
	strokeTurns = 3
	// downFewest and downMore: a tag has from downFewest strokes down
	// through it and up to downMore more. picked: a short name's worth.
	downFewest = 3
	downMore   = 4
	// downTall is how far a stroke down reaches either side of the line,
	// as a share of the cell's height, and downLean how far it leans
	// along for each of its height. picked: a writer's slant.
	downTall = 0.28
	downLean = 0.3
	// bubblesFewest and bubblesMore: a throw-up has from bubblesFewest
	// bubbles and up to bubblesMore more; a piece has pieceMore more
	// again. picked: a few letters' worth of shapes.
	bubblesFewest = 2
	bubblesMore   = 3
	pieceMore     = 2
	// bubbleTall is how far a bubble reaches up and down from its middle,
	// as a share of the cell's height, and bubbleLap how much wider than
	// its share of the run a bubble is, so that neighbours overlap.
	// picked: fat letters that touch.
	bubbleTall = 0.34
	bubbleLap  = 1.3
	// bubbleRound is the power of a bubble's outline: two is an ellipse
	// and more is squarer. picked: a fat letter.
	bubbleRound = 2.6
	// bubbleJitter is how far up or down a bubble may sit from the
	// middle line, as a share of the cell's height. picked: a bounce.
	bubbleJitter = 0.06
	// outlineWidth is how thick an outline is, as a share of a bubble.
	// picked: a line read across a street.
	outlineWidth = 0.24
	// splitWave is how far the line between a piece's two paints strays
	// from the middle, as a share of the cell's height, and splitTurns
	// how many times it turns along the piece. picked: a fade's edge.
	splitWave  = 0.08
	splitTurns = 2
	// lineTagShare is the share of tags written in the line's ink and
	// not in a paint. picked: half of tags are black or silver.
	lineTagShare = 0.5
	// edgeSoft is how much of a stroke's width its edge fades over, the
	// soft edge of a spray can. picked: soft, not blurred.
	edgeSoft = 0.3
	// shineLow and shineHigh are the heights up a bubble, as shares of its
	// half height, between which its streak of light runs, and shineAcross
	// how far along it, as a share of its half width, it reaches either
	// side of the bubble's middle, less to the right. picked: the glint a
	// writer paints on a fat letter's top left.
	shineLow    = 0.35
	shineHigh   = 0.6
	shineAcross = 0.7
	// shineTone and shineMix are how pale the streak is and how far from
	// the fill toward it. picked: a highlight, not a white patch.
	shineTone = 1.0
	shineMix  = 0.55
	// dripShare is the share of a mark's bubbles that drip, dripLength how
	// long the longest drip is, as a share of a cell's height, and
	// dripLeast how much of that the shortest keeps. picked: a few runs,
	// some of them long.
	dripShare  = 0.55
	dripLength = 0.26
	dripLeast  = 0.3
	// dripWidth is half the width of a drip, as a share of a cell's
	// height; dripStart is how far up the foot of a bubble it begins, as a
	// share of the bubble's half height, so that it joins the bubble; and
	// dripTip how soft its end is. picked: a thin run that sags.
	dripWidth = 0.025
	dripStart = 0.8
	dripTip   = 0.04
	// holeShare is the share of bubbles with a counter, a hole in the
	// middle that shows the wall, as the round of an O or an A has, and
	// holeSize how big it is, as a share of the bubble's reach. picked:
	// enough that a run of bubbles reads as lettering.
	holeShare = 0.5
	holeSize  = 0.3
	// dripSalt is the step between the numbers a mark draws for its
	// bubbles and those it draws for its drips.
	dripSalt = 16
)

// The kinds of mark, as a set file says them.
const (
	// MarkTag is a name written fast: a stroke along, with strokes down.
	MarkTag = "tag"
	// MarkThrowUp is a few fat bubbles filled in one paint and outlined.
	MarkThrowUp = "throw-up"
	// MarkPiece is more bubbles in two paints under one outline.
	MarkPiece = "piece"
)

// The salts of the numbers drawn for a marked cell, one for each thing a
// mark draws, past the salts the cell's mark and paint already take.
const (
	drawKind = 3 + iota
	drawTurns
	drawPhase
	drawCount
	drawInk
	// drawBubble is the first salt of the bubbles, one for each.
	drawBubble
)

// drawColour and drawSize are the salts of a cell's paint and its size.
const (
	drawColour = 1
	drawSize   = 2
)

// cellDraw is a number drawn for a marked cell, nought to one, by a salt.
type cellDraw func(step uint64) float64

// graffiti is marks at arm's reach on walls: in a cell of the wall, one
// in a share of them marked, the share rising with age, higher by Catch
// where a home is empty, by Sought on what a writer seeks, lower to
// Shunned on what a writer shuns, and Heaven of it where a writer has
// climbed. What a mark covers in its cell is its shape.
func (layer Layer) graffiti(agent Agent, site *Site) float64 {
	if !layer.wall(site, agent.Level) || !(agent.Grain > 0) {
		return 0
	}
	reached := layer.reached(agent, site)
	if !(reached > 0) {
		return 0
	}
	share := agent.Share * reached * favour(agent, site.Name) *
		grown(layer.Age, agent.Settle)
	if layer.empty(site.Point) {
		share *= 1 + agent.Catch
	}
	cell, _, _ := layer.markCell(agent, site)
	if noise.Hash2(layer.Seed+salt(agent), cell[0], cell[1]) >= share {
		return 0
	}
	cover, _ := layer.mark(agent, site)
	return cover
}

// reached is how much of a writer's share reaches a site: all of it at
// arm's reach over the street, Heaven of it where a writer climbs to,
// and none anywhere else.
func (layer Layer) reached(agent Agent, site *Site) float64 {
	height := layer.height(site)
	if height >= agent.Low && height <= agent.High {
		return 1
	}
	if agent.Heaven > 0 && layer.climbed(agent, site) {
		return agent.Heaven
	}
	return 0
}

// climbed is whether a writer can stand within arm's reach under a site,
// above the street, with open sky over them: in front of the wall on a
// roof, a ledge or a catwalk, or behind it on a roof, reaching over a
// parapet.
func (layer Layer) climbed(agent Agent, site *Site) bool {
	inFront := site.Point.Add(site.Normal.Scale(layer.Probe))
	if layer.standing(agent, inFront) {
		return true
	}
	if agent.Behind > 0 {
		behind := site.Point.Sub(site.Normal.Scale(agent.Behind))
		return layer.standing(agent, behind)
	}
	return false
}

// standing is whether a writer stands under a place within arm's reach:
// there is a floor under it between Low and High down, the floor is
// higher than the street's reach, and nothing stands over the place
// within the agent's shelter.
func (layer Layer) standing(agent Agent, from maths.Vec) bool {
	drop, found := layer.Over.Below(from, agent.High)
	if !found || drop < agent.Low {
		return false
	}
	if from.Dot(layer.Up)-layer.Ground-drop <= agent.High {
		return false
	}
	_, _, covered := layer.Over.Above(from, agent.Shelter)
	return !covered
}

// favour is how many times its share a writer marks a surface: Sought
// more again on what they seek, Shunned of it on what they shun, and
// once on anything else. Shunning wins over seeking.
func favour(agent Agent, name string) float64 {
	switch {
	case len(agent.Shun) > 0 && named(name, agent.Shun):
		return agent.Shunned
	case len(agent.Seek) > 0 && named(name, agent.Seek):
		return 1 + agent.Sought
	}
	return 1
}

// markCell is which cell of a wall a site is in, counted along the wall
// and up it, and where in that cell the site is, nought to one each way.
// A cell is Grain long and a squash of that tall.
func (layer Layer) markCell(agent Agent, site *Site) ([2]int64, float64,
	float64) {
	long := math.Max(agent.Grain, tiny)
	along := site.Point.Dot(layer.Up.Cross(site.Normal).Unit()) / long
	up := layer.height(site) * tagSquash / long
	cell := [2]int64{int64(math.Floor(along)), int64(math.Floor(up))}
	return cell, along - math.Floor(along), up - math.Floor(up)
}

// mark is the mark of a site's cell at the site: how much of the site it
// covers, and its paint there. The mark is drawn in its own frame, x from
// nought at its start to one at its end and y up from its middle line in
// cell heights, scaled by the size drawn for it.
func (layer Layer) mark(agent Agent, site *Site) (float64, maths.Light) {
	cell, along, up := layer.markCell(agent, site)
	draw := cellDraw(func(step uint64) float64 {
		return noise.Hash2(layer.Seed+salt(agent)+step, cell[0], cell[1])
	})
	size := tagLeast + (1-tagLeast)*draw(drawSize)
	x := (along-cellMiddle)/(size*(1-2*markMargin)) + cellMiddle
	y := (up - markMiddle) / size
	if x < 0 || x > 1 {
		return 0, maths.Light{}
	}
	fill := mix(agent.Colour, agent.Other, draw(drawColour))
	switch kindOf(agent, draw(drawKind)) {
	case MarkTag:
		ink := fill
		if draw(drawInk) < lineTagShare {
			ink = agent.Line
		}
		return tag(x, y, draw), ink
	case MarkThrowUp:
		bubbles := bubblesFewest + int(draw(drawCount)*bubblesMore)
		return lettered(x, y, bubbles, draw, fill, agent.Line)
	}
	split := splitWave * math.Sin(2*math.Pi*(splitTurns*x+draw(drawPhase)))
	below := agent.Colour
	if y < split {
		below = agent.Other
	}
	pieces := bubblesFewest + bubblesMore + int(draw(drawCount)*pieceMore)
	return lettered(x, y, pieces, draw, below, agent.Line)
}

// kindOf is the kind of mark a cell holds: the agent's own, or if it
// writes all three, one by a number drawn for the cell, tags the most.
func kindOf(agent Agent, drawn float64) string {
	switch {
	case agent.Mark != "":
		return agent.Mark
	case drawn < tagShare:
		return MarkTag
	case drawn < tagShare+throwShare:
		return MarkThrowUp
	}
	return MarkPiece
}

// lettered is how much a run of fat bubbles covers a point of its frame,
// and the paint there: the fill inside a bubble, a lighter streak across
// the top of each where the paint catches the light, the outline round
// it, and the fill again in the drips that run below. A drip is laid over
// whatever it crosses.
func lettered(x, y float64, count int, draw cellDraw,
	fill, line maths.Light) (float64, maths.Light) {
	cover, where := bubbles(x, y, count, draw)
	if run := drips(x, y, count, draw); run > cover {
		return run, fill
	}
	switch where {
	case regionFill:
		return cover, fill
	case regionShine:
		return cover, mix(fill, maths.Grey(shineTone), shineMix)
	}
	return cover, line
}

// tag is how much of a tag covers a point of its frame: a stroke along
// it that wanders up and down, and strokes down through the stroke that
// lean as a hand does.
func tag(x, y float64, draw cellDraw) float64 {
	turns := 1 + draw(drawTurns)*(strokeTurns-1)
	phase := draw(drawPhase)
	line := func(at float64) float64 {
		return strokeWave * math.Sin(2*math.Pi*(turns*at+phase))
	}
	fromLine := math.Abs(y - line(x))
	cover := stroke(fromLine, strokeWidth)
	moreDowns := int(draw(drawCount) * downMore)
	count := float64(downFewest + moreDowns)
	slot := math.Min(math.Floor(x*count), count-1)
	middle := (slot + cellMiddle) / count
	rise := y - line(middle)
	if math.Abs(rise) > downTall {
		return cover
	}
	away := math.Abs((x-middle)*tagSquash - downLean*rise)
	return math.Max(cover, stroke(away, strokeWidth))
}

// stroke is how much of a stroke of a width covers a point a distance
// from its middle line, with the soft edge of a spray can.
func stroke(distance, width float64) float64 {
	return ramp(width-distance, 0, width*edgeSoft)
}

// region is the part of a lettered mark a point is in.
type region int

const (
	// regionLine is the outline round the bubbles.
	regionLine region = iota
	// regionFill is the inside of a bubble.
	regionFill
	// regionShine is the streak of light across the top of a bubble.
	regionShine
)

// bubbles is how much of a run of fat bubbles covers a point of its
// frame, and which part of a mark the point is in. Each bubble is a
// rounded square, set a little up or down, and they overlap their
// neighbours. The nearest bubble to the point decides the part.
func bubbles(x, y float64, count int, draw cellDraw) (float64, region) {
	nearest, nearestBubble := math.Inf(1), 0
	var side, rise float64
	wide := bubbleLap / float64(2*count) * tagSquash
	for each := range count {
		middle := (float64(each) + cellMiddle) / float64(count)
		lift := bubbleJitter * (2*draw(drawBubble+uint64(each)) - 1)
		across := (x - middle) * tagSquash / wide
		up := (y - lift) / bubbleTall
		acrossPower := math.Pow(math.Abs(across), bubbleRound)
		upPower := math.Pow(math.Abs(up), bubbleRound)
		reach := math.Pow(acrossPower+upPower, 1/bubbleRound)
		if reach < nearest {
			nearest, nearestBubble, side, rise = reach, each, across, up
		}
	}
	edge := 1 + outlineWidth
	cover := ramp(edge-nearest, 0, outlineWidth*edgeSoft)
	holed := draw(drawBubble+3*dripSalt+uint64(nearestBubble)) < holeShare
	switch {
	case nearest >= 1:
		return cover, regionLine
	case holed && nearest < holeSize:
		return 0, regionLine
	case holed && nearest < holeSize+outlineWidth:
		return 1, regionLine
	case rise > shineLow && rise < shineHigh && side > -shineAcross &&
		side < shineAcross/2:
		return cover, regionShine
	}
	return cover, regionFill
}

// drips is how much the drips below a run of bubbles cover a point of its
// frame: from the foot of some of the bubbles, a thin run of paint that
// hangs a little way down, as wet paint runs.
func drips(x, y float64, count int, draw cellDraw) float64 {
	var most float64
	for each := range count {
		if draw(drawBubble+dripSalt+uint64(each)) >= dripShare {
			continue
		}
		middle := (float64(each) + cellMiddle) / float64(count)
		length := dripLength * (dripLeast + (1-dripLeast)*
			draw(drawBubble+2*dripSalt+uint64(each)))
		top, tip := -bubbleTall*dripStart, -bubbleTall-length
		along := ramp(y, tip, tip+dripTip) * (1 - ramp(y, top, top+dripTip))
		dripCover := stroke(math.Abs(x-middle)*tagSquash, dripWidth)
		most = math.Max(most, dripCover*along)
	}
	return most
}
