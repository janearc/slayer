package emptyspace

// populate includes the neighbouring samples used by trilinear interpolation.
// A brick spans physical grid cells, whose samples lie at half-cell centres.
func (mask *Mask) populate(fields []Field) {
	for z := range mask.bricks[2] {
		for y := range mask.bricks[1] {
			for x := range mask.bricks[0] {
				if mask.supported(fields, [3]int{x, y, z}) {
					index := (z*mask.bricks[1]+y)*mask.bricks[0] + x
					mask.words[index/64] |= uint64(1) << uint(index%64)
				}
			}
		}
	}
}

// supportRange covers the adjacent interpolation sample on both brick edges.
// Arithmetic never forms an unchecked(start+BrickSide) beyond the last sample.
func (mask *Mask) supportRange(axis, brick int) (int, int) {
	start := brick * mask.options.BrickSide
	last := mask.cells[axis] - 1
	return max(0, start-1), start + min(mask.options.BrickSide, last-start)
}

// supported exits at the first nonzero channel in any contributing field.
func (mask *Mask) supported(fields []Field, brick [3]int) bool {
	x0, x1 := mask.supportRange(0, brick[0])
	y0, y1 := mask.supportRange(1, brick[1])
	z0, z1 := mask.supportRange(2, brick[2])
	for z := z0; z <= z1; z++ {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				index := (z*mask.cells[1]+y)*mask.cells[0] + x
				if supportedCell(fields, index) {
					return true
				}
			}
		}
	}
	return false
}

// supportedCell permits separate density and emissive arrays without joining.
func supportedCell(fields []Field, index int) bool {
	for _, field := range fields {
		first := index * field.Channels
		for _, value := range field.Values[first : first+field.Channels] {
			if value != 0 {
				return true
			}
		}
	}
	return false
}
