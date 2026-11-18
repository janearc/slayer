package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

// The Pratt & Whitney J58, the A-12's engine, as its own kind, so that
// it can lie alone, stand on a test stand, or sit in a nacelle. Its
// sizes and what it is made of are data, j58.yaml, with where each was
// read; every station and every radius but the largest is picked.
//
// The engine is built in its own space: its axis is y, running aft
// from the compressor face at nought, and its top is z. A host places
// that space; the engine is not told what holds it.

//go:embed j58.yaml
var j58Raw []byte

// j58Sizes is the engine's sizes, in metres, its mass in kilograms, its
// counts, and the look of each of its parts.
type j58Sizes struct {
	Length, Hot, Diameter, Envelope, Mass, Pressure float64
	Counts                                          struct {
		Compressor, Turbine, Cans, Tubes, Bleed int
	}
	Stations struct {
		Frame, Compressor, Burner, Turbine, Afterburner float64
	}
	Radii struct {
		Flange, Face, Spinner, Exit, Burner, Turbine, Wall, Rim,
		Seam float64
	}
	Vanes struct {
		Count        int
		Chord, Thick float64
	}
	Blades struct {
		Compressor        []int
		Chord, Thick, Gap float64
	}
	Tubes   struct{ Across, Rise, Enter float64 }
	Cans    struct{ Boss, Height, Manifold float64 }
	Turbine struct {
		Hub, Chord float64
		Blades     []int
	}
	Afterburner struct {
		Cone, Spray, Holder, Bar, Gutter, Sheet float64
		Rings                                   []float64
		Struts                                  int
	}
	Nozzle struct {
		Exit, Rib, Rod   float64
		Flaps, Actuators int
	}
	Mounts struct{ Forward, Aft, Reach, Lug float64 }
	Looks  map[string]j58Look
	Read   string
	// dressed is what it wears, made once when the file is read and
	// held to be sound, so that building it has nothing to refuse.
	dressed looks
}

// j58Look is what one part is made of: the metal as a source says it,
// and the look slayer draws for it, a material and a finish.
type j58Look struct{ Material, Finish, Alloy string }

// The engine's own looks.
const (
	j58Frame slot = iota
	j58Compressor
	j58Vane
	j58Spinner
	j58Blade
	j58Burner
	j58Turbine
	j58Afterburner
	j58Flange
	j58Tube
	j58Boss
	j58Manifold
	j58TurbineBlade
	j58Disc
	j58Cone
	j58Spray
	j58Holder
	j58Strut
	j58Nozzle
	j58Flap
	j58Actuator
	j58Lug
	j58Slots // defined: how many looks the engine has; a host's start here
)

// j58Names is what each of the engine's looks is called, which is also
// how its look is found in the sizes file.
var j58Names = map[slot]string{
	j58Frame:       "front frame",
	j58Compressor:  "compressor case",
	j58Vane:        "inlet guide vane",
	j58Spinner:     "spinner",
	j58Blade:       "compressor blade",
	j58Burner:      "burner case",
	j58Turbine:     "turbine case",
	j58Afterburner: "afterburner case",
	j58Flange:      "flange",
	j58Tube:        "bypass tube",
	j58Boss:        "can boss",
	j58Manifold:    "fuel manifold",

	j58TurbineBlade: "turbine blade",
	j58Disc:         "turbine disc",
	j58Cone:         "tail cone",
	j58Spray:        "spray ring",
	j58Holder:       "flame holder",
	j58Strut:        "afterburner strut",
	j58Nozzle:       "primary nozzle",
	j58Flap:         "nozzle flap",
	j58Actuator:     "nozzle actuator",
	j58Lug:          "mount lug",
}

// j58LugTurns is where the mount lugs stand round the axis, in degrees
// from the top. picked: one each side, level with the axis, where a
// cradle's posts or a nacelle's links can reach them and the tubes,
// on top, under and at sixty degrees from them, leave room.
var j58LugTurns = []float64{90, 270} // picked: as the comment above says

const (
	j58Gamma      = 1.4  // defined: the ratio of air's specific heats, a diatomic gas
	j58Diffuser   = 0.2  // picked: metres over which the case widens from the compressor to the burner
	j58NoseSteps  = 6    // picked: corners along the spinner's nose; enough to read as round
	j58NoseTip    = 0.03 // picked: metres aft of the face where the spinner's point stands, so the face is a plane
	j58VaneAt     = 0.6  // picked: the vanes' middle is this share of the way back through the front frame
	j58VaneSink   = 0.8  // picked: a vane's root is this share of the spinner's radius out, sunk into it
	j58BladePitch = 40   // picked: degrees a compressor blade's chord turns from the axis; its twist is not drawn
	j58RootSink   = 0.01 // picked: metres a blade's root is sunk into the hub, so no gap shows
	j58Least      = 1    // defined: one of anything is the fewest there can be

	j58TubeTurn     = 0    // picked: degrees round from the top of the first tube, so a tube is on top and one under, and the sides are clear for the mounts
	j58BallBands    = 8    // picked: bands from pole to pole on a tube's elbow; enough to read as round
	j58TorusSides   = 12   // picked: flat sides round the manifold's pipe; enough to read as round
	j58JointGap     = 1e-3 // picked: metres each pipe stops short of a bend, inside its ball, so two pipes' ends never share a corner in one mesh
	j58TurbinePitch = -50  // picked: degrees a turbine blade's chord turns from the axis, the other hand from the compressor's, as a turbine's blades are
)

var j58Read, j58Error = readJ58(j58Raw)

