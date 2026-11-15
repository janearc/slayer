package cheeptune

// ramp is a scalar ramp: position in, number out, straight lines between
// stops, flat past either end.
//
// libdaffy has the same shape for colour, position in and swatch out. This
// one is here rather than there because giving a colour library a scalar
// type is a contract decision that has not been made. If a second user turns
// up, that is the time to move it.
type ramp []stop

type stop struct{ at, v float64 }

// at is the ramp's value at x.
func (r ramp) at(x float64) float64 {
	if len(r) == 0 {
		return 0
	}
	if x <= r[0].at {
		return r[0].v
	}
	for i := 1; i < len(r); i++ {
		if x < r[i].at {
			a, b := r[i-1], r[i]
			return a.v + (b.v-a.v)*(x-a.at)/(b.at-a.at)
		}
	}
	return r[len(r)-1].v
}
