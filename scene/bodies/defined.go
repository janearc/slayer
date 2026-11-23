package bodies

// The numbers every check here counts in, each true by definition.
const (
	// nought is defined: no amount, the start of a share, and the first
	// place in a list.
	nought = 0
	// unity is defined: one amount, and the whole of a share.
	unity = 1
	// infinitePositive is defined by math.IsInf: a sign above zero asks
	// for positive infinity only.
	infinitePositive = 1
	// anySign is defined by math.IsInf: a sign of zero asks for infinity
	// of either sign.
	anySign = 0
)
