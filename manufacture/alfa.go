package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Project 705 "Lira", NATO's Alfa class, the titanium attack submarine,
// as its own kind, so a scene can float it or sink it. Its sizes, where
// each was read, and what it is made of are data, alfa.yaml; its length,
// beam and draughts are measured, and its form is chosen.
//
// The hull's own space is the one it is made in: metres, y up, the keel
// at y nought, the centreline on x nought, the bow toward minus z (the
// way slayer's spaces face) and starboard toward plus x, the middle of
// the length at z nought.
//
// For the water: the hull is one closed solid, wound counterclockwise
// seen from outside, as every part is; the other parts are closed solids
// of their own, sunk into it where they join.
//
// Its surfaced waterline is a named height above the keel, given by
// Floating, by which a scene lowers the hull to sit it at sea level.
// The hull wears the look named "hull"; the look named "wet hull" is
// made and worn by no face, for the water to put where the hull is wet.

//go:embed alfa.yaml
var alfaRaw []byte

// alfaKind is the kind's name in the table.
const alfaKind = "alfa"
const alfaHeroKind = "alfa-hero"

// alfaSizes is the boat's sizes, in metres, its displacements in tonnes,
// its draughts by name, and the look of each of its parts.
type alfaSizes struct {
	Length, Beam float64
	Draughts     map[string]float64
	Draught      string
	Displacement struct{ Surfaced, Submerged float64 }
	Depth        struct{ Operating, Maximum float64 }
	Reserve      float64
	Hull         struct {
		Nose, Bluntness, Tail, Taper, Stern, Boss float64
	}
	Sail struct {
		Front    float64
		Sections []alfaSection
	}
	Sphere struct{ Diameter, Aft, Up, Clear float64 }
	Planes struct{ Aft, Chord, Span, Thick float64 }
	Stern  struct {
		Aft, Chord, Tip, Thick, Across, Up, Down, Rudder, Flap float64
	}
	Pods      struct{ Long, Radius float64 }
	Propeller struct {
		Blades                                   int
		Diameter, Aft, Chord, Thick, Pitch, Skew float64
	}
	Auxiliary struct {
		Blades int
		Scale  float64
	}
	Finish   string
	Finishes map[string]alfaFinish
	Looks    map[string]alfaLook
	Read     string
	// dressed is what it wears, made once when the file is read and
	// held to be sound, so that building it has nothing to refuse.
	dressed looks
}

// alfaFinish is one finish the hull may wear: its look dry, and wet.
type alfaFinish struct{ Dry, Wet alfaLook }

// alfaLook is what one part is made of: a material slayer knows in a
// finish, painted a colour or not, or the chosen finish when it wears
// the coat, and what a source says the part is made of.
type alfaLook struct {
	Material, Finish, Alloy string
	Colour                  []float64
	Coat                    bool
}

// The boat's own looks.
const (
	alfaHull slot = iota
	alfaWet
	alfaBoss
	alfaSail
	alfaSphere
	alfaPlane
	alfaFin
	alfaRudder
	alfaFlap
	alfaPod
	alfaBlade
	alfaMast
	alfaFitting
	alfaAuxBlade
)

// alfaNames is what each of the boat's looks is called, which is also
// how its look is found in the sizes file.
var alfaNames = map[slot]string{
	alfaHull:     "hull",
	alfaWet:      "wet hull",
	alfaBoss:     "propeller boss",
	alfaSail:     "sail",
	alfaSphere:   "rescue sphere",
	alfaPlane:    "bow plane",
	alfaFin:      "stern fin",
	alfaRudder:   "rudder",
	alfaFlap:     "stern plane",
	alfaPod:      "thruster pod",
	alfaBlade:    "propeller blade",
	alfaMast:     "mast and antenna",
	alfaFitting:  "deck fitting",
	alfaAuxBlade: "auxiliary propeller blade",
}

