package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Fast light: the lamp store.
//
// The exact tracer, at every step of every ray through a boxed medium,
// asks each lamp two things: is the lamp hidden from here by a thing
// (a shadow ray through the whole scene), and how much of it the
// volumes on the way take (a march of shades steps through every
// volume). The store answers both from numbers worked out once a frame.
//
// It keeps what arrives, not what leaves: for each lamp, the share of
// its light, each channel, that gets to a place past the things and
// through the volumes. Nothing in it knows the eye. The lamp's own
// strength, falloff, cone or beam profile and the medium's phase are
// still worked at the step itself, exactly, by lamp_light.
//
// Two shapes, by the lamp:
//
//   - A lamp with a place or a way (a point, a ball, a distant one, a
//     polar pair) keeps a grid over each boxed medium: one cell's three
//     numbers are the kernel's own blocked and clear from the cell's
//     middle. A step reads it blended between the eight cells round it.
//   - A narrow beam keeps a table along its axis instead. A beam a
//     metre wide in a medium a kilometre across is narrower than any
//     cell, and off its axis its light is nought, so a grid of it would
//     hold noughts. What reaches a point in the beam is what reaches the
//     axis at the same distance along; the profile across is lamp_light's.
//
// What it is not: it is not exact. A shadow's edge inside the medium is
// as soft as a cell is wide, and a thing thinner than a cell casts a
// faint shadow or none. The cells are picked below and the difference
// from the exact picture is measured by reach_test.go.
//
// What it leaves alone: surfaces are lit exactly, as before; a gas, a
// giant, a body's air and clouds, a puff, a wisp, a shell, a ring and a
// wake are lit exactly, as before. A giant's clouds and a body's air and
// clouds are shells far thinner than a cell over their box, and from a
// grid a giant rippled.
// A world with more lamps than are each lit at a point (lampsEvery) is
// refused by name, since the exact tracer picks among those and a store
// would not.

// reachCells is how many cells a lamp's grid has along its medium's
// longest side, the others as many as keep the cells near cubes. Picked
// (fable-team, 2026-11-12) from the cloud sea: see reach_test.go and
// doc/fast-light.md for the measured difference at this and at half.
const reachCells = 128

// reachBeamCells is how many places along a beam's axis its table
// keeps, the first at the beam's start and the last at its reach.
// Picked (fable-team, 2026-11-12): a kilometre of beam a metre apart,
// finer than any boxed medium's own cells.
const reachBeamCells = 1024

// reachChannels is defined: a light is three numbers.
const reachChannels = 3

// reachGrid is one boxed medium that keeps the lamps' grids: which
// volume, its box and cells, and where its first lamp's grid begins
// among the boxes' numbers.
type reachGrid struct {
	index int
	grid  boxed
	first int
}

// reachStore is what a scene keeps for fast light: the media with
// grids, which lamps keep a grid in each (a bit a lamp), which keep a
// table, and where the tables begin among the boxes' numbers.
type reachStore struct {
	grids  []reachGrid
	wide   []int
	narrow []int
	tables int
	// shared is the samples along a pixel's side where the sky's looks
	// are shared among a pixel's samples, and nought where every sample
	// takes them all (sky_share.go)
	shared int
}

// errTooManyLamps is fast light asked of a world whose points pick
// among their lamps.
type errTooManyLamps struct{ lamps int }

// Error says how many lamps there are and how many a store keeps.
func (found errTooManyLamps) Error() string {
	return fmt.Sprintf("gpu: fast light keeps a store for every lamp "+
		"and this world has %d, more than the %d a point lights each "+
		"of; draw it with exact lighting", found.lamps, lampsEvery)
}

// reachNumbers is the kernel's declaration of the store's sizes.
func reachNumbers() string {
	return fmt.Sprintf("constexpr constant int reach_beam_cells = %d;\n"+
		"constexpr constant int sky_looks = %d;\n", reachBeamCells,
		skyLooks)
}

// lampShape is how a lamp's row reads to the store: a flood, which
// keeps nothing, a narrow beam, which keeps a table, or any other,
// which keeps a grid.
type lampShape uint8

const (
	floodShape lampShape = iota
	narrowShape
	wideShape
)

// shapeOfLamp reads a lamp's row as the kernel does: a flood by its
// kind, a beam by its radius.
func shapeOfLamp(row []byte) lampShape {
	number := func(field string, part int) float32 {
		at := fieldOffset(lampStruct, field) + part*wordBytes
		return math.Float32frombits(binary.LittleEndian.Uint32(row[at:]))
	}
	if number("second", 3) == floodKind {
		return floodShape
	}
	if number("extent", 1) > 0 {
		return narrowShape
	}
	return wideShape
}

