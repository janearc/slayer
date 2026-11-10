package manufacture

import "github.com/janearc/slayer/maths"

// guideLauncher is an authored launch tube, open at its positive-y exit.
// Its base is y=0; the adapter places its exit at the beam origin.
type guideLauncher struct{}

// launcherValue reads picked display dimensions in metres or shares.
func launcherValue(name string) float64 { return groundValue("launcher-" + name) }

// made supplies the ordinary quick specimen.
func (row guideLauncher) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt changes circular tessellation while retaining the same open exit.
func (row guideLauncher) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: groundLooks()}}
	length, radius := launcherValue("length"), launcherValue("radius")
	wall := launcherValue("wall")
	assembly.add(groundDrum(radius, radius-wall, length),
		maths.Local(), leaf)
	for _, share := range []float64{launcherValue("rear-collar"),
		launcherValue("front-collar")} {
		pose := maths.NewSubspace(maths.YAxis.Scale(length*share),
			maths.Identity, spaceOne)
		collar := groundDrum(radius+wall, radius,
			launcherValue("collar-length"))
		assembly.add(collar, pose, spaceStructure)
	}
	width := launcherValue("shoe-width")
	corner := maths.V(-width*spaceHalf, launcherValue("shoe-start"),
		radius+wall)
	assembly.box(width, launcherValue("shoe-length"), wall, corner,
		spaceStructure)
	half := launcherValue("half-width")
	low := maths.V(-half, spaceZero, -half)
	high := maths.V(half, length, half)
	bounds := maths.SegmentBox(low, high, spaceZero)
	made, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *made, nil
}
