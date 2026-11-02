package trace

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// Step records one replay operation: ray index, reflection depth, action, and
// named input and output values. Numeric text uses the shortest form that
// round-trips to the original bits.
type Step struct {
	Ray    int
	Bounce int
	What   string
	Said   string
}

// walk collects replay steps for the current ray and bounce. Ordinary renders
// keep this pointer nil and skip recording after a nil check.
type walk struct {
	ray, bounce int
	steps       []Step
}

// tell appends an action and alternating names and values to the replay.
func (walked *walk) tell(what string, said ...any) {
	walked.steps = append(walked.steps, Step{
		Ray: walked.ray, Bounce: walked.bounce, What: what,
		Said: spoken(said),
	})
}

// spoken formats alternating names and values as space-separated name=value
// pairs.
func spoken(said []any) string {
	var parts []string
	for i := 0; i+1 < len(said); i += 2 {
		part := fmt.Sprint(said[i]) + "=" + exactly(said[i+1])
		parts = append(parts, part)
	}
	return strings.Join(parts, " ")
}

// exactly formats floating-point values with round-trip precision and formats
// vectors, lights, pixels, and rays component by component. Other values use
// the standard fmt representation.
func exactly(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32)
	case maths.Vec:
		return "(" + exactly(v.X) + ", " + exactly(v.Y) + ", " +
			exactly(v.Z) + ")"
	case maths.Light:
		return "rgb(" + exactly(v.R) + ", " + exactly(v.G) + ", " +
			exactly(v.B) + ")"
	case frame.Pixel:
		return "rgb(" + exactly(v.R) + ", " + exactly(v.G) + ", " +
			exactly(v.B) + ") cover " + exactly(v.Cover) +
			" depth " + exactly(v.Depth)
	case maths.Ray:
		return "from " + exactly(v.At) + " along " + exactly(v.Dir)
	case nil:
		return "none"
	}
	if kind := reflect.TypeOf(value); kind.Kind() == reflect.Pointer {
		return kind.Elem().String()
	}
	return fmt.Sprint(value)
}
