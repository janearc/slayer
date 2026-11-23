package mesh

// What the package counts in, each defined by what it counts.
const (
	nought          = 0 // defined: no amount, and the first place of anything
	unity           = 1 // defined: one, and the step to the next place
	pair            = 2 // defined: two, the place after the next, and a halving
	axes            = 3 // defined: x, y and z, the parts of a point
	triangleCorners = 3 // defined: a triangle has three corners
	boundEnds       = 2 // defined: a box has two ends on an axis
	plusInfinity    = 1 // defined by math.Inf: the sign for plus infinity
)

// The places of a triangle's three corners, and of the three weights
// that go with them.
const (
	first  = 0 // defined: the first corner of a face
	second = 1 // defined: the second corner of a face
	third  = 2 // defined: the third corner of a face
)

// The places of a point's three parts.
const (
	alongX = 0 // defined: x is the first part of a point
	alongY = 1 // defined: y is the second part of a point
	alongZ = 2 // defined: z is the third part of a point
)

// The places of a box's two ends on an axis.
const (
	low  = 0 // defined: the end of a box with the smaller part
	high = 1 // defined: the end of a box with the larger part
)

// pushed is how many children a walk puts on its stack for one node: a
// stack as deep as the tree needs this many places more.
const pushed = pair
