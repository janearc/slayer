package session

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

// A thing of the kind scavenged-cluster is the set's to describe: which
// kinds its parts are, how many it has and how far apart its bands stand.
// A thing of the kind superblock is the set's to describe too: its parcel,
// its ceiling, its habit and the parts on its roofs. Another thing is made
// as its kind and recipe say.

// clusterKey is what a thing says of its cluster or its superblock, as
// text, so that two that differ are not taken from the stock as one. It is
// nothing for a thing that is neither.
func clusterKey(each set.Thing) string {
	switch each.Kind() {
	case manufacture.ClusterKind:
		return fmt.Sprint(each.Parts(), each.PartCount(), each.BandSpacing())
	case manufacture.SuperblockKind:
		accretion, drawn := each.Accretion()
		return fmt.Sprint(each.Parts(), each.ParcelWidth(),
			each.ParcelDepth(), each.Ceiling(), each.Habit(),
			each.Occupants(), each.Crowding(), accretion, drawn)
	}
	return ""
}

// madeCraft is a thing's kind made at its complexity and seed: a cluster
// or a superblock as the set describes it, or any other as its kind and
// recipe say.
func (stage *staging) madeCraft(each set.Thing) (*kept, error) {
	if clusterKey(each) == "" {
		return stage.constructed(each.Kind(), each.Recipe(),
			each.Complexity(), each.Seed())
	}
	row, err := askedKind(each)
	if err != nil {
		return nil, err
	}
	one, err := manufacturedKind(stage.given, row, int(each.Seed()),
		each.Complexity())
	return &one, err
}

// askedKind is the kind of a cluster or a superblock thing, with what the
// set asks of it.
func askedKind(each set.Thing) (manufacture.Kind, error) {
	row, err := manufacture.KindNamed(each.Kind())
	if err != nil {
		return row, err
	}
	if each.Kind() == manufacture.SuperblockKind {
		accretion, drawn := each.Accretion()
		return manufacture.Superblocked(row, manufacture.SuperblockAsk{
			Width: each.ParcelWidth(), Depth: each.ParcelDepth(),
			Ceiling: each.Ceiling(), Habit: each.Habit(),
			Parts: each.Parts(), Occupants: each.Occupants(),
			Crowding: each.Crowding(), Accretion: accretion, Drawn: drawn})
	}
	return manufacture.Clustered(row, manufacture.ClusterAsk{
		Parts: each.Parts(), Count: each.PartCount(),
		BandSpacing: each.BandSpacing()})
}
