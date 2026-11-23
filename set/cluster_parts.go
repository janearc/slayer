package set

import (
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// What a thing of the kind scavenged-cluster asks of its cluster, which is
// the set's to say: the kinds its parts are drawn from and how likely each
// is, how many parts it has, and how far apart its bands stand along a
// lashing. A cluster that names no parts is drawn from every kind that fits
// the size range in the manufactory's data.

// mostClusterParts is the most parts a cluster may have.
const mostClusterParts = 1000

// leastBandSpacing is the least distance between two bands, in metres.
const leastBandSpacing = 0.01

// clusterAsk is what a cluster thing asks, as read: its parts as a weight
// for each kind, how many it has, and its band spacing, nought where the
// set did not say.
type clusterAsk struct {
	parts       map[string]float64
	count       float64
	bandSpacing float64
}

// Parts is the kinds the cluster's parts are drawn from, each with its
// weight, or none for a cluster drawn from every kind that fits. The map is
// a copy.
func (made Thing) Parts() map[string]float64 {
	if len(made.cluster.parts) == 0 {
		return nil
	}
	copied := make(map[string]float64, len(made.cluster.parts))
	for kind, weight := range made.cluster.parts {
		copied[kind] = weight
	}
	return copied
}

// PartCount is how many parts the cluster has, or nought if the set did
// not say.
func (made Thing) PartCount() int { return int(made.cluster.count) }

// BandSpacing is the distance between bands along a lashing, in metres, or
// nought if the set did not say.
func (made Thing) BandSpacing() float64 { return made.cluster.bandSpacing }

// clusterParts is a step that reads a cluster's parts: each key a kind the
// manufactory makes, other than a cluster, and its value a weight above
// nought.
func (reading reader) clusterParts(target *map[string]float64) step {
	return func(_, value *yaml.Node) error {
		parts := map[string]float64{}
		err := reading.each(value, func(name, weight *yaml.Node) error {
			var share float64
			known := slices.Contains(manufacture.KindNames(), name.Value)
			if !known || name.Value == manufacture.ClusterKind ||
				name.Value == manufacture.SuperblockKind {
				return reading.refuse(name, "parts names %q, which is not "+
					"a kind a cluster or a roof is made of", name.Value)
			}
			if weight.Decode(&share) != nil || !(share > 0) ||
				math.IsInf(share, 0) {
				return reading.refuse(weight, "parts gives %s a weight "+
					"above nought, not %q", name.Value, weight.Value)
			}
			parts[name.Value] = share
			return nil
		})
		*target = parts
		return err
	}
}
