package budget

import "math"

const (
	// noLoad is defined as the origin of nonnegative load averages.
	noLoad = 0.0
	// eitherInfinity is defined by math.IsInf as selecting either sign.
	eitherInfinity = 0
)

// hostPolicy holds explicitly supplied load, swap and free-disk thresholds.
// Load is the load-average ceiling. Swap and free disk are in bytes.
// Load and swap ceilings are exclusive. The disk floor is inclusive.
type hostPolicy struct {
	loadLimit float64
	swapLimit byteCount
	diskFloor byteCount
}

// hostSample holds supplied measurements and their availability.
// Failed reads are unknown even when a numerical field has a value.
// stopped is the persisted stop, never inferred from a favorable sample.
// This value does not sample or persist anything.
type hostSample struct {
	load                            float64
	swap, freeDisk                  byteCount
	loadKnown, swapKnown, diskKnown bool
	stopped                         bool
}

// hostDecision describes host conditions. It grants no authority
// to start work, reserve resources, persist a stop or signal a process.
type hostDecision uint8

const (
	hostUnknown hostDecision = iota // absent or invalid measurement
	// hostInvalidPolicy means an absent or invalid threshold.
	hostInvalidPolicy
	// hostStopped means the previous stop remains latched.
	hostStopped
	// hostDiskLow means measured disk is below its floor.
	hostDiskLow
	// hostLoadHigh means measured load reaches its ceiling.
	hostLoadHigh
	// hostSwapHigh means measured swap reaches its ceiling.
	hostSwapHigh
	// hostReady means these host conditions permit work.
	hostReady
)

// assess checks supplied host conditions in the design's stated order.
// It does no I/O, reservations or state changes.
func (policy hostPolicy) assess(sample hostSample) hostDecision {
	if !(policy.loadLimit > noLoad) ||
		math.IsInf(policy.loadLimit, eitherInfinity) ||
		policy.swapLimit == noBytes || policy.diskFloor == noBytes {
		return hostInvalidPolicy
	}
	if sample.stopped {
		return hostStopped
	}
	if sample.diskKnown && sample.freeDisk < policy.diskFloor {
		return hostDiskLow
	}
	if !sample.loadKnown || !sample.swapKnown || !sample.diskKnown ||
		!(sample.load >= noLoad) ||
		math.IsInf(sample.load, eitherInfinity) {
		return hostUnknown
	}
	if sample.load >= policy.loadLimit {
		return hostLoadHigh
	}
	if sample.swap >= policy.swapLimit {
		return hostSwapHigh
	}
	return hostReady
}
