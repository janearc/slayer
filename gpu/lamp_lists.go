package gpu

import (
	"encoding/binary"
	"math"
)

// The lists of lamps a kernel loop asks of. Three loops of the kernel
// used to walk every lamp of the world at every point or step and skip
// most of them: the floods' light (flooded), the cones a clear medium's
// single scatter counts, and the beams' march region. Each now walks a
// list of the lamps it does not skip, written here once, in the lamps'
// order, so that the sums it makes are added in the order they were.
//
// The lists lie in the lamps' buffer after the lamp tree and the lamps
// outside it (lamp_tree.go), each lamp's index as a float: the floods,
// then the cones, then the beams. The header's lamp_lists says how many
// of each. A world with none of a kind has none of its list, and a world
// with no lamp of any of them has no lists at all.
//
// This file has no Metal in it. The kernel reads them by lamp_list in
// trace.metal.

// lampLists is the index of every lamp a loop of the kernel asks of, in
// the order of the lamps.
type lampLists struct {
	// floods are the lamps that light from everywhere: the kernel's
	// is_flood.
	floods []int32
	// cones are the lamps that are not floods and have a cone's width,
	// the film.Spots, which a clear medium's single scatter counts.
	cones []int32
	// beams are the collimated sources, whose extent's y is positive.
	beams []int32
}

// lampNumber is the part-th number of a lamp's field, in the lamp's row.
func lampNumber(row []byte, name string, part int) float32 {
	at := fieldOffset(lampStruct, name) + part*wordBytes
	return math.Float32frombits(binary.LittleEndian.Uint32(row[at:]))
}

// lampListsOf is the lists of the lamps in a buffer of their rows, read
// as the kernel reads them: a flood by its kind, a cone by a width above
// nought that is not a flood's, a beam by a radius above nought.
func lampListsOf(rows []byte) lampLists {
	var lists lampLists
	size := lampStruct.size()
	for index := range len(rows) / size {
		row := rows[index*size : (index+1)*size]
		place := int32(index)
		if lampNumber(row, "second", 3) == floodKind {
			lists.floods = append(lists.floods, place)
		} else if lampNumber(row, "toward", 3) > 0 {
			lists.cones = append(lists.cones, place)
		}
		if lampNumber(row, "extent", 1) > 0 {
			lists.beams = append(lists.beams, place)
		}
	}
	return lists
}

// empty is whether no lamp is on any list.
func (lists lampLists) empty() bool {
	return len(lists.floods) == 0 && len(lists.cones) == 0 &&
		len(lists.beams) == 0
}

// bytes is the lists as the kernel reads them: each index as a float,
// the floods, then the cones, then the beams, little-endian as every
// buffer.
func (lists lampLists) bytes() []byte {
	var packed packer
	for _, list := range [][]int32{lists.floods, lists.cones, lists.beams} {
		for _, index := range list {
			packed.word(math.Float32bits(float32(index)))
		}
	}
	return packed.bytes
}
