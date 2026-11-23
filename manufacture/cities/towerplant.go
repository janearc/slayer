package cities

import (
	"math"

	"github.com/janearc/slayer/scene/thing"
)

// The keys a face's planting draws by, so that no two things of a balcony
// share a place: a face has a block of keys, each stretch of it a block
// of its own, and in that block each kind of plant has a range of its own.
const (
	plantFace = 100000
	plantSpan = 10000
	// plantHedge, plantVine, plantHang and plantTree start the ranges of
	// the keys of each kind of plant on one stretch of a balcony.
	plantHedge = 0
	plantVine  = 2500
	plantHang  = 5000
	plantTree  = 7500
	// plantChair is the key of the chair of a balcony, among the keys of
	// the hedges, which a balcony does not fill.
	plantChair = 1000
)

// planted is the planting of one face: a balcony every few storeys,
// and a terrace on the ledge where a setback leaves one, each holding
// shrubs and trees, with plants climbing the wall from it and hanging
// below its edge. It gives the parts, and the trees as the catalogue's
// kinds, to be grown by whoever has the catalogue.
func (job faceJob) planted() ([]placedElement, []Tree) {
	drawing, frame := job.drawing, job.frame
	every := int(drawing.fixed("planted-every"))
	var parts []placedElement
	var trees []Tree
	for spanned, span := range frame.spans {
		for index := range job.storeys {
			global := job.firstStorey + index
			terrace := index == 0 && job.ledge > 0
			balcony := (global+1)%every == 0 && !job.mechanical(global)
			half := frame.half((float64(index) + 0.5) / float64(job.storeys))
			left := math.Max(span[0], -half) + job.margin
			right := math.Min(span[1], half) - job.margin
			if !(terrace || balcony) || !(right > left) {
				continue
			}
			key := job.face*plantFace + spanned*plantSpan
			grown, found := job.garden(index, key, left, right, terrace)
			parts, trees = append(parts, grown...), append(trees, found...)
		}
	}
	return parts, trees
}

// garden is one balcony or terrace from left to right at a storey: its
// slab, if it is a balcony, a planter along its edge with shrubs in it,
// climbing and hanging plants, and trees at a spacing.
func (job faceJob) garden(index, key int, left, right float64,
	terrace bool) ([]placedElement, []Tree) {
	drawing, frame := job.drawing, job.frame
	depth := drawing.fixed("planted-depth")
	thick, wall := drawing.fixed("planted-thick"),
		drawing.fixed("planted-wall")
	lip := drawing.fixed("planted-wall-thick")
	foot := float64(index) * frame.slope / float64(job.storeys)
	base := foot + thick
	if terrace {
		depth = math.Min(depth, job.ledge)
		base = foot
	}
	if !(depth > 2*lip) {
		return nil, nil
	}
	at := func(offset int) place {
		return place{Bay: job.shaft, Storey: job.firstStorey + index,
			Opening: key + offset}
	}
	soil := paint("soil", drawing.pick("planted-soil",
		drawing.options("planted-soil"), place{}))
	parts := []placedElement{
		frame.onFace("planter wall", Box{right - left, wall, lip},
			job.looks.podium, left, base, depth-lip),
		frame.onFace("planter soil", Box{right - left, wall - lip,
			depth - lip}, soil, left, base, 0)}
	if !terrace {
		parts = append(parts, frame.onFace("balcony", Box{right - left, thick,
			job.recess + depth}, job.looks.podium, left, foot, -job.recess))
		parts = append(parts, job.hangings(at, left, right, foot, depth)...)
	}
	parts = append(parts, job.shrubs(at, left, right, base+wall-lip, depth)...)
	parts = append(parts, job.chair(at, left, right, base+wall-lip, depth)...)
	parts = append(parts, job.climbers(at, left, right, base)...)
	return parts, job.trees(at, left, right, base+wall-lip, depth)
}

// leaf is the material of green things drawn for a place.
func (job faceJob) leaf(at place) *thing.Material {
	drawing := job.drawing
	return paint("leaf", drawing.pick("planted-leaf",
		drawing.options("planted-leaf"), at))
}

