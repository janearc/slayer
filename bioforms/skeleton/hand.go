package skeleton

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

var errNoDigit = errors.New("skeleton: a digit needs a metacarpal")

// The thickness of a hand's bones, each a share of the humerus's half
// thickness. These shares are chosen: a flipper's forearm and hand are
// flat plates, thinner than the arm bone, and a digit thins to its tip.
const (
	elbowShare      = .8  // the humerus at the elbow
	forearmShare    = .6  // the radius and the ulna
	forearmApart    = .6  // the two's distance from the line, in thicknesses
	carpalShare     = .5  // a wrist bone
	digitRootShare  = .45 // a metacarpal at the knuckle
	digitTipShare   = .25 // the last phalanx at the tip
	digitTaperShare = .2  // how much of a digit bone's thickness its tail loses
)

// Hand is a fore flipper's bones in a frame of its own: along runs from
// the shoulder toward the tip, across from the trailing edge toward the
// leading one, and the hand lies flat in the plane of the two. Both are
// unit directions at right angles in the skeleton's rest space.
type Hand struct {
	shoulder         maths.Vec
	along, across    maths.Vec
	humerus, forearm float64
	wrist            float64
	thick            float64
	digits           []digit
}

// digit is one finger: where its knuckle lies across the wrist, how far
// it splays from along toward across, in radians, and the lengths of its
// metacarpal and then its phalanges, root to tip.
type digit struct {
	across, splay float64
	bones         []float64
}

// NewHand is a hand with its shoulder joint at shoulder, laid out along
// and across, whose humerus is thick in half thickness.
func NewHand(shoulder, along, across maths.Vec, thick float64) Hand {
	return Hand{shoulder: shoulder, along: along.Unit(),
		across: across.Unit(), thick: thick}
}

// Arm is the hand with its humerus, forearm and wrist this long.
func (hand Hand) Arm(humerus, forearm, wrist float64) Hand {
	hand.humerus, hand.forearm, hand.wrist = humerus, forearm, wrist
	return hand
}

// Digit is the hand with one more finger, numbered from the leading
// edge: its knuckle across the wrist, its splay, and the lengths of its
// metacarpal and then each phalanx.
func (hand Hand) Digit(across, splay float64, bones ...float64) Hand {
	hand.digits = append(append([]digit{}, hand.digits...),
		digit{across: across, splay: splay, bones: bones})
	return hand
}

// Hand adds the bones of hand, each named after prefix, the humerus
// jointed to parent: humerus, radius (leading), ulna (trailing), and for
// each digit n a carpal n, a metacarpal n and phalanges n-1, n-2 and on.
func (skeleton *Skeleton) Hand(prefix, parent string, hand Hand) error {
	if err := skeleton.arm(prefix, parent, hand); err != nil {
		return err
	}
	for n, finger := range hand.digits {
		if err := skeleton.digit(prefix, n+1, hand, finger); err != nil {
			return err
		}
	}
	return nil
}

// arm adds the humerus, the radius and the ulna of hand.
func (skeleton *Skeleton) arm(prefix, parent string, hand Hand) error {
	elbow := hand.shoulder.Add(hand.along.Scale(hand.humerus))
	err := skeleton.Add(prefix+"humerus", parent, hand.shoulder, elbow,
		hand.thick, hand.thick*elbowShare)
	if err != nil {
		return err
	}
	if err := skeleton.forearm(prefix, "radius", elbow, hand, 1); err != nil {
		return err
	}
	return skeleton.forearm(prefix, "ulna", elbow, hand, -1)
}

// forearm adds the forearm bone named name from the elbow, on the
// leading side of the line when sign is one and the trailing when minus.
func (skeleton *Skeleton) forearm(prefix, name string, elbow maths.Vec,
	hand Hand, sign float64,
) error {
	thick := hand.thick * forearmShare
	head := elbow.Add(hand.across.Scale(sign * forearmApart * thick))
	tail := head.Add(hand.along.Scale(hand.forearm))
	return skeleton.Add(prefix+name, prefix+"humerus", head, tail, thick, thick)
}

// digit adds finger n of hand: its carpal from the forearm's end to the
// knuckle, then its metacarpal and phalanges in a line from there.
func (skeleton *Skeleton) digit(prefix string, n int, hand Hand,
	finger digit,
) error {
	if len(finger.bones) == 0 {
		return fmt.Errorf("%w: %s%d", errNoDigit, prefix, n)
	}
	forearm := hand.humerus + hand.forearm
	start := hand.shoulder.Add(hand.along.Scale(forearm)).
		Add(hand.across.Scale(finger.across / 2))
	knuckle := hand.shoulder.Add(hand.along.Scale(forearm + hand.wrist)).
		Add(hand.across.Scale(finger.across))
	parent := prefix + "radius"
	if finger.across < 0 {
		parent = prefix + "ulna"
	}
	carpal := fmt.Sprintf("%scarpal%d", prefix, n)
	thick := hand.thick * carpalShare
	if err := skeleton.Add(carpal, parent, start, knuckle, thick,
		thick); err != nil {
		return err
	}
	return skeleton.phalanges(prefix, n, carpal, knuckle, hand, finger)
}

// phalanges adds finger n's metacarpal and phalanges in a line from the
// knuckle, each thinner than the one before.
func (skeleton *Skeleton) phalanges(prefix string, n int, parent string,
	knuckle maths.Vec, hand Hand, finger digit,
) error {
	alongPart := hand.along.Scale(math.Cos(finger.splay))
	acrossPart := hand.across.Scale(math.Sin(finger.splay))
	way := alongPart.Add(acrossPart)
	head := knuckle
	for k, length := range finger.bones {
		share := float64(k) / float64(len(finger.bones))
		thick := hand.thick * (digitRootShare +
			(digitTipShare-digitRootShare)*share)
		name := fmt.Sprintf("%sphalanx%d-%d", prefix, n, k)
		if k == 0 {
			name = fmt.Sprintf("%smetacarpal%d", prefix, n)
		}
		tail := head.Add(way.Scale(length))
		err := skeleton.Add(name, parent, head, tail, thick,
			thick*(1-digitTaperShare))
		if err != nil {
			return err
		}
		head, parent = tail, name
	}
	return nil
}