// readJ58 reads the engine's sizes and checks them, so that making it
// afterwards cannot fail.
func readJ58(raw []byte) (j58Sizes, error) {
	var read j58Sizes
	if err := yamlfile.Read("j58.yaml", raw, &read); err != nil {
		return j58Sizes{}, err
	}
	for _, check := range []func() error{read.sound, read.counted,
		read.stationed, read.fits, read.tubesFit, read.coreFits,
		read.afterburnerFits, read.nozzleFits, read.mountsFit} {
		if err := check(); err != nil {
			return j58Sizes{}, err
		}
	}
	dressed, err := read.looks()
	if err != nil {
		return j58Sizes{}, err
	}
	read.dressed = dressed
	return read, nil
}

// lengths is every length the file gives, by name, for sound to check.
func (size j58Sizes) lengths() map[string]float64 {
	radii, stations := size.Radii, size.Stations
	return map[string]float64{"length": size.Length, "hot": size.Hot,
		"diameter": size.Diameter, "envelope": size.Envelope,
		"mass": size.Mass, "pressure": size.Pressure,
		"frame": stations.Frame, "flange": radii.Flange,
		"face": radii.Face, "spinner": radii.Spinner, "exit": radii.Exit,
		"burner": radii.Burner, "turbine": radii.Turbine,
		"wall": radii.Wall, "rim": radii.Rim, "seam": radii.Seam,
		"vane chord": size.Vanes.Chord, "vane thick": size.Vanes.Thick,
		"blade chord": size.Blades.Chord, "blade thick": size.Blades.Thick,
		"blade gap": size.Blades.Gap, "tube across": size.Tubes.Across,
		"tube rise": size.Tubes.Rise, "tube enter": size.Tubes.Enter,
		"can boss": size.Cans.Boss, "can height": size.Cans.Height,
		"manifold": size.Cans.Manifold, "turbine hub": size.Turbine.Hub,
		"turbine chord": size.Turbine.Chord, "cone": size.Afterburner.Cone,
		"spray": size.Afterburner.Spray, "holder": size.Afterburner.Holder,
		"bar": size.Afterburner.Bar, "gutter": size.Afterburner.Gutter,
		"sheet": size.Afterburner.Sheet, "nozzle exit": size.Nozzle.Exit,
		"rib": size.Nozzle.Rib, "rod": size.Nozzle.Rod,
		"mount forward": size.Mounts.Forward, "mount aft": size.Mounts.Aft,
		"mount reach": size.Mounts.Reach, "mount lug": size.Mounts.Lug}
}

// sound refuses a size at nought or below, and sizes read nowhere.
func (size j58Sizes) sound() error {
	for name, each := range size.lengths() {
		if !sound(each) {
			return fmt.Errorf("manufacture: the j58's %s is %v", name, each)
		}
	}
	if size.Read == "" {
		return fmt.Errorf("manufacture: the j58's sizes were read nowhere")
	}
	return nil
}

// counted refuses a count below one, a compressor whose blade list is
// not one row a stage, and bleed air taken from a stage it lacks.
func (size j58Sizes) counted() error {
	counts := size.Counts
	for _, each := range []int{counts.Compressor, counts.Turbine,
		counts.Cans, counts.Tubes, counts.Bleed, size.Vanes.Count} {
		if each < j58Least {
			return fmt.Errorf("manufacture: the j58's counts are %+v and "+
				"%d vanes; each needs one", counts, size.Vanes.Count)
		}
	}
	if len(size.Blades.Compressor) != counts.Compressor ||
		counts.Bleed > counts.Compressor {
		return fmt.Errorf("manufacture: the j58 has %d compressor stages, "+
			"%d rows of blades, and bleeds from stage %d", counts.Compressor,
			len(size.Blades.Compressor), counts.Bleed)
	}
	if len(size.Turbine.Blades) != counts.Turbine {
		return fmt.Errorf("manufacture: the j58 has %d turbine stages and "+
			"%d rows of turbine blades", counts.Turbine,
			len(size.Turbine.Blades))
	}
	rows := append(append([]int{}, size.Blades.Compressor...),
		size.Turbine.Blades...)
	for stage, blades := range rows {
		if blades < j58Least {
			return fmt.Errorf("manufacture: the j58's stage %d has %d "+
				"blades", stage+spaceOne, blades)
		}
	}
	return nil
}

// stationed refuses sections out of order from front to back, or past
// the engine's length.
func (size j58Sizes) stationed() error {
	stations := size.Stations
	order := []float64{spaceZero, stations.Frame, stations.Compressor,
		stations.Compressor + j58Diffuser, stations.Burner,
		stations.Turbine, stations.Afterburner, size.Length}
	for at := spaceOne; at < len(order); at++ {
		if order[at] <= order[at-spaceOne] {
			return fmt.Errorf("manufacture: the j58's stations %+v are "+
				"not in order inside its length, %v", stations, size.Length)
		}
	}
	return nil
}

// fits refuses radii that cannot be built together: an envelope no wider
// than the case, a section wider than the largest case, a compressor
// that widens, a spinner no narrower than the face, a hub that closes
// the annulus, and vanes too long for the front frame.
func (size j58Sizes) fits() error {
	radii := size.Radii
	largest := size.Diameter * spaceHalf
	if size.Envelope <= size.Diameter ||
		math.Max(radii.Burner, radii.Turbine)+radii.Rim >= largest ||
		radii.Flange+radii.Rim > largest {
		return fmt.Errorf("manufacture: the j58's radii %+v do not fit "+
			"inside its %v case and %v envelope", radii, size.Diameter,
			size.Envelope)
	}
	if radii.Exit >= radii.Face || radii.Spinner >= radii.Face ||
		radii.Face >= radii.Flange || radii.Exit+radii.Wall >= radii.Burner {
		return fmt.Errorf("manufacture: the j58's compressor, %v across at "+
			"its face round a %v spinner and %v at its exit, cannot be made",
			radii.Face, radii.Spinner, radii.Exit)
	}
	if size.Blades.Gap+j58RootSink >= size.tip(spaceOne)-size.hub(spaceOne) {
		return fmt.Errorf("manufacture: the j58's last stage has no room "+
			"for a blade between its hub, %v, and its case, %v",
			size.hub(spaceOne), size.tip(spaceOne))
	}
	vane := size.Stations.Frame * j58VaneAt
	if size.Vanes.Chord*spaceHalf >= math.Min(vane-j58NoseTip,
		size.Stations.Frame-vane) {
		return fmt.Errorf("manufacture: the j58's vanes, %v long, do not "+
			"fit in its front frame", size.Vanes.Chord)
	}
	return nil
}