const (
	alfaNoseSteps  = 32        // chosen: corners along the nose's curve; enough for its outline to read as round at 2K
	alfaTailSteps  = 24        // chosen: corners along the tail's taper
	alfaCapSteps   = 6         // chosen: corners round the boss's rounded end
	alfaSink       = 0.1       // chosen: metres a part is sunk into what holds it, so no gap shows
	alfaLeastPower = 1         // defined: a curve of a power below one bends the wrong way
	alfaPainted    = "painted" // defined: the finish book's name for paint
	alfaChannels   = 3         // defined: red, green and blue
)

var alfaRead, alfaError = readAlfa(alfaRaw)

// readAlfa reads the boat's sizes and checks them, so that making it
// afterwards cannot fail.
func readAlfa(raw []byte) (alfaSizes, error) {
	var read alfaSizes
	if err := yamlfile.Read("alfa.yaml", raw, &read); err != nil {
		return alfaSizes{}, err
	}
	for _, check := range []func() error{read.sound, read.drawn,
		read.shaped, read.finished, read.sailSeated, read.sphereFits,
		read.planesFit, read.sternFits, read.withinBeam,
		read.propellerFits} {
		if err := check(); err != nil {
			return alfaSizes{}, err
		}
	}
	dressed, err := read.looks()
	if err != nil {
		return alfaSizes{}, err
	}
	read.dressed = dressed
	return read, nil
}

// lengths is every number the file gives, by name, for sound to check.
func (size alfaSizes) lengths() map[string]float64 {
	hull := size.Hull
	lengths := map[string]float64{"length": size.Length, "beam": size.Beam,
		"surfaced displacement":  size.Displacement.Surfaced,
		"submerged displacement": size.Displacement.Submerged,
		"operating depth":        size.Depth.Operating,
		"maximum depth":          size.Depth.Maximum,
		"reserve":                size.Reserve, "nose": hull.Nose,
		"bluntness": hull.Bluntness, "tail": hull.Tail, "taper": hull.Taper,
		"stern": hull.Stern, "boss": hull.Boss}
	for name, draught := range size.Draughts {
		lengths["draught "+name] = draught
	}
	lengths["sail front"] = size.Sail.Front
	for index, section := range size.Sail.Sections {
		lengths[fmt.Sprintf("sail section %d chord", index)] = section.Chord
		lengths[fmt.Sprintf("sail section %d wide", index)] = section.Wide
	}
	sphere := size.Sphere
	lengths["sphere diameter"], lengths["sphere aft"] = sphere.Diameter,
		sphere.Aft
	lengths["sphere up"], lengths["sphere clear"] = sphere.Up, sphere.Clear
	planes, stern := size.Planes, size.Stern
	for name, each := range map[string]float64{"aft": planes.Aft,
		"chord": planes.Chord, "span": planes.Span, "thick": planes.Thick} {
		lengths["bow plane "+name] = each
	}
	for name, each := range map[string]float64{"aft": stern.Aft,
		"chord": stern.Chord, "tip": stern.Tip, "thick": stern.Thick,
		"across": stern.Across, "up": stern.Up, "down": stern.Down,
		"rudder": stern.Rudder, "flap": stern.Flap,
		"pod long": size.Pods.Long, "pod radius": size.Pods.Radius} {
		lengths["stern "+name] = each
	}
	screw := size.Propeller
	for name, each := range map[string]float64{"diameter": screw.Diameter,
		"aft": screw.Aft, "chord": screw.Chord, "thick": screw.Thick,
		"pitch": screw.Pitch} {
		lengths["propeller "+name] = each
	}
	return lengths
}

// sound refuses a number at nought or below, and sizes read nowhere.
func (size alfaSizes) sound() error {
	for name, each := range size.lengths() {
		if !sound(each) {
			return fmt.Errorf("manufacture: the alfa's %s is %v", name, each)
		}
	}
	if size.Read == "" {
		return fmt.Errorf("manufacture: the alfa's sizes were read nowhere")
	}
	return nil
}

