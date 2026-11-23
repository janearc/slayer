package skeleton

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
)

var (
	errNoVertebrae = errors.New("skeleton: a column needs vertebrae")
	errRegion      = errors.New(
		"skeleton: a region's count or length is not positive")
)

// Region is one run of vertebrae of a column: its name, which prefixes
// each vertebra's name, how many there are, and the length of its first
// and last vertebra, each relative to the other vertebrae of the column.
// The lengths between run evenly from the first to the last.
type Region struct {
	name        string
	count       int
	first, last float64
}

// NewRegion is count vertebrae named name, from first to last long.
func NewRegion(name string, count int, first, last float64) Region {
	return Region{name: name, count: count, first: first, last: last}
}

// length is the relative length of the region's vertebra at place i,
// counted from zero.
func (region Region) length(i int) float64 {
	if region.count == 1 {
		return region.first
	}
	share := float64(i) / float64(region.count-1)
	return region.first + share*(region.last-region.first)
}

// Path is where a column runs: the point at a share of the way from its
// front (zero) to its back (one), and the half thickness of a vertebral
// body there.
type Path func(share float64) (maths.Vec, float64)

// columnLengths is the relative length of every vertebra of regions, in
// order, or the region that cannot give one.
func columnLengths(regions []Region) ([]float64, error) {
	lengths := []float64{}
	for _, region := range regions {
		if region.count <= 0 || !(region.first > 0 && region.last > 0) {
			return nil, fmt.Errorf("%w: %s", errRegion, region.name)
		}
		for i := 0; i < region.count; i++ {
			lengths = append(lengths, region.length(i))
		}
	}
	if len(lengths) == 0 {
		return nil, errNoVertebrae
	}
	return lengths, nil
}

// columnShares are the shares of the path at which each vertebra
// starts, and one more at which the last ends.
func columnShares(lengths []float64) []float64 {
	total := 0.
	for _, length := range lengths {
		total += length
	}
	shares := []float64{0}
	run := 0.
	for _, length := range lengths {
		run += length
		shares = append(shares, run/total)
	}
	return shares
}

// Column adds the vertebrae of regions along path, front to back, each
// jointed to the one before it and the first to parent. The vertebra
// numbered n (from one) of a region named R is named R followed by n,
// as C1. Each vertebra takes its share of the path by its length.
func (skeleton *Skeleton) Column(parent string, regions []Region,
	path Path,
) error {
	lengths, err := columnLengths(regions)
	if err != nil {
		return err
	}
	shares := columnShares(lengths)
	place, above := 0, parent
	for _, region := range regions {
		for n := 1; n <= region.count; n++ {
			head, thickHead := path(shares[place])
			tail, thickTail := path(shares[place+1])
			name := fmt.Sprintf("%s%d", region.name, n)
			err := skeleton.Add(name, above, head, tail, thickHead, thickTail)
			if err != nil {
				return err
			}
			place, above = place+1, name
		}
	}
	return nil
}