// tubesFit refuses bypass tubes that would not clear the largest case
// inside the envelope, that would not rise out of the compressor to
// their run, or that would reach the afterburner before the turbine's
// flange or past the afterburner's end.
func (size j58Sizes) tubesFit() error {
	tubes := size.Tubes
	if size.tubeRadius()-tubes.Across*spaceHalf <= size.Diameter*spaceHalf {
		return fmt.Errorf("manufacture: the j58's bypass tubes, %v across, "+
			"do not clear its %v case inside its %v envelope", tubes.Across,
			size.Diameter, size.Envelope)
	}
	share := size.stageShare(size.Counts.Bleed - spaceOne)
	stub := size.tip(share) + size.Radii.Wall + tubes.Across
	enter := size.Stations.Turbine + tubes.Enter
	if stub >= size.tubeRadius() ||
		size.along(share)+tubes.Rise >= size.Stations.Turbine ||
		tubes.Enter <= size.Radii.Seam || enter >= size.Stations.Afterburner {
		return fmt.Errorf("manufacture: the j58's bypass tubes cannot rise "+
			"%v from stage %d and enter the afterburner %v aft of the "+
			"turbine", tubes.Rise, size.Counts.Bleed, tubes.Enter)
	}
	return nil
}

// coreFits refuses can bosses or a manifold that would reach the tubes
// over them, bosses too wide for the burner, and a turbine whose hub
// leaves its blades no room or whose blades are too long for a stage.
func (size j58Sizes) coreFits() error {
	cans, radii := size.Cans, size.Radii
	under := size.tubeRadius() - size.Tubes.Across*spaceHalf
	drum := size.Stations.Burner - size.Stations.Compressor - j58Diffuser
	if radii.Burner+cans.Height >= under || cans.Boss >= drum ||
		radii.Burner+(cans.Height+cans.Manifold)*spaceHalf >= under {
		return fmt.Errorf("manufacture: the j58's can bosses, %v high and "+
			"%v across, do not fit between its burner case and its tubes",
			cans.Height, cans.Boss)
	}
	turbine := size.Turbine
	stage := (size.Stations.Turbine - size.Stations.Burner) /
		float64(size.Counts.Turbine)
	if turbine.Hub+size.Blades.Gap+j58RootSink >= radii.Turbine-radii.Wall ||
		turbine.Chord >= stage {
		return fmt.Errorf("manufacture: the j58's turbine, its hub %v and "+
			"its blades %v long, does not fit its case", turbine.Hub,
			turbine.Chord)
	}
	return nil
}

// afterburnerFits refuses a tail cone reaching the spray rings, spray
// rings not ahead of the flame holders, holders past the afterburner's
// end, rings out of order or outside its case, and no struts.
func (size j58Sizes) afterburnerFits() error {
	burner := size.Afterburner
	turbine := size.Stations.Turbine
	if burner.Cone >= burner.Spray || burner.Spray+burner.Bar >= burner.Holder ||
		turbine+burner.Holder+burner.Gutter >= size.Stations.Afterburner {
		return fmt.Errorf("manufacture: the j58's afterburner, its cone %v, "+
			"spray rings %v and holders %v aft of the turbine, is not in "+
			"order inside its case", burner.Cone, burner.Spray, burner.Holder)
	}
	inner := size.afterburner() - size.Radii.Wall
	half, low := burner.Gutter*spaceHalf, float64(spaceZero)
	for _, ring := range burner.Rings {
		if ring-half <= low || ring+half >= inner {
			return fmt.Errorf("manufacture: the j58's afterburner rings %v "+
				"do not fit inside its case, %v", burner.Rings, inner)
		}
		low = ring + half
	}
	if len(burner.Rings) < j58Least || burner.Struts < j58Least ||
		burner.Sheet+burner.Sheet >= burner.Gutter {
		return fmt.Errorf("manufacture: the j58's afterburner has %d rings, "+
			"%d struts and holders of %v sheet", len(burner.Rings),
			burner.Struts, burner.Sheet)
	}
	return nil
}

// nozzleFits refuses a nozzle that widens or does not close round the
// last ring, and one with no flaps or actuators.
func (size j58Sizes) nozzleFits() error {
	nozzle := size.Nozzle
	rings := size.Afterburner.Rings
	widest := rings[len(rings)-spaceOne] + size.Afterburner.Gutter*spaceHalf
	if nozzle.Exit+size.Radii.Wall >= size.afterburner()-size.Radii.Wall ||
		nozzle.Exit <= widest || nozzle.Flaps < j58Least ||
		nozzle.Actuators < j58Least {
		return fmt.Errorf("manufacture: the j58's nozzle, %v at its exit "+
			"with %d flaps and %d actuators, cannot be made", nozzle.Exit,
			nozzle.Flaps, nozzle.Actuators)
	}
	return nil
}

