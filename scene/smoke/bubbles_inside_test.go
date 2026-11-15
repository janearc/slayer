package smoke

import (
	"math"
	"testing"
)

// Every bubble is whole inside the ball of water, with room to spare.
// This is the anti-test of the ball of bubbles: a bubble that broke the
// ball's face would have air on both sides of part of its own, and
// would not be a bubble under water; and a bubble of no size, or a ball
// with none, would be no test at all.
func TestEveryBubbleIsWholeInsideTheBall(t *testing.T) {
	const room = 0.01 // picked: the least water between a bubble and the ball's face
	if len(bubbles) == nought {
		t.Fatal("the ball has no bubbles")
	}
	for at, each := range bubbles {
		out := math.Sqrt(each[0]*each[0] + each[1]*each[1] + each[2]*each[2])
		if !(each[3] > nought) || out+each[3] > ballRadius-room {
			t.Errorf("bubble %d, %v out and %v across, is not whole inside a "+
				"ball of %v", at, out, each[3], ballRadius)
		}
	}
}
