package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// What holds a face to its tower, drawn where it can be seen: bolts down
// the front of the posts between its windows, which are the caps of a
// curtain wall's mullions and the piers of a stone, concrete or
// vaporwave face, and anchor plates on the bands across it, each plate
// with its own four bolts. They are steel, so the world's rust finds
// them.
//
// A face's fasteners are cut into tiles, a storey high and a few windows
// wide, and each tile is a node. The leveller gives a tile its own level
// by how much of the film it spans, which is less than a band of storeys
// spans, so a tile makes its fasteners only when the eye is close enough
// to count them, and a face further off never makes any.
//
// A face drawn
// below the demo level has none, so the lit panes that level keeps are
// as they were.

// fixing is where a skin is held, the fronts of its members measured
// from the plane of its glass, outward: of the posts between its windows
// and of the bands across it, and how high a band is. A skin whose posts
// or bands are not known has none of the fasteners that stand on them.
type fixing struct {
	post, plate, band float64
}

// fixingOf is where a face's skin is held.
func (job faceJob) fixingOf() fixing {
	drawing := job.drawing
	switch job.skin {
	case skinCurtain, skinMirror:
		return fixing{post: drawing.fixed("cw-mullion-proud") +
			drawing.fixed("cw-cap-proud"),
			plate: job.pane - job.recess,
			band: drawing.fixed("cw-spandrel-height") -
				drawing.fixed("cw-transom-height")}
	case skinGranite:
		if job.style != styleRibbon {
			return fixing{post: drawing.fixed("granite-pier-proud")}
		}
	case skinConcrete:
		return fixing{post: drawing.fixed("concrete-pier-proud"),
			plate: drawing.fixed("concrete-slab-proud"),
			band:  drawing.fixed("concrete-slab-thick")}
	case skinVaporwave:
		return fixing{post: drawing.fixed("vapor-band-proud"),
			plate: drawing.fixed("vapor-band-proud"),
			band:  drawing.fixed("vapor-band-height")}
	}
	return fixing{}
}

// fastenerSizes are the sizes of a face's fasteners, in metres: a bolt's
// head, how far in from a plate's edge it stands, how far it stands out
// of what it holds, how far a plate stands out of its band, and the
// width and height of an anchor plate.
type fastenerSizes struct {
	bolt, inset, bolted, proud, plateWide, plateHigh float64
}

// sized is the sizes of this tower's fasteners.
func (job faceJob) sized() fastenerSizes {
	drawing := job.drawing
	return fastenerSizes{bolt: drawing.fixed("bolt-head"),
		inset:     drawing.fixed("bolt-inset"),
		bolted:    drawing.fixed("bolt-proud"),
		proud:     drawing.fixed("fastener-proud"),
		plateWide: drawing.fixed("anchor-width"),
		plateHigh: drawing.fixed("anchor-height")}
}

// fastenerCell is one window of a face at one storey: where it starts
// and ends along the face, where its storey starts up it and how tall it
// is, and the number of its column and storey in the span.
type fastenerCell struct {
	left, right, foot, tall float64
	column, index           int
}

// fasteners are a span's tiles of fasteners, each a node a storey high
// and a few windows wide, for every storey that is not the ground floor,
// which has its own dressing, nor a plant floor. A tile draws nothing
// below the demo level, and a skin whose fixings are not known has none.
func (job faceJob) fasteners(low, pitch float64, columns int) []placedElement {
	fix := job.fixingOf()
	if fix == (fixing{}) {
		return nil
	}
	every := max(1, job.drawing.count("fastener-tile", place{}))
	var tiles []placedElement
	for index := range job.storeys {
		global := job.firstStorey + index
		if global == 0 || job.mechanical(global) {
			continue
		}
		for from := 0; from < columns; from += every {
			tiles = append(tiles, deferred(job.tile(fix, index, low, pitch,
				from, min(columns, from+every), columns)))
		}
	}
	return tiles
}

// tile is the fasteners of the windows from one column up to another, at
// one storey, as a node that is drawn only at the demo level.
func (job faceJob) tile(fix fixing, index int, low, pitch float64, from, to,
	columns int) *node {
	storey := job.frame.slope / float64(job.storeys)
	foot := float64(index) * storey
	left, right := low+float64(from)*pitch, low+float64(to)*pitch
	return &node{Bounds: job.tileBox(fix, left, right, foot, storey),
		Draw: func(level Detail) ([]placedElement, error) {
			if level != demoLevel {
				return nil, nil
			}
			parts := job.fastenings(fix, low, pitch, from, to, columns, index)
			return parts, job.drawing.trouble
		}}
}