// drawn refuses a draught the file names but does not give, and a
// draught as deep as the hull is tall, which puts no waterline on it.
func (size alfaSizes) drawn() error {
	if _, given := size.Draughts[size.Draught]; !given {
		return fmt.Errorf("manufacture: the alfa's draught %q is not one "+
			"of its draughts, %v", size.Draught, size.Draughts)
	}
	for name, draught := range size.Draughts {
		if draught >= size.Beam {
			return fmt.Errorf("manufacture: the alfa's draught %s, %v, is "+
				"as deep as its hull is tall, %v", name, draught, size.Beam)
		}
	}
	return nil
}

// shaped refuses a hull whose nose, tail and boss are longer than the
// boat, whose stern is as wide as its beam, whose boss is too short for
// its rounded end, or whose curves bend the wrong way.
func (size alfaSizes) shaped() error {
	hull := size.Hull
	if hull.Nose+hull.Tail+hull.Boss >= size.Length ||
		hull.Stern >= size.radius() || hull.Boss <= hull.Stern+alfaSink {
		return fmt.Errorf("manufacture: the alfa's hull, its nose %v, tail "+
			"%v and boss %v long to a stern of %v, does not fit its %v "+
			"length and %v beam", hull.Nose, hull.Tail, hull.Boss,
			hull.Stern, size.Length, size.Beam)
	}
	if hull.Bluntness < alfaLeastPower || hull.Taper < alfaLeastPower {
		return fmt.Errorf("manufacture: the alfa's hull curves by the "+
			"powers %v and %v; each needs one or more", hull.Bluntness,
			hull.Taper)
	}
	return nil
}

// finished refuses a finish, chosen or not, whose dry or wet look
// cannot be made, so that choosing it later cannot fail.
func (size alfaSizes) finished() error {
	for name, finish := range size.Finishes {
		for _, look := range []alfaLook{finish.Dry, finish.Wet} {
			if _, err := alfaLookOf("finish "+name, look); err != nil {
				return err
			}
		}
	}
	return nil
}

// looks is what the boat wears: each part's own look, or the chosen
// finish where it wears the coat, dry, or wet for the wet hull. A part
// with no look or no alloy said, a finish the file does not give, and a
// material or finish slayer does not know are refused.
func (size alfaSizes) looks() (looks, error) {
	coat, given := size.Finishes[size.Finish]
	if !given {
		return nil, fmt.Errorf("manufacture: the alfa's finish %q is not "+
			"one of its finishes", size.Finish)
	}
	dressed := looks{}
	for where, name := range alfaNames {
		look, given := size.Looks[name]
		if !given || look.Alloy == "" {
			return nil, fmt.Errorf("manufacture: the alfa's %s has no look, "+
				"or says nothing of its alloy", name)
		}
		made, err := alfaLookOf(name, worn(where, look, coat))
		if err != nil {
			return nil, err
		}
		made.Name = name
		dressed[where] = made
	}
	return dressed, nil
}

// worn is the look a part wears: its own, or the coat's, wet for the
// wet hull and dry for the rest.
func worn(where slot, look alfaLook, coat alfaFinish) alfaLook {
	switch {
	case !look.Coat:
		return look
	case where == alfaWet:
		return coat.Wet
	}
	return coat.Dry
}

// alfaLookOf is one look: a material in a finish, or painted a colour.
func alfaLookOf(name string, look alfaLook) (*thing.Material, error) {
	whose := "the alfa's " + name
	if look.Finish != alfaPainted {
		return surfaceOf(whose, look.Material, look.Finish)
	}
	if len(look.Colour) != alfaChannels || !share(look.Colour[0]) ||
		!share(look.Colour[1]) || !share(look.Colour[2]) {
		return nil, fmt.Errorf("manufacture: %s's colour %v is not three "+
			"shares of nought to one", whose, look.Colour)
	}
	kind, err := materials.ByName(look.Material)
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", whose, err)
	}
	return thing.Made(kind, thing.Painted(maths.RGB(look.Colour[0],
		look.Colour[1], look.Colour[2])))
}

// radius is the hull's radius at its widest, half the beam, which is
// also how high its axis stands over the keel.
func (size alfaSizes) radius() float64 {
	return size.Beam * spaceHalf
}

// waterline is how high the surfaced waterline stands over the keel:
// the draught the file names.
func (size alfaSizes) waterline() float64 {
	return size.Draughts[size.Draught]
}

