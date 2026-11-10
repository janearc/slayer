package biome

import (
	"crypto/sha256"
	"encoding/binary"
)

// ID is the full spatial identity, not an index in the current visible list.
// Seed hashes can collide; callers keep this tuple when identity matters.
type ID struct {
	Region      string
	Seed        uint64
	Column, Row int64
	Ordinal     uint32
}

// Candidate is a fixed root and recipe before terrain/habitat/clearance
// queries.
type Candidate struct {
	ID          ID
	Recipe      string
	East, South float64
	Seed        uint64
	Turn        float64 // turns about local up, in [0,1)
}

// fingerprint uses a fixed-width encoding after the region's hashed namespace.
// Domains distinguish cell counts, candidate properties and orientation.
func (region *Region) fingerprint(column, row int64, ordinal, domain uint32) [sha256.Size]byte {
	var input [sha256.Size + 8 + 8 + 4 + 4]byte
	copy(input[:], region.prefix[:])
	binary.LittleEndian.PutUint64(input[32:40], uint64(column))
	binary.LittleEndian.PutUint64(input[40:48], uint64(row))
	binary.LittleEndian.PutUint32(input[48:52], ordinal)
	binary.LittleEndian.PutUint32(input[52:56], domain)
	return sha256.Sum256(input[:])
}

// unit takes 53 random bits, matching binary64's integral significand capacity.
func unit(bytes []byte) float64 {
	return float64(binary.LittleEndian.Uint64(bytes)>>11) * 0x1p-53
}

// count uses randomized rounding of the mean, not a Poisson growth process.
func (region *Region) count(column, row int64) uint32 {
	draw := region.fingerprint(column, row, 0, 0)
	count := uint32(region.mean)
	if unit(draw[:8]) < region.mean-float64(count) {
		count++
	}
	return count
}

// recipe maps a stable draw to the authored mixture's relative weights.
func (region *Region) recipe(draw float64) string {
	target, cumulative := draw*region.weight, 0.0
	for _, member := range region.definition.Species {
		cumulative += member.Weight
		if target < cumulative {
			return member.Recipe
		}
	}
	return region.definition.Species[len(region.definition.Species)-1].Recipe
}

// candidate keeps draws independent of query boundaries and enumeration order.
func (region *Region) candidate(column, row int64, ordinal uint32) Candidate {
	draw := region.fingerprint(column, row, ordinal, 1)
	turn := region.fingerprint(column, row, ordinal, 2)
	return Candidate{ID: ID{Region: region.definition.Name, Seed: region.definition.Seed,
		Column: column, Row: row, Ordinal: ordinal},
		Recipe: region.recipe(unit(draw[16:24])),
		East:   latticePoint(column, region.cell, unit(draw[:8])),
		South:  latticePoint(row, region.cell, unit(draw[8:16])),
		Seed:   binary.LittleEndian.Uint64(draw[24:32]),
		Turn:   unit(turn[:8])}
}