// tileBox is the box a tile fills, in the tower's frame: the windows it
// spans, from the recess of the glass to as far out as a bolt on a post
// or on a plate stands.
func (job faceJob) tileBox(fix fixing, left, right, foot,
	tall float64) maths.Box {
	drawing, frame := job.drawing, job.frame
	reach := max(fix.post, fix.plate+drawing.fixed("fastener-proud")) +
		drawing.fixed("bolt-proud")
	local := maths.Box{Min: maths.V(left, foot, -job.recess),
		Max: maths.V(right, foot+tall, reach)}
	return corners(local, func(point maths.Vec) maths.Vec {
		return frame.at(point.X, point.Y, point.Z).Origin
	})
}

// fastenings are what holds the windows of a tile, from one column up to
// another at one storey: bolts down the posts at the left of each, and at
// the right of the last of a span, and an anchor plate in some of them.
func (job faceJob) fastenings(fix fixing, low, pitch float64, from, to,
	columns, index int) []placedElement {
	storey := job.frame.slope / float64(job.storeys)
	size, look := job.sized(), job.fastener(index, from)
	var parts []placedElement
	for column := from; column < to; column++ {
		left := low + float64(column)*pitch
		cell := fastenerCell{left: left, right: left + pitch,
			foot: float64(index) * storey, tall: storey, column: column,
			index: index}
		if !job.holds(cell) {
			continue
		}
		parts = append(parts, job.postBolts(cell, fix, size, look,
			column == columns-1)...)
		parts = append(parts, job.anchorPlate(cell, fix, size, look)...)
	}
	return parts
}

// holds is whether a window is wholly on the face, at the top of its
// storey, which is where a leaning face is narrowest.
func (job faceJob) holds(cell fastenerCell) bool {
	half := job.frame.half((float64(cell.index) + 1) / float64(job.storeys))
	return cell.right <= half && cell.left >= -half
}

// fastener is the metal of a tile's fasteners.
func (job faceJob) fastener(index, column int) *thing.Material {
	drawing := job.drawing
	return paint("fastener steel", drawing.pick("fastener-colour",
		drawing.options("fastener-colour"), job.place(index, column)))
}

// bolt is a bolt's head, its corner at a place along the face and up it,
// standing out of what it holds, which is out of the plane of the glass
// by a height.
func (job faceJob) bolt(name string, size fastenerSizes, look *thing.Material,
	along, up, out float64) placedElement {
	return job.frame.onFace(name, Box{size.bolt, size.bolt, size.bolted},
		look, along, up, out)
}

// postBolts are the bolts down the front of the post at the left of a
// window, and of the post at its right too when it is the last of a span,
// spread up the storey, each centred on its post.
func (job faceJob) postBolts(cell fastenerCell, fix fixing, size fastenerSizes,
	look *thing.Material, last bool) []placedElement {
	if !(fix.post > 0) {
		return nil
	}
	lines := []float64{cell.left}
	if last {
		lines = append(lines, cell.right)
	}
	count := max(1, job.drawing.count("post-bolts", place{}))
	var parts []placedElement
	for _, line := range lines {
		for each := range count {
			up := cell.foot + (float64(each)+0.5)*cell.tall/float64(count)
			parts = append(parts, job.bolt("post bolt", size, look,
				line-size.bolt/2, up-size.bolt/2, fix.post))
		}
	}
	return parts
}

// anchorPlate is the anchor plate of a window, if its skin has bands and
// the window has one: in every so many windows of a storey, staggered
// from one storey to the next, in the middle of the window and of the
// band at the foot of its storey, with a bolt at each corner.
func (job faceJob) anchorPlate(cell fastenerCell, fix fixing,
	size fastenerSizes, look *thing.Material) []placedElement {
	every := max(1, job.drawing.count("anchor-every", place{}))
	if !(fix.band > size.plateHigh) || (cell.column+cell.index)%every != 0 {
		return nil
	}
	left := (cell.left+cell.right)/2 - size.plateWide/2
	up := cell.foot + (fix.band-size.plateHigh)/2
	parts := []placedElement{job.frame.onFace("anchor plate", Box{size.plateWide,
		size.plateHigh, size.proud}, look, left, up, fix.plate)}
	for _, side := range [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		along := left + size.inset + side[0]*(size.plateWide-2*size.inset-
			size.bolt)
		rise := up + size.inset + side[1]*(size.plateHigh-2*size.inset-
			size.bolt)
		parts = append(parts, job.bolt("anchor bolt", size, look, along,
			rise, fix.plate+size.proud))
	}
	return parts
}
