package manufacture

// seed is what a thing grows from: the number that makes one spruce
// differ from the next. the same seed is the same thing every time, on
// any machine, and the kernel can draw the same numbers from it.
// doc/manufactory.md says why a thing is grown and not packed by hand.
//
// made only by sow. an empty seed was never sown, and the doors refuse
// it, so a thing cannot be grown from a seed somebody forgot to give.
type seed struct {
	number uint64
	sown   bool
}

// sow is the seed of a number. zero is a seed like any other.
func sow(number uint64) seed {
	return seed{number: number, sown: true}
}

// made is whether the seed was sown.
func (from seed) made() bool { return from.sown }

// draw is a share, from zero up to but not one, that is the same for the same
// seed and the same three places: which part, which of those, and which number
// of it. it is unit, which conifer_proper.go declares and the kernel mirrors,
// so the arithmetic is written once.
func (from seed) draw(part, index, which int) float64 {
	return unit(from.number, part, index, which)
}

// swing is a draw stretched to either side of zero, from minus one up to but
// not one: how far a thing strays from where it would be.
func (from seed) swing(part, index, which int) float64 {
	return across(from.number, part, index, which)
}

// sprouts is the place a sprout is drawn from: the one no arrangement
// draws from for anything else.
const sprouts = 3

// sprout is the seed of a part carried at a row and an index: a seed
// of its own, drawn from its carrier's, so two fronds of one fern do
// not grow the same leaflets. it is a draw, and so unit, widened back
// to the whole number a draw was made from.
func (from seed) sprout(row, index int) seed {
	return sow(uint64(from.draw(row, index, sprouts) * drawSteps))
}

// pieces is the place a piece's seed is drawn from: like sprouts, one
// no arrangement draws from.
const pieces = 4

// onPiece is the seed that what a stem's piece carries is placed and grown
// from: the seed itself for the first piece, so a stem of one piece draws
// exactly as it always did and a spruce stays ninja's, and a seed of its own
// for each piece after, so two pieces of one stem do not carry the same parts.
func (from seed) onPiece(piece int) seed {
	if piece == 0 {
		return from
	}
	return sow(uint64(from.draw(piece, 0, pieces) * drawSteps))
}

// drawSteps is how many different shares a draw can be: unit keeps 53
// bits, as many as a float64 holds exactly.
const drawSteps = 1 << 53