// hullEnd is how far aft of the bow the hull ends and the boss begins.
func (size alfaSizes) hullEnd() float64 {
	return size.Length - size.Hull.Boss
}

// tailStart is how far aft of the bow the parallel middle ends and the
// tail begins.
func (size alfaSizes) tailStart() float64 {
	return size.hullEnd() - size.Hull.Tail
}

// hullRadius is the hull's radius so far aft of the bow: on its nose a
// superellipse, on its middle the beam's, on its tail falling to the
// stern by its power, and nought off either end.
func (size alfaSizes) hullRadius(aft float64) float64 {
	hull, radius := size.Hull, size.radius()
	switch {
	case aft <= spaceZero || aft >= size.hullEnd():
		return spaceZero
	case aft < hull.Nose:
		ahead := math.Pow((hull.Nose-aft)/hull.Nose, hull.Bluntness)
		return radius * math.Pow(spaceOne-ahead, spaceOne/hull.Bluntness)
	case aft <= size.tailStart():
		return radius
	}
	return size.tailRadius((aft - size.tailStart()) / hull.Tail)
}

// tailRadius is the hull's radius a share of the way down its tail,
// from the beam's at nought to the stern's at one.
func (size alfaSizes) tailRadius(share float64) float64 {
	hull := size.Hull
	return hull.Stern + (size.radius()-hull.Stern)*(spaceOne-
		math.Pow(share, hull.Taper))
}

// at is a point of the boat's own space: so far out to starboard, so
// high over the keel, and so far aft of the bow.
func (size alfaSizes) at(starboard, up, aft float64) maths.Vec {
	return maths.V(starboard, up, aft-size.Length*spaceHalf)
}

// axial is the space a turned profile is made in, placed on the hull's
// axis: its y runs aft from the bow, and its turn is about the axis.
func (size alfaSizes) axial() maths.Subspace {
	return size.axialAt(spaceZero, size.radius(), spaceZero)
}

// axialAt is a turned profile's space with its axis parallel to the
// hull's, through a point so far to starboard, so high and so far aft.
func (size alfaSizes) axialAt(starboard, up, aft float64) maths.Subspace {
	return maths.NewSubspace(size.at(starboard, up, aft), maths.Orientation{
		X: maths.XAxis, Y: maths.ZAxis, Z: maths.YAxis.Neg()}, spaceOne)
}

// hullProfile is the hull's turned profile, radius and distance aft of
// the bow: from the bow's point round the nose, along the middle, down
// the tail to the stern, and in to the axis at the hull's end.
func (size alfaSizes) hullProfile() [][coordinatePair]float64 {
	hull, radius := size.Hull, size.radius()
	points := [][coordinatePair]float64{{spaceZero, spaceZero}}
	power := coordinatePair / hull.Bluntness
	for step := spaceOne; step <= alfaNoseSteps; step++ {
		turn := math.Pi * spaceHalf * float64(step) / alfaNoseSteps
		points = append(points, [coordinatePair]float64{
			radius * math.Pow(math.Sin(turn), power),
			hull.Nose * (spaceOne - math.Pow(math.Cos(turn), power))})
	}
	for step := range alfaTailSteps + spaceOne {
		share := float64(step) / alfaTailSteps
		points = append(points, [coordinatePair]float64{
			size.tailRadius(share), size.tailStart() + hull.Tail*share})
	}
	return append(points, [coordinatePair]float64{spaceZero, size.hullEnd()})
}

// bossProfile is the propeller boss's turned profile: from inside the
// hull's end at the stern's radius aft to its rounded end at the
// boat's length.
func (size alfaSizes) bossProfile() [][coordinatePair]float64 {
	stern := size.Hull.Stern
	start := size.hullEnd() - alfaSink
	points := [][coordinatePair]float64{{spaceZero, start}, {stern, start}}
	for step := range alfaCapSteps {
		turn := math.Pi * spaceHalf * float64(step) / alfaCapSteps
		points = append(points, [coordinatePair]float64{
			stern * math.Cos(turn), size.Length - stern + stern*math.Sin(turn)})
	}
	return append(points, [coordinatePair]float64{spaceZero, size.Length})
}

