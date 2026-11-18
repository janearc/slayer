package manufacture

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// ClusterPattern selects offspring displacement in a Neyman–Scott process.
type ClusterPattern string

const (
	ThomasGaussian  ClusterPattern = "thomas-gaussian"
	NeymanScottDisk ClusterPattern = "neyman-scott-disk"
)

// NeymanScottSpec defines Poisson parents in a finite rectangular window,
// with Poisson offspring per parent. Parent intensity is 1/ParentSpacing².
// Gaussian Spread is standard deviation per axis; disk Spread is radius.
// Offspring are not clipped to the parent window. This finite-parent model
// does not claim the infinite-plane stationary process or a road network.
type NeymanScottSpec struct {
	Pattern                             ClusterPattern
	Width, Depth, ParentSpacing, Spread measure.Length
	MeanOffspring                       float64
}

// ClusterPoint retains an independent seed, parent identity and metre position.
type ClusterPoint struct {
	At           maths.Vec
	Seed, Parent uint64
}

// ClusterSample owns genuine unconstrained cluster parents and offspring.
type ClusterSample struct{ parents, offspring []ClusterPoint }

// Parents returns the Poisson parent points in their stable sample order.
func (sample ClusterSample) Parents() []ClusterPoint { return slices.Clone(sample.parents) }

// Offspring returns unbounded offspring without road/lot conditioning.
func (sample ClusterSample) Offspring() []ClusterPoint { return slices.Clone(sample.offspring) }

// check bounds expected populations before any random allocation.
func (spec NeymanScottSpec) check() error {
	if spec.Pattern != ThomasGaussian && spec.Pattern != NeymanScottDisk {
		return fmt.Errorf("manufacture: unknown Neyman–Scott offspring rule")
	}
	for _, length := range []measure.Length{spec.Width, spec.Depth, spec.ParentSpacing, spec.Spread} {
		if !positive(length.InMetres()) || length.InMetres() > 1e6 {
			return fmt.Errorf("manufacture: cluster dimensions need finite positive metres")
		}
	}
	expected := spec.Width.InMetres() * spec.Depth.InMetres() / math.Pow(spec.ParentSpacing.InMetres(), 2)
	if !positive(expected) || !positive(spec.MeanOffspring) || spec.MeanOffspring > 2000 || expected > 4096 || expected*spec.MeanOffspring > 100_000 {
		return fmt.Errorf("manufacture: cluster process exceeds finite population budget")
	}
	return nil
}

// SampleNeymanScott draws actual Poisson parents and offspring. Thomas uses
// independent Gaussian axis displacements; the disk alternative is uniform
// in disk area. Evaluation neither reads a clock nor shares a random stream.
func SampleNeymanScott(spec NeymanScottSpec, from seed) (ClusterSample, error) {
	if err := spec.check(); err != nil {
		return ClusterSample{}, err
	}
	if !from.made() {
		return ClusterSample{}, errGood
	}
	mean := spec.Width.InMetres() * spec.Depth.InMetres() / math.Pow(spec.ParentSpacing.InMetres(), 2)
	parents := clusterPoisson(from, 0, mean)
	sample := ClusterSample{}
	for i := range parents {
		own := sow(citySeed(from.number ^ uint64(i) ^ 0x4e534350))
		at := maths.V(own.swing(0, 0, 0)*spec.Width.InMetres()/2, 0, own.swing(0, 0, 1)*spec.Depth.InMetres()/2)
		parent := ClusterPoint{At: at, Seed: own.number, Parent: own.number}
		sample.parents = append(sample.parents, parent)
		count := clusterPoisson(own, 1, spec.MeanOffspring)
		if len(sample.offspring)+count > 100_000 {
			return ClusterSample{}, fmt.Errorf("manufacture: sampled cluster population exceeds budget")
		}
		for child := range count {
			one := sow(citySeed(from.number ^ (uint64(i) << 32) ^ uint64(child) ^ 0x4f464653))
			u, v := one.draw(2, 0, 0), one.draw(2, 0, 1)
			radius := spec.Spread.InMetres() * math.Sqrt(u)
			if spec.Pattern == ThomasGaussian {
				radius = spec.Spread.InMetres() * math.Sqrt(-2*math.Log(math.Max(u, 1e-15)))
			}
			angle := 2 * math.Pi * v
			sample.offspring = append(sample.offspring, ClusterPoint{
				At: at.Add(maths.V(radius*math.Cos(angle), 0, radius*math.Sin(angle))), Seed: one.number, Parent: own.number})
		}
	}
	return sample, nil
}

// clusterPoisson uses Poisson additivity: bounded Knuth draws sum to the
// requested mean, avoiding exp(-mean) underflow at high parent/child rates.
func clusterPoisson(from seed, channel int, mean float64) int {
	chunks := max(1, int(math.Ceil(mean/16)))
	threshold := math.Exp(-mean / float64(chunks))
	count := 0
	for chunk := range chunks {
		product := 1.0
		for draw := 0; ; draw++ {
			product *= from.draw(700+channel, chunk, draw)
			if product <= threshold {
				break
			}
			count++
		}
	}
	return count
}
