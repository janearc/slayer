package gpu

// whorledStruct is a conifer grown in whorls as the kernel reads it: where it
// stands and manufacture's parameters (ProperConiferParams). no branch is
// stored: the kernel works each one out from its whorl and its place in the
// whorl, as manufacture.BranchOf does, in the tree's own space with up along z.
//
// the axes are the world's directions of the tree's own x, y and z, and the
// scale in origin's w is its unit.
var whorledStruct = structure{"Whorled", []field{
	floatField("origin", "the trunk's foot in the world; scale in w"),
	floatField("x", "the tree's own axes in the world"),
	floatField("y", ""),
	floatField("z", ""),
	floatField("shape", "height, trunk radius, crown start, branch length"),
	intField("count", "whorls, branches in the lowest, bark, needles"),
	intField("seed", "seed low, seed high"),
}}