// alfaTurned is a turned profile shaded smooth along its length as well
// as round its axis: a corner's normal is that of the faces meeting at
// it within the crease, so a curved profile shows no rings where its
// corners turn, and a sharp turn, as at the hull's end, stays sharp.
type alfaTurned struct{ turnedProfile }

// normals is the turned skin's creased normals.
func (shape alfaTurned) normals(skin surface) [][3]maths.Vec {
	return creased(skin)
}

// build puts every drawn part of the boat into an assembly.
func (size alfaSizes) build(assembly *instrumentAssembly, detail string) {
	assembly.add(alfaTurned{turnedProfile{points: size.hullProfile()}},
		size.axial(), alfaHull)
	assembly.add(alfaTurned{turnedProfile{points: size.bossProfile()}},
		size.axial(), alfaBoss)
	assembly.add(alfaLoft{rings: size.sailRings()}, maths.Local(), alfaSail)
	assembly.add(roundProfile{turnedProfile{points: alfaBall(
		size.Sphere.Diameter * spaceHalf)}}, maths.NewSubspace(
		size.sphereCentre(), maths.Identity, spaceOne), alfaSphere)
	size.planes(assembly)
	size.stern(assembly)
	size.propeller(assembly)
	size.auxiliaryPropellers(assembly)
	if detail == "demo" {
		size.demoDetails(assembly)
	}
}

// alfaBoat is the maker of the boat, on its keel in its own space.
type alfaBoat struct{ finish string }

// alfaRows supplies the boat by name. A sizes file that cannot be read
// stops the build, as the other embedded data files do.
func alfaRows() map[string]maker {
	if alfaError != nil {
		panic(alfaError)
	}
	return map[string]maker{alfaKind: alfaBoat{}, alfaHeroKind: alfaBoat{finish: "painted"}}
}

// made keeps quick as the boat's default detail.
func (row alfaBoat) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt builds the boat at a detail, its box round every part. Its
// repeated flat parts are kept as shared mesh groups.
func (row alfaBoat) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	size := alfaRead
	if row.finish != "" {
		size.Finish = row.finish
		dressed, err := size.looks()
		if err != nil {
			return good{}, err
		}
		size.dressed = dressed
	}
	assembly := size.laid(detail)
	bounds := partBounds(assembly.object.parts)
	if err := assembly.shareRepeats(); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	object, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// laid is the boat's assembly at a detail, dressed in its own copy of
// each look, with every part placed and none yet shared.
func (size alfaSizes) laid(detail string) *instrumentAssembly {
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: size.dressedCopy()}}
	size.build(assembly, detail)
	return assembly
}

// dressedCopy is a fresh copy of each of the boat's looks, so that one
// made boat's looks are its own.
func (size alfaSizes) dressedCopy() looks {
	dressed := looks{}
	for where, look := range size.dressed {
		own := *look
		dressed[where] = &own
	}
	return dressed
}

// Afloat is what the water needs of a kind that floats: how high its
// surfaced waterline stands over its keel, the name of the look its
// hull wears, and the look of its hull wet, which no face wears.
type Afloat struct {
	Waterline measure.Length
	Hull      string
	Wet       *thing.Material
}

// Floating is what the water needs of a kind, by its name, and whether
// the kind floats at all. A scene lowers the kind by its waterline to
// sit it in the sea; the water wets its hull with the wet look.
func Floating(name string) (Afloat, bool) {
	if (name != alfaKind && name != alfaHeroKind) || alfaError != nil {
		return Afloat{}, false
	}
	size := alfaRead
	if name == alfaHeroKind {
		size.Finish = "painted"
		var err error
		size.dressed, err = size.looks()
		if err != nil {
			return Afloat{}, false
		}
	}
	wet := *size.dressed[alfaWet]
	return Afloat{Waterline: measure.Metres(alfaRead.waterline()),
		Hull: alfaNames[alfaHull], Wet: &wet}, true
}