// lampShapes is which lamps keep a grid and which a table, by their
// place among the lamps.
func (encoded *scene) lampShapes() (wide, narrow []int) {
	size := lampStruct.size()
	for index := range encoded.counts.lamps {
		row := encoded.lamps.bytes[index*size : (index+1)*size]
		switch shapeOfLamp(row) {
		case wideShape:
			wide = append(wide, index)
		case narrowShape:
			narrow = append(narrow, index)
		}
	}
	return wide, narrow
}

// reachDims is a grid's cells for a box at so many along its longest
// side: near cubes, and never fewer than two a side.
func reachDims(grid boxed, cells int) [3]int {
	size := grid.high.Sub(grid.low)
	longest := math.Max(size.X, math.Max(size.Y, size.Z))
	var dims [3]int
	for axis, side := range []float64{size.X, size.Y, size.Z} {
		dims[axis] = max(2, int(math.Ceil(side/longest*float64(cells))))
	}
	return dims
}

// reserveReachAt makes room among the boxes for the lamp store, at a
// longest side of cells, and writes each medium's grid into its
// volume's row. Production gives it reachCells; the resolution study
// varies it. A world with more than lampsEvery lamps, or a store past
// the boxes' byte limit, is refused by name.
func (encoded *scene) reserveReachAt(cells int) error {
	if encoded.counts.lamps > lampsEvery {
		return errTooManyLamps{encoded.counts.lamps}
	}
	store := &reachStore{grids: encoded.reached}
	store.wide, store.narrow = encoded.lampShapes()
	numbers := len(encoded.boxes.bytes) / wordBytes
	for index := range store.grids {
		job := &store.grids[index]
		job.grid.cells = reachDims(job.grid, cells)
		job.first = numbers
		numbers += job.grid.count() * reachChannels * len(store.wide)
	}
	store.tables = numbers
	numbers += reachBeamCells * reachChannels * encoded.counts.lamps
	if numbers > math.MaxInt32/wordBytes {
		return fmt.Errorf("gpu: fast light's store of %d numbers is "+
			"past the volume buffer's limit; draw with exact lighting",
			numbers)
	}
	grown := numbers*wordBytes - len(encoded.boxes.bytes)
	encoded.boxes.bytes = append(encoded.boxes.bytes, make([]byte, grown)...)
	for _, job := range store.grids {
		encoded.writeReachRow(job)
	}
	encoded.reach = store
	return nil
}

// writeReachRow writes a medium's grid, its cells and first number,
// into its volume's lighting field.
func (encoded *scene) writeReachRow(job reachGrid) {
	offset := job.index*volumeStruct.size() +
		fieldOffset(volumeStruct, "lighting")
	for part, value := range append(job.grid.cells[:], job.first) {
		binary.LittleEndian.PutUint32(
			encoded.volumes.bytes[offset+part*wordBytes:], uint32(value))
	}
}

// mask is the lamps that keep a grid, a bit a lamp.
func (store *reachStore) mask() int {
	bits := 0
	for _, lamp := range store.wide {
		bits |= 1 << lamp
	}
	return bits
}

// bytes is how much of the boxes the store takes.
func (store *reachStore) bytes(lamps int) int64 {
	numbers := reachBeamCells * reachChannels * lamps
	for _, job := range store.grids {
		numbers += job.grid.count() * reachChannels * len(store.wide)
	}
	return int64(numbers) * wordBytes
}

// writeReachHeader writes the header's lighting field. For a bake: the
// volume and the lamp it is for. For the trace, reading: in x the
// samples along a pixel's side where the sky's looks are shared, which
// lamps keep grids, and one more than where the tables begin, which is
// what tells the kernel to read the store. A header with no store keeps
// its noughts.
func (encoded *scene) writeReachHeader(volume, lamp int, reading bool) {
	offset := fieldOffset(headerStruct, "lighting")
	words := [fieldWords]int{volume, lamp, 0, 0}
	if reading {
		words = [fieldWords]int{encoded.reach.shared, 0,
			encoded.reach.mask(), encoded.reach.tables + 1}
	}
	for part, value := range words {
		binary.LittleEndian.PutUint32(
			encoded.header.bytes[offset+part*wordBytes:], uint32(value))
	}
}