// mountsFit refuses mounts out of order or off the cases they hold, pins
// inside a case or its flange or past the envelope, and a lug in the way
// of a bypass tube.
func (size j58Sizes) mountsFit() error {
	mounts, stations := size.Mounts, size.Stations
	half := mounts.Lug * spaceHalf
	refused := fmt.Errorf("manufacture: the j58's mounts, at %v and %v aft "+
		"and %v out, do not fit it", mounts.Forward, mounts.Aft, mounts.Reach)
	if mounts.Forward-half <= stations.Frame || mounts.Aft <= mounts.Forward ||
		mounts.Aft+half >= stations.Afterburner-size.Radii.Seam ||
		mounts.Reach >= size.Envelope*spaceHalf {
		return refused
	}
	for _, pin := range size.pins() {
		if mounts.Reach <= size.caseOutside(pin.Y)+size.Radii.Rim {
			return refused
		}
		for tube := range size.Counts.Tubes {
			run := j58Radial(size.tubeTurn(tube)).Scale(size.tubeRadius())
			if math.Hypot(pin.X-run.X, pin.Z-run.Z) <=
				half*spaceHalf+size.Tubes.Across*spaceHalf {
				return refused
			}
		}
	}
	return nil
}

// caseOutside is the outside radius of the case so far aft of the face,
// without its flanges.
func (size j58Sizes) caseOutside(along float64) float64 {
	radii, stations := size.Radii, size.Stations
	diffuser := stations.Compressor + j58Diffuser
	switch {
	case along < stations.Frame:
		return radii.Flange
	case along < stations.Compressor:
		return size.tip((along-stations.Frame)/
			(stations.Compressor-stations.Frame)) + radii.Wall
	case along < diffuser:
		share := (along - stations.Compressor) / j58Diffuser
		return radii.Exit + radii.Wall + (radii.Burner-radii.Exit-
			radii.Wall)*share
	case along < stations.Burner:
		return radii.Burner
	case along < stations.Turbine:
		return radii.Turbine
	case along < stations.Afterburner:
		return size.afterburner()
	}
	return size.nozzleOutside(along)
}

// pins is where the mount lugs' pins stand, in the engine's own space:
// the forward pair, then the aft, each in the order of j58LugTurns.
func (size j58Sizes) pins() []maths.Vec {
	var pins []maths.Vec
	for _, along := range []float64{size.Mounts.Forward, size.Mounts.Aft} {
		for _, degrees := range j58LugTurns {
			pins = append(pins, j58Radial(degrees*maths.Degree).Scale(
				size.Mounts.Reach).Add(maths.Up.Scale(along)))
		}
	}
	return pins
}

// lugs stands a mount lug under each pin, out of the case's wall to the
// pin, as long along the axis as the file says and half as thick.
func (size j58Sizes) lugs(build j58Build) {
	mounts := size.Mounts
	for at, pin := range size.pins() {
		turn := j58LugTurns[at%len(j58LugTurns)] * maths.Degree
		root := size.caseOutside(pin.Y) - size.Radii.Wall*spaceHalf
		build.add(boxShape(mounts.Lug*spaceHalf, mounts.Reach-root,
			mounts.Lug), j58Blading(turn, root, pin.Y, spaceZero), j58Lug)
	}
}

// tubeRadius is how far from the axis each bypass tube's run is: its
// outside on the read envelope.
func (size j58Sizes) tubeRadius() float64 {
	return (size.Envelope - size.Tubes.Across) * spaceHalf
}

// tip is the compressor's tip radius, the inside of its case, at a share
// of the way from the face to the last stage: it narrows evenly.
func (size j58Sizes) tip(share float64) float64 {
	return size.Radii.Face + (size.Radii.Exit-size.Radii.Face)*share
}

// hub is the compressor's hub radius at a share of the way from the face
// to the last stage. derived: the air is pressed denser by the pressure
// ratio to the power one over gamma, and the annulus narrows as it does,
// at a steady axial speed (picked), spread evenly by share.
func (size j58Sizes) hub(share float64) float64 {
	radii := size.Radii
	face := radii.Face*radii.Face - radii.Spinner*radii.Spinner
	denser := math.Pow(size.Pressure, share/j58Gamma)
	tip := size.tip(share)
	return math.Sqrt(tip*tip - face/denser)
}

// stageShare is how far through the compressor a stage's middle is.
func (size j58Sizes) stageShare(stage int) float64 {
	return (float64(stage) + spaceHalf) / float64(size.Counts.Compressor)
}

// along is how far aft of the face a share through the compressor is.
func (size j58Sizes) along(share float64) float64 {
	stations := size.Stations
	return stations.Frame + (stations.Compressor-stations.Frame)*share
}

// looks is what the engine wears: each part's material in its finish,
// as the sizes file gives it. A part with no look there, a look with no
// alloy said, and a material or finish slayer does not know are refused.
func (size j58Sizes) looks() (looks, error) {
	dressed := looks{}
	for where, name := range j58Names {
		look, given := size.Looks[name]
		if !given || look.Alloy == "" {
			return nil, fmt.Errorf("manufacture: the j58's %s has no look, "+
				"or says nothing of its alloy", name)
		}
		made, err := surfaceOf("the j58's "+name, look.Material, look.Finish)
		if err != nil {
			return nil, err
		}
		made.Name = name
		dressed[where] = made
	}
	return dressed, nil
}

// j58Mount is what a host needs to hold the engine, in the engine's own
// space: its axis is y, the compressor face is at Front, the
// afterburner's aft flange at Aft, nothing reaches further than Radius
// from the axis, and the engine is held by its Pins.
type j58Mount struct {
	Front, Aft, Radius float64
	Pins               []maths.Vec
}

// mount is the engine's mount, for a nacelle or a stand to place it by.
func (size j58Sizes) mount() j58Mount {
	return j58Mount{Front: spaceZero, Aft: size.Stations.Afterburner,
		Radius: size.Envelope * spaceHalf, Pins: size.pins()}
}

// j58Build is an assembly and the place the engine's own space has in
// it: every part is added through it, placed there.
type j58Build struct {
	assembly *instrumentAssembly
	place    maths.Subspace
}

// add puts a part, in the engine's own space, into the assembly.
func (build j58Build) add(shape form, space maths.Subspace, material slot) {
	build.assembly.add(shape, space.Within(build.place), material)
}

