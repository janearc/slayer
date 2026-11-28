// Package skeleton is an animal's bones: a tree of bones, each with its
// rest place and thickness, jointed where the animal is jointed and posed
// by turning those joints. It knows no species. A maker lays out its
// animal's bones from that animal's sources and binds its skin to them.
package skeleton

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/janearc/slayer/maths"
)

// noParent is the parent of a root bone, one jointed to nothing.
const noParent = -1

var (
	errNoParent  = errors.New("skeleton: no such parent bone")
	errSameName  = errors.New("skeleton: a bone of that name already exists")
	errShortBone = errors.New("skeleton: a bone has no length")
	errThickness = errors.New("skeleton: a bone's thickness is not positive")
)

// Bone is one bone at rest: the joint it turns about (its head), its far
// end (its tail), and its half thickness at each end, in metres.
type Bone struct {
	name       string
	parent     int
	head, tail maths.Vec
	thickHead  float64
	thickTail  float64
}

// Name is the bone's name.
func (bone Bone) Name() string { return bone.name }

// Head is the rest place of the joint the bone turns about.
func (bone Bone) Head() maths.Vec { return bone.head }

// Tail is the rest place of the bone's far end.
func (bone Bone) Tail() maths.Vec { return bone.tail }

// Parent is the place of the bone it is jointed to, or minus one.
func (bone Bone) Parent() int { return bone.parent }

// Length is the distance from the bone's head to its tail.
func (bone Bone) Length() float64 { return bone.tail.Sub(bone.head).Len() }

// Thickness is the bone's half thickness at its head and at its tail.
func (bone Bone) Thickness() (head, tail float64) {
	return bone.thickHead, bone.thickTail
}

// Skeleton is the bones of one animal at rest. A bone's parent is always
// added before it, so a walk in order meets every parent first.
type Skeleton struct {
	bones  []Bone
	byName map[string]int
}

// New is a skeleton with no bones.
func New() *Skeleton { return &Skeleton{byName: map[string]int{}} }

func (skeleton *Skeleton) snapshot() Skeleton {
	return Skeleton{bones: slices.Clone(skeleton.bones), byName: maps.Clone(skeleton.byName)}
}

// Add puts a bone on the skeleton, jointed to the bone named parent at
// its head; an empty parent makes a root. A name used twice, a parent
// not yet added, a bone of no length or of no thickness is refused.
func (skeleton *Skeleton) Add(name, parent string, head, tail maths.Vec,
	thickHead, thickTail float64,
) error {
	if _, taken := skeleton.byName[name]; taken {
		return fmt.Errorf("%w: %s", errSameName, name)
	}
	if tail.Sub(head).Len() == 0 {
		return fmt.Errorf("%w: %s", errShortBone, name)
	}
	if !(thickHead > 0 && thickTail > 0) {
		return fmt.Errorf("%w: %s", errThickness, name)
	}
	at := noParent
	if parent != "" {
		found, ok := skeleton.byName[parent]
		if !ok {
			return fmt.Errorf("%w: %s for %s", errNoParent, parent, name)
		}
		at = found
	}
	skeleton.byName[name] = len(skeleton.bones)
	skeleton.bones = append(skeleton.bones, Bone{name: name, parent: at,
		head: head, tail: tail, thickHead: thickHead, thickTail: thickTail})
	return nil
}

// Len is how many bones the skeleton has.
func (skeleton *Skeleton) Len() int { return len(skeleton.bones) }

// Bone is the bone at place i, in the order the bones were added.
func (skeleton *Skeleton) Bone(i int) Bone { return skeleton.bones[i] }

// Find is the place of the bone named name, and whether there is one.
func (skeleton *Skeleton) Find(name string) (int, bool) {
	at, ok := skeleton.byName[name]
	return at, ok
}

// Count is how many bones have names that begin with prefix.
func (skeleton *Skeleton) Count(prefix string) int {
	count := 0
	for _, bone := range skeleton.bones {
		if strings.HasPrefix(bone.name, prefix) {
			count++
		}
	}
	return count
}