// shrubs are the stretches of shrubs along a planter, each a box of green
// as high as its place draws and as long as the culture says.
func (job faceJob) shrubs(at func(int) place, left, right, top,
	depth float64) []placedElement {
	drawing := job.drawing
	run, lip := drawing.fixed("planted-hedge-run"),
		drawing.fixed("planted-wall-thick")
	var parts []placedElement
	clear := drawing.fixed("planted-clear")
	for each := 0; left+float64(each)*run < right; each++ {
		start := left + float64(each)*run
		length := math.Min(run, right-start)
		if start < clear && start+length > -clear {
			continue
		}
		place := at(plantHedge + each)
		parts = append(parts, job.frame.onFace("hedge", Box{
			length, drawing.number("planted-hedge", place), depth - 2*lip},
			job.leaf(place), start, top, lip))
	}
	return parts
}

// chair is a chair on a balcony, if the culture gives that balcony one,
// in the clear stretch in the middle of the face, off to one side: a
// seat, a back and a leg at each side, in the tower's grid metal, standing
// on the planter's soil.
func (job faceJob) chair(at func(int) place, left, right, top,
	depth float64) []placedElement {
	drawing, frame := job.drawing, job.frame
	place := at(plantChair)
	along := drawing.fixed("planted-chair-at")
	if !drawing.chance("planted-chair", place) || along < left ||
		along > right {
		return nil
	}
	size, seat := drawing.fixed("planted-chair-size"),
		drawing.fixed("planted-chair-seat")
	back, board := drawing.fixed("planted-chair-back"),
		drawing.fixed("planted-chair-board")
	out := depth/2 - size/2
	metal := job.looks.grid
	return []placedElement{
		frame.onFace("chair seat", Box{size, board, size}, metal,
			along-size/2, top+seat, out),
		frame.onFace("chair back", Box{size, back, board}, metal,
			along-size/2, top+seat+board, out),
		frame.onFace("chair leg", Box{board, seat, size}, metal,
			along-size/2, top, out),
		frame.onFace("chair leg", Box{board, seat, size}, metal,
			along+size/2-board, top, out)}
}

// climbers are the plants that climb the wall from a planter, one at each
// spacing along it, as many storeys up as their place draws, as far as the
// face holds them.
func (job faceJob) climbers(at func(int) place, left, right,
	base float64) []placedElement {
	drawing, frame := job.drawing, job.frame
	width, proud := drawing.fixed("planted-vine-width"),
		drawing.fixed("planted-vine-proud")
	spacing := drawing.fixed("planted-vine-spacing")
	storey := frame.slope / float64(job.storeys)
	var parts []placedElement
	for each := 0; left+float64(each)*spacing < right; each++ {
		along := left + float64(each)*spacing
		place := at(plantVine + each)
		tall := math.Min(float64(drawing.count("planted-vine-storeys",
			place))*storey, frame.slope-base)
		if !job.within(along+width/2, base, width, tall) {
			continue
		}
		parts = append(parts, frame.onFace("climbing plant", Box{width, tall,
			job.recess + proud}, job.leaf(place), along, base, -job.recess))
	}
	return parts
}

// hangings are the plants that hang below the edge of a balcony, one at
// each spacing along it, as long as their place draws.
func (job faceJob) hangings(at func(int) place, left, right, foot,
	depth float64) []placedElement {
	drawing := job.drawing
	width, spacing := drawing.fixed("planted-hang-width"),
		drawing.fixed("planted-hang-spacing")
	var parts []placedElement
	for each := 0; left+float64(each)*spacing < right; each++ {
		place := at(plantHang + each)
		long := math.Min(drawing.number("planted-hang-length", place),
			foot+job.beneath)
		parts = append(parts, job.frame.onFace("hanging plant", Box{width,
			long, width}, job.leaf(place), left+float64(each)*spacing,
			foot-long, depth-width))
	}
	return parts
}

// trees are the trees of a planter, one at each spacing along it, of a
// kind the catalogue names, drawn for the place, one of a few of its kind
// by the seed, standing in the middle of the planter on its soil.
func (job faceJob) trees(at func(int) place, left, right, top,
	depth float64) []Tree {
	drawing, frame := job.drawing, job.frame
	spacing := drawing.fixed("planted-tree-spacing")
	seeds := float64(drawing.count("planted-tree-seeds", place{}))
	var found []Tree
	for each := 0; left+(float64(each)+0.5)*spacing < right; each++ {
		place := at(plantTree + each)
		along := left + (float64(each)+0.5)*spacing
		if math.Abs(along) < drawing.fixed("planted-clear") {
			continue
		}
		found = append(found, Tree{Kind: drawing.pick("planted-tree",
			drawing.options("planted-tree"), place).Name,
			Seed:   uint64(drawing.spot(spotAlong, place) * seeds),
			At:     frame.at(along, top, depth/2).Origin,
			Height: drawing.fixed("planted-tree-height")})
	}
	return found
}