// turned turns a closed profile of radius and distance aft about the
// engine's axis, its corners carrying the round surface's normal.
func (build j58Build) turned(points [][coordinatePair]float64,
	material slot,
) {
	build.add(roundProfile{turnedProfile{points: points}}, maths.Local(),
		material)
}

// j58Radial is the way out from the engine's axis at a turn round it,
// from its top toward its x.
func j58Radial(turn float64) maths.Vec {
	return maths.V(math.Sin(turn), spaceZero, math.Cos(turn))
}

// j58Shell is a case's wall: its inside through the given corners of
// radius and distance aft, its outside a wall's thickness further out,
// closed at both ends, counterclockwise as seen from outside.
func j58Shell(inside [][coordinatePair]float64,
	wall float64,
) [][coordinatePair]float64 {
	loop := [][coordinatePair]float64{inside[spaceZero]}
	for _, corner := range inside {
		loop = append(loop, [coordinatePair]float64{corner[spaceZero] + wall,
			corner[spaceOne]})
	}
	for at := len(inside) - spaceOne; at > spaceZero; at-- {
		loop = append(loop, inside[at])
	}
	return loop
}

// j58Ring is a ring round the axis from inner to outer radius, from one
// distance aft to another.
func j58Ring(inner, outer, front, back float64) [][coordinatePair]float64 {
	return [][coordinatePair]float64{{inner, front}, {outer, front},
		{outer, back}, {inner, back}}
}

// build puts every drawn part of the engine into an assembly, at a place.
func (size j58Sizes) build(assembly *instrumentAssembly, place maths.Subspace) {
	build := j58Build{assembly: assembly, place: place}
	size.cases(build)
	size.flanges(build)
	size.spinner(build)
	size.vanes(build)
	size.compressorBlades(build)
	size.bypassTubes(build)
	size.burnerCans(build)
	size.turbine(build)
	size.afterburnerInside(build)
	size.nozzle(build)
	size.lugs(build)
}

// afterburnerInside sets what is seen through the nozzle: the turbine's
// tail cone, a spray ring on each of the afterburner's rings and a flame
// holder behind it, a V open aft, each ring held by struts from the case.
func (size j58Sizes) afterburnerInside(build j58Build) {
	burner, turbine := size.Afterburner, size.Stations.Turbine
	build.turned([][coordinatePair]float64{{spaceZero, turbine},
		{size.Turbine.Hub, turbine}, {spaceZero, turbine + burner.Cone}},
		j58Cone)
	spray, holder := turbine+burner.Spray, turbine+burner.Holder
	for _, ring := range burner.Rings {
		build.turned(j58Torus(ring, burner.Bar*spaceHalf, spray), j58Spray)
		build.turned(size.gutter(ring, holder), j58Holder)
	}
	wall := size.afterburner() - size.Radii.Wall*spaceHalf
	inner := burner.Rings[spaceZero] - burner.Gutter*spaceHalf
	for strut := range burner.Struts {
		out := j58Radial(fullTurn * (float64(strut) + spaceHalf) /
			float64(burner.Struts))
		for _, along := range []float64{spray, holder + burner.Gutter*
			spaceHalf} {
			build.pipe(out.Scale(inner).Add(maths.Up.Scale(along)),
				out.Scale(wall).Add(maths.Up.Scale(along)),
				burner.Bar*spaceHalf, j58Strut)
		}
	}
}

// gutter is a flame holder cut across: a V of sheet with its point
// forward at a ring's radius and its mouth open aft, a gutter wide,
// counterclockwise as seen from outside.
func (size j58Sizes) gutter(ring, along float64) [][coordinatePair]float64 {
	burner := size.Afterburner
	half, sheet := burner.Gutter*spaceHalf, burner.Sheet
	mouth := along + burner.Gutter
	return [][coordinatePair]float64{{ring, along}, {ring + half, mouth},
		{ring + half - sheet, mouth}, {ring, along + sheet + sheet},
		{ring - half + sheet, mouth}, {ring - half, mouth}}
}

// nozzleOutside is the primary nozzle's outside radius so far aft: flush
// with the afterburner case's inside at its end, closing evenly to the
// exit, a wall out from the nozzle's inside there.
func (size j58Sizes) nozzleOutside(along float64) float64 {
	start := size.afterburner() - size.Radii.Wall
	end := size.Nozzle.Exit + size.Radii.Wall
	share := (along - size.Stations.Afterburner) /
		(size.Length - size.Stations.Afterburner)
	return start + (end-start)*share
}

// nozzle sets the variable primary nozzle, drawn open: a cone of flaps
// from inside the afterburner's end to its exit at the engine's length,
// a rib down each flap's edge where it overlaps the next, and the
// actuators from the aft flange to the flaps' middles.
func (size j58Sizes) nozzle(build j58Build) {
	radii, nozzle := size.Radii, size.Nozzle
	end := size.Stations.Afterburner
	inside := size.afterburner() - radii.Wall - radii.Wall
	build.turned(j58Shell([][coordinatePair]float64{{inside, end - radii.Seam},
		{inside, end}, {nozzle.Exit, size.Length}}, radii.Wall), j58Nozzle)
	from, to := end+radii.Seam, size.Length-radii.Seam
	for flap := range nozzle.Flaps {
		turn := fullTurn * float64(flap) / float64(nozzle.Flaps)
		build.add(boxShape(nozzle.Rib, size.nozzleRun(from, to).Len(),
			nozzle.Rib), size.flapSpace(turn, from, to), j58Flap)
	}
	middle := (end + size.Length) * spaceHalf
	for actuator := range nozzle.Actuators {
		out := j58Radial(fullTurn * (float64(actuator) + spaceHalf) /
			float64(nozzle.Actuators))
		build.pipe(out.Scale(size.Diameter*spaceHalf-radii.Rim).Add(
			maths.Up.Scale(end+radii.Seam*spaceHalf)),
			out.Scale(size.nozzleOutside(middle)).Add(maths.Up.Scale(middle)),
			nozzle.Rod*spaceHalf, j58Actuator)
	}
}

