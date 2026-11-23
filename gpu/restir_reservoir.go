package gpu

// A reservoir is what restir keeps for a sample (doc/restir.md): one
// candidate lamp chosen from many by weighted reservoir sampling, and the
// numbers that say how to weigh it. The kernels do this arithmetic on the
// device; this is the same arithmetic on the CPU, which the tests hold to
// its estimate and which the kernels are written against.

// reservoirFloats is the floats a reservoir takes on the device: the
// lamp, W, M and one spare.
const reservoirFloats = 4

// gbufferFloats is the floats of a sample's G-buffer: the point, the
// normal, the index of the sphere the point leaves, and the depth.
const gbufferFloats = 8

type reservoir struct {
	// Lamp is the lamp kept, or minus one for none.
	Lamp int32
	// WSum is the sum of the weights of every candidate offered.
	WSum float32
	// M is how many candidates this reservoir stands for.
	M float32
	// W is the unbiased contribution weight: the chosen lamp's light is
	// its contribution, times W. Nought where nothing is kept.
	W float32
}

// emptyReservoir keeps nothing.
func emptyReservoir() reservoir { return reservoir{Lamp: -1} }

// offer puts one candidate of resampling weight weight to the reservoir,
// which keeps it with the chance of its weight in the sum, by u in the
// half-open range from nought to one. It says whether it was kept.
func (kept *reservoir) offer(lamp int32, weight, u float32) bool {
	kept.WSum += weight
	kept.M++
	if weight > 0 && u*kept.WSum < weight {
		kept.Lamp = lamp
		return true
	}
	return false
}

// finish sets W, given phat of the lamp kept at the point.
func (kept *reservoir) finish(phat float32) {
	kept.W = 0
	if kept.Lamp >= 0 && phat > 0 && kept.M > 0 {
		kept.W = kept.WSum / (kept.M * phat)
	}
}

// join adds another reservoir's candidate to this one. phatHere is the
// other's lamp's target at this reservoir's point; the other's weight is
// that times its W times its M, and its M is capped at cap times the
// M this one had, as the temporal and spatial reuse of the paper do.
// Without a cap, cap is nought. It says whether the other's lamp was kept.
func (kept *reservoir) join(other reservoir, phatHere, u float32,
	limit float32) bool {
	m := other.M
	if limit > 0 && m > limit*max(kept.M, 1) {
		m = limit * max(kept.M, 1)
	}
	weight := float32(0)
	if other.Lamp >= 0 {
		weight = phatHere * other.W * m
	}
	before := kept.M
	chosen := kept.offer(other.Lamp, weight, u)
	kept.M = before + m
	return chosen
}
