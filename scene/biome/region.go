// Package biome distributes recipe candidates on a fixed spatial lattice.
// It supplies authored scenery populations, not a model of plant ecology.
package biome

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/celestial/measure"
)

// Rect is a half-open rectangle in a fixed east/south metre coordinate system.
// Its origin must belong to the source/world, not the current render crop.
type Rect struct{ MinEast, MinSouth, MaxEast, MaxSouth float64 }

// Species selects a named construction recipe with a positive relative weight.
// A recipe may describe vegetation, stones or other ecosystem scenery.
type Species struct {
	Recipe string
	Weight float64
}

// Definition fixes identity, footprint, density and recipe mixture for a
// region. PerHectare counts candidates; later habitat/clearance filters may
// reject them. Identity is stable only for a fixed definition, including
// mixture order.
type Definition struct {
	Name       string
	Seed       uint64
	Bounds     Rect
	Cell       measure.Length
	PerHectare float64
	Species    []Species
}

// Region retains an immutable mixture and fixed source-coordinate definition.
type Region struct {
	definition Definition
	prefix     [sha256.Size]byte
	cell, mean float64
	weight     float64
}

// hectareSide is defined by one hectare being a 100 m by 100 m area.
const hectareSide = 100

// New validates a region without enumerating its cells or allocating geometry.
func New(definition Definition) (*Region, error) {
	cell := definition.Cell.InMetres()
	if strings.TrimSpace(definition.Name) == "" || !definition.Bounds.valid() ||
		!finite(cell) || cell <= 0 || !finite(definition.PerHectare) || definition.PerHectare < 0 {
		return nil, fmt.Errorf("biome: invalid region identity, footprint or density")
	}
	mean := definition.PerHectare * (cell / hectareSide) * (cell / hectareSide)
	if !finite(mean) || mean >= math.MaxUint32 {
		return nil, fmt.Errorf("biome: density exceeds the per-cell identity range")
	}
	weight, err := mixtureWeight(definition.Species)
	if err != nil {
		return nil, err
	}
	definition.Species = append([]Species(nil), definition.Species...)
	region := &Region{definition: definition, cell: cell, mean: mean, weight: weight}
	region.prefix = regionPrefix(definition.Name, definition.Seed)
	return region, nil
}

// mixtureWeight refuses empty names, duplicates, nonpositive weights and
// overflow.
func mixtureWeight(species []Species) (float64, error) {
	weight := 0.0
	names := map[string]bool{}
	for _, member := range species {
		if strings.TrimSpace(member.Recipe) == "" || names[member.Recipe] ||
			!finite(member.Weight) || member.Weight <= 0 {
			return 0, fmt.Errorf("biome: invalid or duplicate mixture recipe %q", member.Recipe)
		}
		names[member.Recipe] = true
		weight += member.Weight
	}
	if len(species) == 0 || !finite(weight) {
		return 0, fmt.Errorf("biome: empty or overflowing mixture")
	}
	return weight, nil
}

// regionPrefix separates this generator version, region name and scene seed.
func regionPrefix(name string, seed uint64) [sha256.Size]byte {
	bytes := []byte("slayer-biome-v2\x00")
	bytes = binary.LittleEndian.AppendUint64(bytes, seed)
	bytes = binary.LittleEndian.AppendUint64(bytes, uint64(len(name)))
	bytes = append(bytes, name...)
	return sha256.Sum256(bytes)
}

// Bounds returns the region footprint independently of the current view.
func (region *Region) Bounds() Rect { return region.definition.Bounds }

// finite refuses non-finite coordinates before lattice-index conversion.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// valid requires a finite nonempty rectangle; empty query intersections are
// separate.
func (rect Rect) valid() bool {
	return finite(rect.MinEast) && finite(rect.MinSouth) &&
		finite(rect.MaxEast) && finite(rect.MaxSouth) &&
		rect.MinEast < rect.MaxEast && rect.MinSouth < rect.MaxSouth
}

// Contains uses half-open ownership so adjoining queries do not duplicate
// roots.
func (rect Rect) Contains(east, south float64) bool {
	return east >= rect.MinEast && east < rect.MaxEast &&
		south >= rect.MinSouth && south < rect.MaxSouth
}

// intersection clips a query without moving the region's spatial lattice.
func (rect Rect) intersection(other Rect) Rect {
	return Rect{MinEast: max(rect.MinEast, other.MinEast), MinSouth: max(rect.MinSouth, other.MinSouth),
		MaxEast: min(rect.MaxEast, other.MaxEast), MaxSouth: min(rect.MaxSouth, other.MaxSouth)}
}