// nozzleRun is the run along the nozzle's outside from one distance aft
// to another, as a radius and a distance aft.
func (size j58Sizes) nozzleRun(from, to float64) maths.Vec {
	return maths.V(size.nozzleOutside(to)-size.nozzleOutside(from),
		to-from, spaceZero)
}

// flapSpace is the space of a flap's rib at a turn round the axis: on
// the nozzle's outside, half sunk into it, its y along the nozzle from
// one distance aft toward another and its z out of the surface.
func (size j58Sizes) flapSpace(turn, from, to float64) maths.Subspace {
	out := j58Radial(turn)
	run := size.nozzleRun(from, to)
	along := out.Scale(run.X).Add(maths.Up.Scale(run.Y)).Unit()
	normal := out.Scale(run.Y).Sub(maths.Up.Scale(run.X)).Unit()
	return maths.NewSubspace(out.Scale(size.nozzleOutside(from)).Add(
		maths.Up.Scale(from)), maths.Orientation{X: along.Cross(normal),
		Y: along, Z: normal}, spaceOne)
}

// pipe lays a round rod of a radius from one point to another in the
// engine's own space, keeping the first error as rod does.
func (build j58Build) pipe(from, to maths.Vec, radius float64,
	material slot,
) {
	run := to.Sub(from)
	space, err := spaceAlong(from, run)
	if err != nil {
		if build.assembly.trouble == nil {
			build.assembly.trouble = err
		}
		return
	}
	build.add(roundProfile{turnedProfile{points: j58Ring(spaceZero, radius,
		spaceZero, run.Len())}}, space, material)
}

// ball lays a ball of a radius about a point in the engine's own space,
// where two pipes meet, so the bend shows no notch.
func (build j58Build) ball(at maths.Vec, radius float64, material slot) {
	var side [][coordinatePair]float64
	for band := range j58BallBands + spaceOne {
		turn := math.Pi * float64(band) / j58BallBands
		reach := radius * math.Sin(turn)
		if band == spaceZero || band == j58BallBands {
			reach = spaceZero // the poles exactly, not sin's residue
		}
		side = append(side, [coordinatePair]float64{reach,
			-radius * math.Cos(turn)})
	}
	build.add(roundProfile{turnedProfile{points: side}},
		maths.NewSubspace(at, maths.Identity, spaceOne), material)
}

// j58Torus is a thin ring cut across: a circle of a radius so far out
// from the axis and so far aft, counterclockwise as seen from outside.
func j58Torus(around, radius, along float64) [][coordinatePair]float64 {
	var loop [][coordinatePair]float64
	for corner := range j58TorusSides {
		turn := fullTurn * float64(corner) / j58TorusSides
		loop = append(loop, [coordinatePair]float64{
			around + radius*math.Cos(turn), along + radius*math.Sin(turn)})
	}
	return loop
}

// tubeRun is the corners a bypass tube runs through at a turn round the
// axis: up out of the compressor's wall at the bleed stage, out and aft
// to its run on the envelope, aft past the turbine's flange, in over the
// afterburner case and down into its wall at the augmentor inlet.
func (size j58Sizes) tubeRun(turn float64) []maths.Vec {
	out := j58Radial(turn)
	at := func(radius, along float64) maths.Vec {
		return out.Scale(radius).Add(maths.Up.Scale(along))
	}
	share := size.stageShare(size.Counts.Bleed - spaceOne)
	port, wall := size.along(share), size.Radii.Wall
	outside := size.tip(share) + wall
	enter := size.Stations.Turbine + size.Tubes.Enter
	above := size.afterburner() + size.Tubes.Across*spaceHalf
	return []maths.Vec{at(outside-wall*spaceHalf, port),
		at(outside+size.Tubes.Across, port),
		at(size.tubeRadius(), port+size.Tubes.Rise),
		at(size.tubeRadius(), size.Stations.Turbine+size.Radii.Seam),
		at(above, enter), at(size.afterburner()-wall*spaceHalf, enter)}
}

// tubeTurn is how far round the axis a bypass tube stands, in radians.
func (size j58Sizes) tubeTurn(tube int) float64 {
	return j58TubeTurn*maths.Degree +
		fullTurn*float64(tube)/float64(size.Counts.Tubes)
}

// bypassTubes lays the bleed collar round the bleed stage and the six
// tubes from it, evenly round the engine, each a run of pipes with a
// ball at each bend.
func (size j58Sizes) bypassTubes(build j58Build) {
	radius, radii := size.Tubes.Across*spaceHalf, size.Radii
	share := size.stageShare(size.Counts.Bleed - spaceOne)
	outside, port := size.tip(share)+radii.Wall, size.along(share)
	build.turned(j58Ring(outside-radii.Wall*spaceHalf, outside+radii.Rim,
		port-radii.Seam, port+radii.Seam), j58Flange)
	for tube := range size.Counts.Tubes {
		run := size.tubeRun(size.tubeTurn(tube))
		last := len(run) - spaceOne
		for step := spaceOne; step <= last; step++ {
			from, to := run[step-spaceOne], run[step]
			short := to.Sub(from).Unit().Scale(j58JointGap)
			if step > spaceOne {
				from = from.Add(short)
			}
			if step < last {
				to = to.Sub(short)
				build.ball(run[step], radius, j58Tube)
			}
			build.pipe(from, to, radius, j58Tube)
		}
	}
}

// burnerMiddle is how far aft the middle of the burner's drum is.
func (size j58Sizes) burnerMiddle() float64 {
	return (size.Stations.Compressor + j58Diffuser + size.Stations.Burner) *
		spaceHalf
}

// burnerCans sets what shows of the eight cans outside the burner case:
// a fuel boss over each, between the tubes, and the manifold ring that
// feeds them, round the drum's middle.
func (size j58Sizes) burnerCans(build j58Build) {
	cans, radii := size.Cans, size.Radii
	middle := size.burnerMiddle()
	build.turned(j58Torus(radii.Burner+cans.Height*spaceHalf,
		cans.Manifold*spaceHalf, middle), j58Manifold)
	for can := range size.Counts.Cans {
		out := j58Radial(j58TubeTurn*maths.Degree + fullTurn*
			(float64(can)+spaceHalf)/float64(size.Counts.Cans))
		build.pipe(out.Scale(radii.Burner-radii.Wall*spaceHalf).Add(
			maths.Up.Scale(middle)), out.Scale(radii.Burner+cans.Height).Add(
			maths.Up.Scale(middle)), cans.Boss*spaceHalf, j58Boss)
	}
}

// turbineAlong is how far aft a turbine stage's middle is.
func (size j58Sizes) turbineAlong(stage int) float64 {
	stations := size.Stations
	return stations.Burner + (stations.Turbine-stations.Burner)*
		(float64(stage)+spaceHalf)/float64(size.Counts.Turbine)
}

// turbine sets the turbine's disc, the hub its blades stand on, and its
// two rings of blades, pitched the other hand from the compressor's,
// with a band on the case over each stage.
func (size j58Sizes) turbine(build j58Build) {
	turbine, radii := size.Turbine, size.Radii
	build.turned(j58Ring(spaceZero, turbine.Hub, size.Stations.Burner,
		size.Stations.Turbine), j58Disc)
	inside := radii.Turbine - radii.Wall
	root, tip := turbine.Hub-j58RootSink, inside-size.Blades.Gap
	for stage, count := range turbine.Blades {
		along := size.turbineAlong(stage)
		build.turned(j58Ring(inside+radii.Wall*spaceHalf, radii.Turbine+
			radii.Rim*spaceHalf, along-radii.Seam*spaceHalf,
			along+radii.Seam*spaceHalf), j58Flange)
		for at := range count {
			turn := fullTurn * float64(at) / float64(count)
			build.add(boxShape(size.Blades.Thick, tip-root, turbine.Chord),
				j58Blading(turn, root, along, j58TurbinePitch*maths.Degree),
				j58TurbineBlade)
		}
	}
}

// cases lays the engine's cases from front to back: the front frame,
// the compressor case narrowing with its blades' tips, the burner case
// widening out of it, the turbine case and the afterburner case, whose
// flanges reach the read diameter.
func (size j58Sizes) cases(build j58Build) {
	radii, stations := size.Radii, size.Stations
	build.turned(j58Shell([][coordinatePair]float64{{radii.Face, spaceZero},
		{radii.Face, stations.Frame}}, radii.Flange-radii.Face), j58Frame)
	build.turned(j58Shell([][coordinatePair]float64{
		{radii.Face, stations.Frame}, {radii.Exit, stations.Compressor}},
		radii.Wall), j58Compressor)
	burner := radii.Burner - radii.Wall
	build.turned(j58Shell([][coordinatePair]float64{
		{radii.Exit, stations.Compressor},
		{burner, stations.Compressor + j58Diffuser},
		{burner, stations.Burner}}, radii.Wall), j58Burner)
	turbine := radii.Turbine - radii.Wall
	build.turned(j58Shell([][coordinatePair]float64{
		{turbine, stations.Burner}, {turbine, stations.Turbine}},
		radii.Wall), j58Turbine)
	build.turned(j58Shell([][coordinatePair]float64{
		{size.afterburner() - radii.Wall, stations.Turbine},
		{size.afterburner() - radii.Wall, stations.Afterburner}},
		radii.Wall), j58Afterburner)
}

// afterburner is the afterburner case's outside radius: a flange's rim
// inside the read diameter, so its flanges reach that diameter exactly.
func (size j58Sizes) afterburner() float64 {
	return size.Diameter*spaceHalf - size.Radii.Rim
}

// flanges rings each joint between two cases with a flange standing out
// of both and sunk into their walls, and the face with one, its front
// on the face, where the inlet's duct is bolted.
func (size j58Sizes) flanges(build j58Build) {
	radii, stations := size.Radii, size.Stations
	joints := []struct{ at, before, after float64 }{
		{stations.Frame, radii.Flange, radii.Face + radii.Wall},
		{stations.Compressor, radii.Exit + radii.Wall, radii.Exit +
			radii.Wall},
		{stations.Burner, radii.Burner, radii.Turbine},
		{stations.Turbine, radii.Turbine, size.afterburner()},
		{stations.Afterburner, size.afterburner(), size.afterburner()}}
	for _, joint := range joints {
		inner := math.Min(joint.before, joint.after) - radii.Wall*spaceHalf
		outer := math.Max(joint.before, joint.after) + radii.Rim
		build.turned(j58Ring(inner, outer, joint.at-radii.Seam*spaceHalf,
			joint.at+radii.Seam*spaceHalf), j58Flange)
	}
	build.turned(j58Ring(radii.Flange-radii.Wall, radii.Flange+radii.Rim,
		spaceZero, radii.Seam), j58Flange)
}

// spinner lays the hub from its pointed nose, just aft of the face, back
// through the compressor, rising as the annulus narrows, and closed at
// the compressor's end.
func (size j58Sizes) spinner(build j58Build) {
	radii, stations := size.Radii, size.Stations
	points := [][coordinatePair]float64{{spaceZero, j58NoseTip}}
	for step := spaceOne; step <= j58NoseSteps; step++ {
		turn := math.Pi * spaceHalf * float64(step) / j58NoseSteps
		points = append(points, [coordinatePair]float64{
			radii.Spinner * math.Sin(turn),
			stations.Frame - (stations.Frame-j58NoseTip)*math.Cos(turn)})
	}
	for stage := range size.Counts.Compressor {
		share := size.stageShare(stage)
		points = append(points, [coordinatePair]float64{size.hub(share),
			size.along(share)})
	}
	points = append(points, [coordinatePair]float64{size.hub(spaceOne),
		stations.Compressor}, [coordinatePair]float64{spaceZero,
		stations.Compressor})
	build.turned(points, j58Spinner)
}

// noseRadius is the spinner's radius so far aft of the face, on its nose.
func (size j58Sizes) noseRadius(along float64) float64 {
	nose := size.Stations.Frame - j58NoseTip
	return size.Radii.Spinner * math.Sin(math.Acos((size.Stations.Frame-
		along)/nose))
}

// vanes stands the inlet guide vanes in the front frame, straight along
// the axis, each from inside the spinner out into the frame's wall.
func (size j58Sizes) vanes(build j58Build) {
	vanes := size.Vanes
	middle := size.Stations.Frame * j58VaneAt
	root := size.noseRadius(middle-vanes.Chord*spaceHalf) * j58VaneSink
	tip := size.Radii.Face + size.Radii.Wall*spaceHalf
	for at := range vanes.Count {
		turn := fullTurn * float64(at) / float64(vanes.Count)
		build.add(boxShape(vanes.Thick, tip-root, vanes.Chord),
			j58Blading(turn, root, middle, spaceZero), j58Vane)
	}
}

// j58Blading is the space of a blade or vane at a turn round the axis:
// its y out along the radius from its root, its z along its chord,
// turned from the axis by pitch toward the way round.
func j58Blading(turn, root, along, pitch float64) maths.Subspace {
	out := j58Radial(turn)
	round := maths.Up.Cross(out)
	chord := maths.Up.Scale(math.Cos(pitch)).Add(round.Scale(math.Sin(pitch)))
	return maths.NewSubspace(out.Scale(root).Add(maths.Up.Scale(along)),
		maths.Orientation{X: out.Cross(chord), Y: out, Z: chord}, spaceOne)
}

// compressorBlades sets the nine rings of rotor blades, each blade from
// inside the hub out to a gap short of the case, its chord pitched.
func (size j58Sizes) compressorBlades(build j58Build) {
	blades := size.Blades
	pitch := j58BladePitch * maths.Degree
	for stage, count := range blades.Compressor {
		share := size.stageShare(stage)
		root := size.hub(share) - j58RootSink
		tip := size.tip(share) - blades.Gap
		for at := range count {
			turn := fullTurn * float64(at) / float64(count)
			build.add(boxShape(blades.Thick, tip-root, blades.Chord),
				j58Blading(turn, root, size.along(share), pitch), j58Blade)
		}
	}
}

// lying is the engine's own space as the kind lays it alone: its axis
// level, its front toward plus x, its top up, and its middle over the
// origin, high enough that the envelope's bottom is on the ground.
func (size j58Sizes) lying() maths.Subspace {
	return size.lyingAt(size.Envelope * spaceHalf)
}

// lyingAt is the engine's own space laid as lying lays it, its axis at a
// height: so a stand holds it, with its pins level either side.
func (size j58Sizes) lyingAt(height float64) maths.Subspace {
	return maths.NewSubspace(maths.V(size.Length*spaceHalf, height,
		spaceZero), maths.Orientation{X: maths.ZAxis.Neg(),
		Y: maths.XAxis.Neg(), Z: maths.Up}, spaceOne)
}

// extentIn is the box round the engine's envelope, a cylinder as long as
// the engine and as wide as the envelope, placed.
func (size j58Sizes) extentIn(place maths.Subspace) maths.Box {
	reach := size.Envelope * spaceHalf
	low := maths.V(math.Inf(spaceOne), math.Inf(spaceOne), math.Inf(spaceOne))
	high := low.Neg()
	for _, x := range []float64{-reach, reach} {
		for _, y := range []float64{spaceZero, size.Length} {
			for _, z := range []float64{-reach, reach} {
				at := place.Out(maths.V(x, y, z))
				low = maths.V(math.Min(low.X, at.X), math.Min(low.Y, at.Y),
					math.Min(low.Z, at.Z))
				high = maths.V(math.Max(high.X, at.X),
					math.Max(high.Y, at.Y), math.Max(high.Z, at.Z))
			}
		}
	}
	return maths.Box{Min: low, Max: high}
}

// j58Engine is the maker of the engine lying alone.
type j58Engine struct{}

// j58Rows supplies the engine and its test stand, by name. A sizes file
// that cannot be read stops the build, as the other embedded data files
// do.
func j58Rows() map[string]maker {
	for _, err := range []error{j58Error, j58StandError} {
		if err != nil {
			panic(err)
		}
	}
	return map[string]maker{"j58": j58Engine{},
		"j58-test-stand": j58Stand{}}
}

// made keeps quick as the engine's default detail.
func (row j58Engine) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt builds the engine lying alone at a detail. Its repeated flat
// parts, the blades and vanes, are kept as shared mesh groups, each
// distinct one stored once.
func (row j58Engine) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	size := j58Read
	assembly := size.laid(detail)
	if err := assembly.shareRepeats(); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	object, err := assembly.finish(size.extentIn(size.lying()))
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// laid is the engine's assembly lying alone at a detail, dressed in its
// own copy of each look, with every part placed and none yet shared.
func (size j58Sizes) laid(detail string) *instrumentAssembly {
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: size.dressedCopy()}}
	size.build(assembly, size.lying())
	return assembly
}

// dressedCopy is a fresh copy of each of the engine's looks, so that one
// made engine's looks are its own.
func (size j58Sizes) dressedCopy() looks {
	dressed := looks{}
	for where, look := range size.dressed {
		own := *look
		dressed[where] = &own
	}
	return dressed
}
