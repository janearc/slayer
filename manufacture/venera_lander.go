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

// The lander of Venera 9 to 14, which Vega 1 and 2 flew again: a flat
// hollow ring that crushed on landing, eighteen short legs, a round
// payload module, a flat drag plate on top of it and the can that held
// the parachutes, with the aerial wound round it. Its sizes and its
// coats are data, venera-lander.yaml, with where each was read.
//
// The module's pressure vessel, the heat shield and the mass are held
// as data and checked, and not drawn: the vessel is inside the
// module's insulation, and the shield is the sphere the lander entered
// in, for the descent, which comes later.

//go:embed venera-lander.yaml
var veneraLanderRaw []byte

// veneraSizes is the lander's sizes, in metres, its mass in kilograms,
// and the colour of each of its coats.
type veneraSizes struct {
	Shell float64
	Ring  struct {
		Radius, Tube, Wide, Hole float64
		Teeth, Holes             int
		Tooth                    struct{ Tall, Base, Thick float64 }
	}
	Skirt struct {
		Across, Tall, Slat float64
		Slats              int
	}
	Legs struct {
		Count       int
		Strut, Boss float64
		Foot        struct{ Wide, Tall, Deep float64 }
	}
	Module struct {
		Across, Vessel, Height, Band, Proud, Window, Housing float64
		Fittings                                             int
		Box                                                  struct {
			Wide, Tall, Deep float64
		}
	}
	Plate struct {
		Across, Thick, Dish, Lip, Rim, Hub, Stay, Seam float64
		Stays                                          int
	}
	Bolt  float64
	Bolts struct{ Plate, Band, Cap int }
	Cable float64
	Pipe  float64
	Booms struct{ Strut, Head, Girth float64 }
	Can   struct {
		Across, Height, Core, Cap float64
		Ribs, Stiffeners, Lugs    int
	}
	Arm struct {
		Length, Wide, Rail float64
		Rungs              int
	}
	Caps  struct{ Across, Deep, Thick float64 }
	Chart struct{ Wide, Tall, Post, Thick float64 }
	Coats map[string]veneraColour
	Mass  float64
	Read  string
	// dressed is what it wears, made once when the file is read and held
	// to be sound, so that building it afterwards has nothing to refuse.
	dressed looks
}

// veneraColour is one coat's colour, each channel a share of the light
// it gives back.
type veneraColour struct{ Red, Green, Blue float64 }

// The lander's own looks.
const (
	veneraRing slot = iota
	veneraLeg
	veneraModule
	veneraPlate
	veneraCan
	veneraAerial
	veneraStay
	veneraBand
	veneraWindow
	veneraBox
	veneraArm
	veneraCap
	veneraChart
	veneraChartWhite
	veneraChartRed
	veneraChartGreen
	veneraChartBlue
	veneraHole
	veneraSkirt
	veneraBolt
	veneraCable
	veneraPipe
	veneraHead
)

// veneraPipeTurns is where the two pipes come down beside the module, in
// degrees round from north. picked: side by side, as Figure 1 shows
// them, between two stays.
var veneraPipeTurns = []float64{235, 245} // picked: as the comment above says

// veneraBoomTurns is where the two booms rise from the ring, in degrees
// round from north. picked: apart, as Figure 1 shows one either side,
// each between two legs' tops.
var veneraBoomTurns = []float64{210, 330} // picked: as the comment above says

// veneraSeams is where the plate's ring seams run, as shares of the way
// from its middle to its rim. picked: by eye from the reference sheet's
// museum models, where the plate is made of rings.
var veneraSeams = []float64{0.45, 0.75} // picked: as the comment above says

// veneraPatches is the colour chart's patches, left to right.
var veneraPatches = []slot{veneraChartWhite, veneraChartRed,
	veneraChartGreen, veneraChartBlue}

// veneraNames is what each of the lander's looks is called, which is
// also how its coat is found in the sizes file.
var veneraNames = map[slot]string{
	veneraRing:   "impact ring",
	veneraLeg:    "leg",
	veneraModule: "payload module",
	veneraPlate:  "drag plate",
	veneraCan:    "parachute can",
	veneraAerial: "aerial",
	veneraStay:   "plate stay",
	veneraBand:   "module band",
	veneraWindow: "camera window",
	veneraBox:    "equipment box",
	veneraArm:    "spring arm",
	veneraCap:    "lens cap",
	veneraChart:  "chart panel",

	veneraChartWhite: "chart white",
	veneraChartRed:   "chart red",
	veneraChartGreen: "chart green",
	veneraChartBlue:  "chart blue",
	veneraHole:       "ring hole",
	veneraSkirt:      "module skirt",
	veneraBolt:       "bolt",
	veneraCable:      "cable",
	veneraPipe:       "pipe",
	veneraHead:       "boom head",
}

const (
	veneraLegsLeast    = 3       // defined: three legs are the fewest that stand
	veneraRibsLeast    = 1       // defined: an aerial wound once is still wound
	veneraStaysLeast   = 3       // defined: three stays are the fewest that hold a plate level
	veneraStayRise     = 40      // picked: degrees above the module's middle where the stays leave it, from the reference sheet's museum models
	veneraStayReach    = 0.8     // picked: the stays meet the plate this share of the way out to its rim, from the reference sheet's museum models
	veneraBandDrop     = 0.25    // picked: the band's middle is this share of its height below the module's middle, from the reference sheet
	veneraBandInside   = 0.9     // picked: the band reaches in to this share of the module's radius, hidden inside it, so no seam shows
	veneraWindows      = 2       // picked: two cameras, on opposite sides, from Venera 13's two panoramas on the reference sheet
	veneraWindowRise   = 20      // picked: degrees above the module's middle the camera housings stand, from the reference sheet
	veneraWindowTilt   = 30      // picked: degrees below level the cameras look, down toward the ring they photographed
	veneraHousingOut   = 0.6     // picked: the camera housing stands this share of its across out of the module, from the reference sheet
	veneraChamfer      = 0.3     // picked: the housing's outer edge is bevelled by this share of its radius
	veneraGlass        = 0.1     // picked: the window stands this share of its across proud of the housing
	veneraBoxInside    = 0.94    // picked: each box's back is this share of the module's radius out, sunk into the module
	veneraBoxEvery     = 2       // picked: every other fitting round the equator is a box and the rest round, from the reference sheet
	veneraFittingShare = 0.3     // picked: a round fitting's radius is this share of a box's height
	veneraLegsPerFoot  = 3       // read: "a Tri-leg configuration with 18 total legs", page 10; three to a foot on the ring is this file's reading of the name and of Figure 15(c), page 19
	veneraToothAt      = 60      // picked: degrees up round the ring's tube from its outer edge where the teeth stand, from Venera 13's photographs on the reference sheet
	veneraToothSink    = 0.2     // picked: each tooth's foot is sunk this share of its height into the ring, so no gap shows under it
	veneraThird        = 1.0 / 3 // defined: a triangle's middle is a third of its height above its base
	veneraArmTurn      = 105     // picked: degrees round from north where the arm lies flat on the ground, in front of the first camera with its lens cap beside it, as in Venera 13's photographs
	veneraCapsOut      = 0.25    // picked: metres beyond the ring's edge the lens caps lie, from Venera 13's photographs
	veneraCapTurn      = 30      // picked: degrees round from its camera where each lens cap lies
	veneraCapTilt      = 20      // picked: degrees each lens cap lies tipped on its rim
	veneraCapSteps     = 6       // picked: corners along each lens cap's curve; enough to read as round
	veneraChartTurn    = 70      // picked: degrees round from north where the colour chart stands on the ring, near the arm
	veneraChartLean    = 45      // picked: degrees the colour chart leans back, facing up and in toward the camera above it
	veneraPatchShare   = 0.8     // picked: each patch fills this share of its place on the chart, the panel showing round it
	veneraHoleProud    = 0.1     // picked: each hole's dark face stands this share of its radius proud of the ring's wall, so it shows and reads as flush
	veneraSkirtInside  = 0.6     // picked: the skirt reaches in to this share of the module's radius, hidden inside it
	veneraRivetInset   = 0.2     // picked: each row of rivets on the band is this share of its height in from its edge, from Figure 1's rows of rivets
	veneraCableSteps   = 3       // picked: straight runs each cable over the module's shoulder is bent into
	veneraPipeOut      = 0.58    // picked: the pipes come down from under the plate this share of the way out to its rim, from Figure 1
	veneraPipeKnee     = 0.35    // picked: the pipes bend in toward the module this share of its radius above its middle, from Figure 1
	veneraBoomIn       = 0.85    // picked: each boom's head stands at this share of the ring's radius out, from Figure 1
	veneraHeadSides    = 8       // picked: corners along each boom head's curve; enough to read as round
	veneraLeadTurn     = 60      // picked: degrees round the can the aerial's lead winds on its way down to the plate, by eye from Figure 1, page 2
	veneraLeadSteps    = 4       // picked: straight runs the lead is bent into
	veneraAcross       = 0       // defined: a profile's corner says how far out first
	veneraUp           = 1       // defined: and how high second
	veneraRingSides    = 12      // picked: flat sides round the ring's tube; enough to read as round
	veneraModuleLevels = 48      // picked: bands from pole to pole on the module's sphere; at 16 the drag plate's shadow broke into teeth along them
	veneraRibsFrom     = 0.15    // picked: the aerial's turns start this share of the can's body up, by eye from the paper's Figure 1, page 2
	veneraRibShare     = 0.6     // picked: each turn of the aerial is this share of its pitch and the dark gap the rest, by eye from Figure 1, page 2
	veneraLidShare     = 0.8     // picked: the lid on the can's cap is this share of the core across, from the reference sheet
	veneraCapStep      = 0.4     // picked: the cap's flange is this share of the cap's height, the lid the rest, by eye from Figure 1, page 2
)

var veneraRead, veneraError = readVenera(veneraLanderRaw)

// readVenera reads the lander's sizes and checks them, so that making
// it afterwards cannot fail: every size above nought, three legs at
// least, the vessel inside the module, the plate and the ring inside
// the heat shield, the module off the ground, a coat for every part,
// and somewhere the sizes were read.
func readVenera(raw []byte) (veneraSizes, error) {
	var read veneraSizes
	if err := yamlfile.Read("venera-lander.yaml", raw, &read); err != nil {
		return veneraSizes{}, err
	}
	if err := read.sound(); err != nil {
		return veneraSizes{}, err
	}
	if err := read.fits(); err != nil {
		return veneraSizes{}, err
	}
	dressed, err := read.looks()
	if err != nil {
		return veneraSizes{}, err
	}
	read.dressed = dressed
	return read, nil
}

// lengths is every length the file gives, by name, for sound to check.
func (size veneraSizes) lengths() map[string]float64 {
	return map[string]float64{"shell": size.Shell,
		"ring radius": size.Ring.Radius, "ring tube": size.Ring.Tube,
		"ring wide": size.Ring.Wide, "ring hole": size.Ring.Hole,
		"skirt across": size.Skirt.Across, "skirt tall": size.Skirt.Tall,
		"skirt slat": size.Skirt.Slat, "tooth tall": size.Ring.Tooth.Tall,
		"tooth base":  size.Ring.Tooth.Base,
		"tooth thick": size.Ring.Tooth.Thick,
		"leg strut":   size.Legs.Strut, "leg boss": size.Legs.Boss,
		"foot wide": size.Legs.Foot.Wide, "foot tall": size.Legs.Foot.Tall,
		"foot deep": size.Legs.Foot.Deep, "module across": size.Module.Across,
		"module vessel": size.Module.Vessel, "module height": size.Module.Height,
		"module band": size.Module.Band, "module proud": size.Module.Proud,
		"module window":  size.Module.Window,
		"module housing": size.Module.Housing,
		"box wide":       size.Module.Box.Wide, "box tall": size.Module.Box.Tall,
		"box deep":     size.Module.Box.Deep,
		"plate across": size.Plate.Across, "plate thick": size.Plate.Thick,
		"plate dish": size.Plate.Dish, "plate lip": size.Plate.Lip,
		"plate rim": size.Plate.Rim, "plate hub": size.Plate.Hub,
		"plate stay": size.Plate.Stay, "plate seam": size.Plate.Seam,
		"bolt": size.Bolt, "cable": size.Cable, "pipe": size.Pipe,
		"boom strut": size.Booms.Strut, "boom head": size.Booms.Head,
		"boom girth": size.Booms.Girth,
		"can across": size.Can.Across, "can height": size.Can.Height,
		"can core": size.Can.Core, "can cap": size.Can.Cap,
		"arm length": size.Arm.Length, "arm wide": size.Arm.Wide,
		"arm rail": size.Arm.Rail, "caps across": size.Caps.Across,
		"caps deep": size.Caps.Deep, "caps thick": size.Caps.Thick,
		"chart wide": size.Chart.Wide, "chart tall": size.Chart.Tall,
		"chart post": size.Chart.Post, "chart thick": size.Chart.Thick,
		"mass": size.Mass}
}

// sound refuses a size at nought or below, too few legs or turns of the
// aerial, and sizes read nowhere, by the name of what is wrong.
func (size veneraSizes) sound() error {
	for name, each := range size.lengths() {
		if !sound(each) {
			return fmt.Errorf("manufacture: the venera lander's %s is %v",
				name, each)
		}
	}
	if size.Legs.Count < veneraLegsLeast ||
		size.Legs.Count%veneraLegsPerFoot != spaceZero {
		return fmt.Errorf("manufacture: the venera lander has %d legs, "+
			"and needs %d to stand, %d to each foot", size.Legs.Count,
			veneraLegsLeast, veneraLegsPerFoot)
	}
	if size.Ring.Teeth < spaceOne || size.Ring.Holes < spaceOne ||
		size.Skirt.Slats < spaceOne {
		return fmt.Errorf("manufacture: the venera lander's ring has %d "+
			"teeth and %d holes, and its skirt %d slats; each needs one",
			size.Ring.Teeth, size.Ring.Holes, size.Skirt.Slats)
	}
	if size.Can.Stiffeners < spaceOne || size.Can.Lugs < spaceOne {
		return fmt.Errorf("manufacture: the venera lander's can has %d "+
			"stiffeners and %d lugs, and needs one of each",
			size.Can.Stiffeners, size.Can.Lugs)
	}
	if size.Can.Ribs < veneraRibsLeast {
		return fmt.Errorf("manufacture: the venera lander's aerial has %d "+
			"can ribs, and needs %d", size.Can.Ribs, veneraRibsLeast)
	}
	if size.Bolts.Plate < spaceOne || size.Bolts.Band < spaceOne ||
		size.Bolts.Cap < spaceOne {
		return fmt.Errorf("manufacture: the venera lander's bolt rings "+
			"have %+v bolts, and each needs one", size.Bolts)
	}
	if size.Plate.Stays < veneraStaysLeast {
		return fmt.Errorf("manufacture: the venera lander's plate has %d "+
			"stays, and needs %d", size.Plate.Stays, veneraStaysLeast)
	}
	if size.Read == "" {
		return fmt.Errorf("manufacture: the venera lander's sizes were " +
			"read nowhere")
	}
	return nil
}

// fits refuses sizes that cannot be built together: a vessel no smaller
// than the module, a plate or a ring no smaller than the heat shield it
// rode in, a module that would be under the ground, and a can whose
// core or cap would not fit it.
func (size veneraSizes) fits() error {
	if size.Module.Vessel >= size.Module.Across {
		return fmt.Errorf("manufacture: the venera lander's vessel, %v "+
			"across, does not fit in its module, %v across",
			size.Module.Vessel, size.Module.Across)
	}
	if size.Plate.Across >= size.Shell {
		return fmt.Errorf("manufacture: the venera lander's plate, %v "+
			"across, does not fit in its heat shield, %v", size.Plate.Across,
			size.Shell)
	}
	if size.ringOut()*2 >= size.Shell {
		return fmt.Errorf("manufacture: the venera lander's ring, %v "+
			"across, does not fit in its heat shield, %v",
			size.ringOut()*2, size.Shell)
	}
	if size.moduleFoot() <= spaceZero {
		return fmt.Errorf("manufacture: the venera lander's module is "+
			"%v below the ground", -size.moduleFoot())
	}
	if err := size.plateFits(); err != nil {
		return err
	}
	if err := size.moduleFits(); err != nil {
		return err
	}
	if err := size.instrumentsFit(); err != nil {
		return err
	}
	return size.canFits()
}

// instrumentsFit refuses an arm whose rails are too thick to lie against
// the ring's wall on the ground, or with no rungs, and a lens cap deeper
// than half across, which is no cap, or thicker than it is deep.
func (size veneraSizes) instrumentsFit() error {
	if size.Arm.Rail >= size.Ring.Tube || size.Arm.Rungs < spaceOne {
		return fmt.Errorf("manufacture: the venera lander's arm, its rails "+
			"%v thick with %d rungs, cannot lie on the ground against the "+
			"ring, %v high", size.Arm.Rail, size.Arm.Rungs, size.Ring.Tube)
	}
	if size.Caps.Deep >= size.Caps.Across*spaceHalf ||
		size.Caps.Thick >= size.Caps.Deep {
		return fmt.Errorf("manufacture: the venera lander's lens caps, %v "+
			"across, %v deep and %v thick, cannot be made", size.Caps.Across,
			size.Caps.Deep, size.Caps.Thick)
	}
	return nil
}

// ringTop is how high the ring's top is.
func (size veneraSizes) ringTop() float64 {
	return size.Ring.Tube + size.Ring.Tube
}

// moduleFits refuses a camera window no narrower than the housing it is
// set in, a skirt as wide as the module, a ring hole as big as the
// ring's tube, too few fittings round the equator to make one box, and a box
// whose back would stand clear of the module, leaving a gap.
func (size veneraSizes) moduleFits() error {
	if size.Module.Window >= size.Module.Housing*(spaceOne-veneraChamfer) {
		return fmt.Errorf("manufacture: the venera lander's camera window, "+
			"%v across, does not fit its housing, %v", size.Module.Window,
			size.Module.Housing)
	}
	if size.Skirt.Across >= size.Module.Across ||
		size.Ring.Hole >= size.Ring.Tube {
		return fmt.Errorf("manufacture: the venera lander's skirt, %v "+
			"across, is no narrower than its module, or a ring hole, %v, "+
			"no smaller than the ring's tube", size.Skirt.Across,
			size.Ring.Hole)
	}
	if size.Module.Fittings < spaceOne {
		return fmt.Errorf("manufacture: the venera lander has %d fittings "+
			"round its equator, and needs one", size.Module.Fittings)
	}
	radius := size.Module.Across * spaceHalf
	back := math.Hypot(radius*veneraBoxInside, size.Module.Box.Wide*spaceHalf)
	top := size.boxFloor() - size.moduleMiddle() + size.Module.Box.Tall
	if math.Hypot(back, top) >= radius {
		return fmt.Errorf("manufacture: the venera lander's equipment box, "+
			"%v wide and %v tall, stands clear of the module",
			size.Module.Box.Wide, size.Module.Box.Tall)
	}
	return nil
}

// boxFloor is how high the boxes and fittings round the equator stand:
// on the band's top.
func (size veneraSizes) boxFloor() float64 {
	return size.moduleMiddle() + size.Module.Band*(spaceHalf-veneraBandDrop)
}

// plateFits refuses a plate whose lip is no taller than the plate is
// thick, whose rim reaches past its collar, or whose collar is no
// narrower than the module it stands on.
func (size veneraSizes) plateFits() error {
	if size.Plate.Lip <= size.Plate.Thick {
		return fmt.Errorf("manufacture: the venera lander's plate lip, %v, "+
			"stands no higher than the plate is thick, %v", size.Plate.Lip,
			size.Plate.Thick)
	}
	if size.Plate.Rim >= (size.Plate.Across-size.Plate.Hub)*spaceHalf {
		return fmt.Errorf("manufacture: the venera lander's plate rim, %v, "+
			"reaches past its collar", size.Plate.Rim)
	}
	if size.Plate.Hub >= size.Module.Across {
		return fmt.Errorf("manufacture: the venera lander's plate hub, %v "+
			"across, is no narrower than its module, %v", size.Plate.Hub,
			size.Module.Across)
	}
	return nil
}

// canFits refuses a can whose core is no narrower than the aerial wound
// round it, or whose cap is no shorter than the can.
func (size veneraSizes) canFits() error {
	if size.Can.Core >= size.Can.Across {
		return fmt.Errorf("manufacture: the venera lander's can core, %v "+
			"across, is no narrower than its aerial, %v", size.Can.Core,
			size.Can.Across)
	}
	if size.Can.Cap >= size.Can.Height {
		return fmt.Errorf("manufacture: the venera lander's can cap, %v "+
			"high, does not fit on its can, %v", size.Can.Cap,
			size.Can.Height)
	}
	return nil
}

// moduleMiddle is how high the module's middle is: the ring's height
// and the payload height above it.
func (size veneraSizes) moduleMiddle() float64 {
	return size.Ring.Tube + size.Module.Height
}

// moduleFoot is how high the module's underside is off the ground.
func (size veneraSizes) moduleFoot() float64 {
	return size.moduleMiddle() - size.Module.Across*spaceHalf
}

// moduleTop is how high the module's top is, where the plate rests.
func (size veneraSizes) moduleTop() float64 {
	return size.moduleMiddle() + size.Module.Across*spaceHalf
}

// plateTop is how high the plate's top is, where the can stands.
func (size veneraSizes) plateTop() float64 {
	return size.moduleTop() + size.Plate.Thick
}

// overall is how high the lander is: the top of the can.
func (size veneraSizes) overall() float64 {
	return size.plateTop() + size.Can.Height
}

// veneraLander is the maker of the lander.
type veneraLander struct{}

// veneraRows supplies the lander, by name. A sizes file that cannot be
// read stops the build, as the other embedded data files do.
func veneraRows() map[string]maker {
	if veneraError != nil {
		panic(veneraError)
	}
	return map[string]maker{"venera-lander": veneraLander{}}
}

// looks is what the lander wears: a matte coat on every part, no sheen
// and no mirror, each of the colour the sizes file gives it. A part
// with no coat there, or a coat outside nought to one, is refused.
func (size veneraSizes) looks() (looks, error) {
	dressed := looks{}
	for where, name := range veneraNames {
		colour, given := size.Coats[name]
		if !given {
			return nil, fmt.Errorf("manufacture: the venera lander's %s "+
				"has no coat", name)
		}
		coat, err := veneraCoat(name, colour)
		if err != nil {
			return nil, err
		}
		dressed[where] = coat
	}
	return dressed, nil
}

// veneraCoat is one part of the lander in a matte coat of one colour
// over titanium: paint has no gloss, so nothing on it mirrors.
func veneraCoat(name string, colour veneraColour) (*thing.Material, error) {
	if !share(colour.Red) || !share(colour.Green) || !share(colour.Blue) {
		return nil, fmt.Errorf("manufacture: the venera lander's %s coat "+
			"is %+v, outside nought to one", name, colour)
	}
	kind, err := materials.ByName("titanium")
	if err != nil {
		return nil, fmt.Errorf("manufacture: the venera lander's %s: %w", name, err)
	}
	paint := maths.RGB(colour.Red, colour.Green, colour.Blue)
	coat, err := thing.Made(kind, thing.Painted(paint))
	if err != nil {
		return nil, fmt.Errorf("manufacture: the venera lander's %s: %w", name, err)
	}
	coat.Name = name
	return coat, nil
}

// extent is the lander's box: as wide as whatever reaches furthest out,
// the plate, the arm on the ground or the lens caps, and as tall as the
// top of the can.
func (size veneraSizes) extent() maths.Box {
	_, run := size.armRun()
	arm := size.ringOut() + math.Hypot(run.X, run.Z) + size.Arm.Wide*spaceHalf
	caps := size.ringOut() + veneraCapsOut + size.Caps.Across*spaceHalf
	out := math.Max(math.Max(size.Plate.Across*spaceHalf, arm), caps)
	return maths.SegmentBox(maths.V(-out, spaceZero, -out),
		maths.V(out, size.overall(), out), spaceZero)
}

// ringOut is how far the ring's outer edge is from the lander's middle.
func (size veneraSizes) ringOut() float64 {
	return size.Ring.Radius + size.Ring.Wide*spaceHalf
}

// ringProfile is the ring's tube cut across: a closed loop of corners
// round the tube, counterclockwise as seen from outside the ring, out
// from the ring's own middle. The tube is wider than it is tall, an
// ellipse, as the ring is a flat band in Figure 1. The ring is the loop
// turned about y, lying flat.
func (size veneraSizes) ringProfile() [][coordinatePair]float64 {
	var loop [][coordinatePair]float64
	for corner := range veneraRingSides {
		turn := fullTurn * float64(corner) / veneraRingSides
		loop = append(loop, size.ringAt(turn))
	}
	return loop
}

// ringAt is the corner of the ring's cut so far round its tube from its
// outer edge, out from the lander's middle and up from the tube's own.
func (size veneraSizes) ringAt(turn float64) [coordinatePair]float64 {
	return [coordinatePair]float64{
		size.Ring.Radius + size.Ring.Wide*spaceHalf*math.Cos(turn),
		size.Ring.Tube * math.Sin(turn)}
}

// moduleProfile is the module's sphere from its underside to its top,
// in bands. The poles are said exactly: sin(pi)'s tiny residue is not
// a real ring.
func (size veneraSizes) moduleProfile() [][coordinatePair]float64 {
	radius := size.Module.Across * spaceHalf
	var side [][coordinatePair]float64
	for band := range veneraModuleLevels + spaceOne {
		turn := math.Pi * float64(band) / veneraModuleLevels
		reach := radius * math.Sin(turn)
		if band == spaceZero || band == veneraModuleLevels {
			reach = spaceZero
		}
		side = append(side, [coordinatePair]float64{reach, -radius * math.Cos(turn)})
	}
	return side
}

// legEnds is where one leg runs from and to. The legs stand three to a
// foot on the ring's middle circle: the middle one of each three runs
// straight in toward the module's middle, and the other two splay to
// either side, so the legs meet the module evenly round it, all at one
// height.
//
// That height is picked: where the line from the ring's middle circle to
// the module's middle meets the module, so every leg leaves the ring upward
// and never crosses the ring's tube or the ground.
func (size veneraSizes) legEnds(leg int) (maths.Vec, maths.Vec) {
	feet := size.Legs.Count / veneraLegsPerFoot
	foot := veneraOut(fullTurn * float64(leg/veneraLegsPerFoot) /
		float64(feet))
	from := foot.Scale(size.Ring.Radius).Add(maths.Up.Scale(size.Ring.Tube))
	// the middle leg of each three has the foot's own turn
	turn := fullTurn * float64(leg-spaceOne) / float64(size.Legs.Count)
	radius := size.Module.Across * spaceHalf
	slope := maths.V(size.Ring.Radius, size.Ring.Tube-size.moduleMiddle(),
		spaceZero).Unit()
	to := veneraOut(turn).Scale(radius * slope.X).Add(maths.Up.Scale(
		size.moduleMiddle() + radius*slope.Y))
	return from, to
}

// made keeps quick as the lander's default detail.
func (row veneraLander) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt builds the lander: the ring, flat on the ground; the module, a
// sphere above it; the legs, one straight strut each from the ring to
// the module; the plate, a flat disc on the module's top; and the can
// with its aerial on the plate. The vessel, the shell and the mass are
// checked and not drawn.
//
// Its parts of one flat look that repeat, the bolts and holes and legs,
// are kept as shared mesh groups, each distinct part stored once: the
// demo's bolts alone are most of its faces, and cut finely they would
// not flatten. Its rounded looks stay parts, shaded smooth, which a
// group does not hold.
func (row veneraLander) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	size := veneraRead
	// each lander made has its own copy of each look
	dressed := looks{}
	for where, look := range size.dressed {
		own := *look
		dressed[where] = &own
	}
	assembly := size.stood(detail)
	assembly.object.dressed = dressed
	if err := assembly.shareRepeats(); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	object, err := assembly.finish(size.extent())
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// stood is the lander's assembly at a detail with every part placed, as
// drawn, and none yet shared.
func (size veneraSizes) stood(detail string) *instrumentAssembly {
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: looks{}}}
	size.stand(assembly)
	return assembly
}

// veneraAt is the lander's own space lifted to a height.
func veneraAt(height float64) maths.Subspace {
	return maths.NewSubspace(maths.V(spaceZero, height, spaceZero),
		maths.Identity, spaceOne)
}

// veneraTurned is a closed profile turned about y whose corners carry
// the round surface's normal, so a coarse mesh shades as one smooth
// thing and not as flat facets with teeth between them.
func veneraTurned(points [][coordinatePair]float64) form {
	return roundProfile{turnedProfile{points: points}}
}

// veneraBlock is a ring, or a disc when inner is nought, cut across: a
// closed loop of corners from inner to outer and from low to high,
// counterclockwise as seen from outside, so its faces look outward.
func veneraBlock(inner, outer, low, high float64) [][coordinatePair]float64 {
	return [][coordinatePair]float64{{inner, low}, {outer, low}, {outer, high},
		{inner, high}}
}

// stand puts every drawn part into the assembly, bottom to top.
func (size veneraSizes) stand(assembly *instrumentAssembly) {
	assembly.add(veneraTurned(size.ringProfile()),
		veneraAt(size.Ring.Tube), veneraRing)
	assembly.add(veneraTurned(size.moduleProfile()),
		veneraAt(size.moduleMiddle()), veneraModule)
	size.landingGear(assembly)
	size.ringTeeth(assembly)
	size.ringHoles(assembly)
	size.moduleSkirt(assembly)
	size.moduleBand(assembly)
	size.cameraWindows(assembly)
	size.equipment(assembly)
	size.dragPlate(assembly)
	size.parachuteCan(assembly)
	size.springArm(assembly)
	size.lensCaps(assembly)
	size.colourChart(assembly)
	size.plateSeams(assembly)
	size.boltRings(assembly)
	size.cableRuns(assembly)
	size.pipes(assembly)
	size.booms(assembly)
	size.canDetail(assembly)
}

// canDetail adds what Figure 1 shows on the can besides the aerial's
// turns: upright stiffeners on the core, showing between the turns; the
// aerial's lead, winding down from its lowest turn to the plate; and
// lugs standing on the cap's flange round its lid.
func (size veneraSizes) canDetail(assembly *instrumentAssembly) {
	foot := size.plateTop()
	core, out := size.Can.Core*spaceHalf, size.Can.Across*spaceHalf
	body := size.Can.Height - size.Can.Cap
	stiff := (out - core) * spaceHalf
	for at := range size.Can.Stiffeners {
		way := veneraOut(fullTurn * float64(at) / float64(size.Can.Stiffeners))
		assembly.add(boxShape(stiff, body, stiff), maths.NewSubspace(
			way.Scale(core).Add(maths.Up.Scale(foot)), veneraFacing(way),
			spaceOne), veneraCan)
	}
	start := body * veneraRibsFrom
	rib := (body - start) / float64(size.Can.Ribs) * veneraRibShare
	lead := measure.Degrees(veneraLeadTurn).InRadians()
	point := func(step int) maths.Vec {
		share := float64(step) / veneraLeadSteps
		return veneraOut(-lead * share).Scale((core + out) * spaceHalf).Add(
			maths.Up.Scale(foot + rib*spaceHalf + (start)*(spaceOne-share)))
	}
	for step := range veneraLeadSteps {
		assembly.rod(point(step), point(step+spaceOne), rib*spaceHalf,
			veneraAerial)
	}
	lid := core * veneraLidShare
	flange := foot + body + size.Can.Cap*veneraCapStep
	for at := range size.Can.Lugs {
		way := veneraOut(fullTurn * (float64(at) + spaceHalf) /
			float64(size.Can.Lugs))
		assembly.add(boxShape(stiff, size.Can.Cap*(spaceOne-veneraCapStep),
			out-lid), maths.NewSubspace(way.Scale((out+lid)*spaceHalf).Add(
			maths.Up.Scale(flange)), veneraFacing(way), spaceOne), veneraCan)
	}
}

// veneraFacing is the turn that faces a part's z out along way, its y
// up and its x along the circle round the lander's axis.
func veneraFacing(way maths.Vec) maths.Orientation {
	return maths.Orientation{X: maths.Up.Cross(way), Y: maths.Up, Z: way}
}

// veneraTorus is a thin ring cut across: a circle of the given radius
// so far out from the axis, counterclockwise as seen from outside.
func veneraTorus(around, radius float64) [][coordinatePair]float64 {
	var loop [][coordinatePair]float64
	for corner := range veneraRingSides {
		turn := fullTurn * float64(corner) / veneraRingSides
		loop = append(loop, [coordinatePair]float64{
			around + radius*math.Cos(turn), radius * math.Sin(turn)})
	}
	return loop
}

// cableRuns lays a cable round the module just above its fittings, and
// from each round fitting a cable up over the module's shoulder to the
// collar under the plate, bent in straight runs along the surface.
func (size veneraSizes) cableRuns(assembly *instrumentAssembly) {
	radius := size.Module.Across * spaceHalf
	cable := size.Cable
	loop := size.boxFloor() + size.Module.Box.Tall + cable -
		size.moduleMiddle()
	assembly.add(veneraTurned(veneraTorus(math.Sqrt(radius*radius-
		loop*loop)+cable, cable)), veneraAt(size.moduleMiddle()+loop),
		veneraCable)
	low := math.Asin(loop / radius)
	high := math.Acos(size.Plate.Hub * spaceHalf / radius)
	over := radius + cable
	for at := range size.Module.Fittings {
		if at%veneraBoxEvery == spaceZero {
			continue // the boxes carry no cable up
		}
		way := veneraOut(fullTurn * (float64(at) + spaceHalf) /
			float64(size.Module.Fittings))
		point := func(step int) maths.Vec {
			rise := low + (high-low)*float64(step)/veneraCableSteps
			return way.Scale(over * math.Cos(rise)).Add(maths.Up.Scale(
				size.moduleMiddle() + over*math.Sin(rise)))
		}
		for step := range veneraCableSteps {
			assembly.rod(point(step), point(step+spaceOne), cable, veneraCable)
		}
	}
}

// pipes brings the two pipes down from under the plate beside the
// module, as Figure 1 shows them: straight down, then bent in to the
// module just above its fittings.
func (size veneraSizes) pipes(assembly *instrumentAssembly) {
	radius := size.Module.Across * spaceHalf
	out := size.Plate.Across * spaceHalf * veneraPipeOut
	knee := size.moduleMiddle() + radius*veneraPipeKnee
	foot := size.boxFloor() + size.Module.Box.Tall + size.Pipe
	for _, degrees := range veneraPipeTurns {
		way := veneraOut(measure.Degrees(degrees).InRadians())
		top := way.Scale(out).Add(maths.Up.Scale(size.plateUnder(out)))
		bend := way.Scale(out).Add(maths.Up.Scale(knee))
		end := way.Scale(radius * veneraBoxInside).Add(maths.Up.Scale(foot))
		assembly.rod(top, bend, size.Pipe, veneraPipe)
		assembly.rod(bend, end, size.Pipe, veneraPipe)
	}
}

// booms raises the two booms from the ring's top that Figure 1 shows,
// each leaning in, with an egg-shaped head at its end. What they carried
// is not in the sources at hand; their shape is Figure 1's.
func (size veneraSizes) booms(assembly *instrumentAssembly) {
	boom := size.Booms
	var head [][coordinatePair]float64
	for corner := range veneraHeadSides + spaceOne {
		turn := math.Pi * float64(corner) / veneraHeadSides
		head = append(head, [coordinatePair]float64{
			boom.Girth * spaceHalf * math.Sin(turn),
			boom.Head * spaceHalf * (spaceOne - math.Cos(turn))})
	}
	head[spaceZero][veneraAcross] = spaceZero
	head[veneraHeadSides][veneraAcross] = spaceZero
	for _, degrees := range veneraBoomTurns {
		way := veneraOut(measure.Degrees(degrees).InRadians())
		from := way.Scale(size.Ring.Radius).Add(maths.Up.Scale(size.ringTop()))
		to := way.Scale(size.Ring.Radius * veneraBoomIn).Add(maths.Up.Scale(
			size.moduleFoot() + boom.Head))
		assembly.rod(from, to, boom.Strut, veneraLeg)
		along := to.Sub(from).Unit()
		assembly.turnedAlong(veneraTurned(head), to.Sub(along.Scale(
			boom.Head*spaceHalf)), along, veneraHead)
	}
}

// plateSeams raises the seams on the plate's top where it was joined:
// rings round it, and a straight seam out from the collar to the lip
// over each stay.
func (size veneraSizes) plateSeams(assembly *instrumentAssembly) {
	seam := size.Plate.Seam
	rim := size.Plate.Across * spaceHalf
	for _, share := range veneraSeams {
		out := rim * share
		assembly.add(veneraTurned(veneraBlock(out-seam*spaceHalf,
			out+seam*spaceHalf, spaceZero, seam)), veneraAt(
			size.plateUnder(out)+size.Plate.Thick-seam*spaceHalf), veneraPlate)
	}
	inner, outer := size.Plate.Hub*spaceHalf, rim-size.Plate.Rim
	for at := range size.Plate.Stays {
		way := veneraOut(fullTurn * (float64(at) + spaceHalf) /
			float64(size.Plate.Stays))
		from := way.Scale(inner).Add(maths.Up.Scale(size.plateUnder(inner) +
			size.Plate.Thick))
		to := way.Scale(outer).Add(maths.Up.Scale(size.plateUnder(outer) +
			size.Plate.Thick))
		assembly.rod(from, to, seam*spaceHalf, veneraPlate)
	}
}

// boltRings sets the rows of bolts: round the top of the plate's lip,
// along both edges of the module's band, as Figure 1 shows rivets round
// the body, and round the flange of the can's cap.
func (size veneraSizes) boltRings(assembly *instrumentAssembly) {
	rim := size.Plate.Across * spaceHalf
	size.boltRing(assembly, size.Bolts.Plate, rim-size.Plate.Rim*spaceHalf,
		size.moduleTop()+size.Plate.Dish+size.Plate.Lip, true)
	wall := size.Module.Across*spaceHalf + size.Module.Proud
	low := size.boxFloor() - size.Module.Band
	inset := size.Module.Band * veneraRivetInset
	for _, high := range []float64{low + inset, size.boxFloor() - inset} {
		size.boltRing(assembly, size.Bolts.Band, wall, high, false)
	}
	flange := (size.Can.Across + size.Can.Core*veneraLidShare) *
		spaceHalf * spaceHalf
	size.boltRing(assembly, size.Bolts.Cap, flange, size.overall()-
		size.Can.Cap+size.Can.Cap*veneraCapStep, true)
}

// boltRing sets so many bolt heads in a ring round the lander's axis, so
// far out and so high: each a short drum sunk half into what it holds,
// standing up when up is true and out from the axis when it is not.
func (size veneraSizes) boltRing(assembly *instrumentAssembly, count int,
	out, high float64, up bool,
) {
	bolt := size.Bolt
	for at := range count {
		way := veneraOut(fullTurn * (float64(at) + spaceHalf) / float64(count))
		axis := way
		if up {
			axis = maths.Up
		}
		centre := way.Scale(out).Add(maths.Up.Scale(high))
		assembly.rodAlong(centre.Sub(axis.Scale(bolt*spaceHalf)),
			axis.Scale(bolt), bolt, veneraBolt)
	}
}

// armRun is where the arm's near end lies, on the ground against the
// ring's wall, its rails' middles one rail's radius up so they rest on
// the ground, and the level run from there out to its far end.
func (size veneraSizes) armRun() (maths.Vec, maths.Vec) {
	out := veneraOut(measure.Degrees(veneraArmTurn).InRadians())
	// the ring's wall at the rails' height, from its cut
	low := math.Asin((size.Arm.Rail - size.Ring.Tube) / size.Ring.Tube)
	near := size.ringAt(low)[veneraAcross]
	hinge := out.Scale(near).Add(maths.Up.Scale(size.Arm.Rail))
	return hinge, out.Scale(size.Arm.Length)
}

// springArm lays the spring-loaded arm flat on the ground, deployed out
// from the ring and pointing away from the lander, as Venera 13's
// photographs show it after landing: a ladder of two rails with rungs
// between, the scale the camera read, a hinge across its near end at
// the ring's wall and a round pad at its far end. It is an instrument
// lying on the ground, not steps.
func (size veneraSizes) springArm(assembly *instrumentAssembly) {
	arm := size.Arm
	hinge, run := size.armRun()
	along := maths.Up.Cross(maths.V(run.X, spaceZero, run.Z).Unit())
	half := along.Scale(arm.Wide * spaceHalf)
	for _, side := range []maths.Vec{half, half.Neg()} {
		assembly.rodAlong(hinge.Add(side), run, arm.Rail, veneraArm)
	}
	for rung := range arm.Rungs {
		share := (float64(rung) + spaceOne) / (float64(arm.Rungs) + spaceOne)
		assembly.rodAlong(hinge.Add(run.Scale(share)).Sub(half),
			along.Scale(arm.Wide), arm.Rail*spaceHalf, veneraArm)
	}
	assembly.rodAlong(hinge.Sub(half).Sub(along.Scale(arm.Rail)),
		along.Scale(arm.Wide+arm.Rail+arm.Rail), arm.Rail, veneraArm)
	end := hinge.Add(run)
	assembly.add(veneraTurned(veneraBlock(spaceZero, arm.Wide*spaceHalf,
		spaceZero, arm.Rail+arm.Rail)), maths.NewSubspace(maths.V(end.X,
		spaceZero, end.Z), maths.Identity, spaceOne), veneraArm)
}

// lensCaps lays the two lens caps on the ground beyond the ring, each
// to one side of its camera and tipped on its rim, where the cameras
// threw them off: white curved shells, as Venera 13 photographed one.
func (size veneraSizes) lensCaps(assembly *instrumentAssembly) {
	shell := veneraTurned(size.capProfile())
	tilt := measure.Degrees(veneraCapTilt).InRadians()
	aside := measure.Degrees(veneraCapTurn).InRadians()
	for camera := range veneraWindows {
		out := veneraOut(fullTurn*(float64(camera)+spaceHalf)/veneraWindows +
			aside)
		lean := maths.Up.Scale(math.Cos(tilt)).Add(
			maths.Up.Cross(out).Scale(math.Sin(tilt)))
		assembly.turnedAlong(shell, out.Scale(size.ringOut()+veneraCapsOut),
			lean, veneraCap)
	}
}

// capProfile is a lens cap cut across: a thin shell of a sphere, its
// outside from the rim up to the top and its inside back down to the
// rim, counterclockwise as seen from outside.
func (size veneraSizes) capProfile() [][coordinatePair]float64 {
	caps := size.Caps
	rim := caps.Across * spaceHalf
	sphere := (rim*rim + caps.Deep*caps.Deep) / (caps.Deep + caps.Deep)
	widest := math.Asin(rim / sphere)
	centre := caps.Deep - sphere
	var outside, inside [][coordinatePair]float64
	for step := range veneraCapSteps + spaceOne {
		down := widest * float64(veneraCapSteps-step) / veneraCapSteps
		up := widest * float64(step) / veneraCapSteps
		outside = append(outside, [coordinatePair]float64{
			sphere * math.Sin(down), centre + sphere*math.Cos(down)})
		inside = append(inside, [coordinatePair]float64{
			(sphere - caps.Thick) * math.Sin(up),
			centre + (sphere-caps.Thick)*math.Cos(up)})
	}
	return append(outside, inside...)
}

// colourChart stands the colour chart on two posts on the ring's top: a
// panel leaning back to face up and in toward the camera above it, with
// a row of patches, white, red, green and blue, for the camera to be
// read against.
func (size veneraSizes) colourChart(assembly *instrumentAssembly) {
	chart := size.Chart
	out := veneraOut(measure.Degrees(veneraChartTurn).InRadians())
	along := maths.Up.Cross(out)
	lean := measure.Degrees(veneraChartLean).InRadians()
	rise := out.Scale(math.Sin(lean)).Add(maths.Up.Scale(math.Cos(lean)))
	face := maths.Up.Scale(math.Sin(lean)).Sub(out.Scale(math.Cos(lean)))
	facing := maths.Orientation{X: along.Neg(), Y: rise, Z: face}
	middle := out.Scale(size.Ring.Radius)
	foot := middle.Add(maths.Up.Scale(size.ringTop() + chart.Post))
	for _, side := range []float64{-spaceHalf, spaceHalf} {
		assembly.rodAlong(middle.Add(along.Scale(side*chart.Wide*
			veneraPatchShare)).Add(maths.Up.Scale(size.Ring.Tube)),
			maths.Up.Scale(foot.Y-size.Ring.Tube), chart.Thick, veneraChart)
	}
	assembly.add(boxShape(chart.Wide, chart.Tall, chart.Thick),
		maths.NewSubspace(foot, facing, spaceOne), veneraChart)
	place := chart.Wide / float64(len(veneraPatches))
	for index, colour := range veneraPatches {
		across := chart.Wide*spaceHalf - (float64(index)+spaceHalf)*place
		centre := foot.Add(along.Scale(across)).Add(rise.Scale(
			chart.Tall * (spaceOne - veneraPatchShare) * spaceHalf)).Add(
			face.Scale(chart.Thick * spaceHalf))
		assembly.add(boxShape(place*veneraPatchShare,
			chart.Tall*veneraPatchShare, chart.Thick), maths.NewSubspace(
			centre, facing, spaceOne), colour)
	}
}

// veneraOut is the way out from the lander's middle at a turn round it,
// from north through east.
func veneraOut(turn float64) maths.Vec {
	return maths.V(math.Sin(turn), spaceZero, -math.Cos(turn))
}

// turnedAlong lays a form turned about its own y from a point along a
// run, its y along the run, keeping the first error as rod does.
func (assembly *instrumentAssembly) turnedAlong(shape form, from,
	run maths.Vec, material slot,
) {
	space, err := spaceAlong(from, run)
	if err != nil {
		assembly.trouble = err
		return
	}
	assembly.add(shape, space, material)
}

// landingGear stands the legs, three to a foot, from the ring to the
// module: at each foot a bracket straddling the ring's top, where its
// three legs meet, and at each leg's top a round boss on the module,
// sunk into it as the boxes are.
func (size veneraSizes) landingGear(assembly *instrumentAssembly) {
	radius := size.Module.Across * spaceHalf
	middle := maths.Up.Scale(size.moduleMiddle())
	for leg := range size.Legs.Count {
		from, to := size.legEnds(leg)
		assembly.rod(from, to, size.Legs.Strut*spaceHalf, veneraLeg)
		out := to.Sub(middle).Unit()
		sunk := radius * veneraBoxInside
		assembly.rodAlong(middle.Add(out.Scale(sunk)),
			out.Scale(radius+size.Legs.Boss-sunk), size.Legs.Strut, veneraLeg)
	}
	foot := size.Legs.Foot
	floor := size.Ring.Tube + size.Ring.Tube - foot.Tall*spaceHalf
	for at := range size.Legs.Count / veneraLegsPerFoot {
		from, _ := size.legEnds(at * veneraLegsPerFoot)
		out := maths.V(from.X, spaceZero, from.Z).Unit()
		facing := maths.Orientation{X: maths.Up.Cross(out), Y: maths.Up,
			Z: out}
		assembly.add(boxShape(foot.Wide, foot.Tall, foot.Deep),
			maths.NewSubspace(out.Scale(size.Ring.Radius).Add(
				maths.Up.Scale(floor)), facing, spaceOne), veneraLeg)
	}
}

// ringTeeth stands the row of triangular teeth round the ring's outer
// edge that Venera 13's photographs show: thin plates, each upright and
// facing out, their feet sunk into the ring's top.
func (size veneraSizes) ringTeeth(assembly *instrumentAssembly) {
	tooth := size.Ring.Tooth
	edge := size.ringAt(measure.Degrees(veneraToothAt).InRadians())
	foot := size.Ring.Tube + edge[veneraUp] - tooth.Tall*veneraToothSink
	// a triangle round its own middle, its point toward minus z, which
	// the facing below turns upward
	plate := instrumentPlate{thickness: tooth.Thick, outline: []maths.Vec{
		maths.V(spaceZero, spaceZero, -tooth.Tall*(spaceOne-veneraThird)),
		maths.V(tooth.Base*spaceHalf, spaceZero, tooth.Tall*veneraThird),
		maths.V(-tooth.Base*spaceHalf, spaceZero, tooth.Tall*veneraThird)}}
	for at := range size.Ring.Teeth {
		out := veneraOut(fullTurn * float64(at) / float64(size.Ring.Teeth))
		along := maths.Up.Cross(out)
		facing := maths.Orientation{X: along, Y: out, Z: along.Cross(out)}
		centre := out.Scale(edge[veneraAcross] - tooth.Thick*spaceHalf).Add(
			maths.Up.Scale(foot + tooth.Tall*veneraThird))
		assembly.add(plate, maths.NewSubspace(centre, facing, spaceOne),
			veneraRing)
	}
}

// ringHoles sets the row of holes round the ring's outer wall that let
// the air in and out as it crushed, as the report says (page 3) and
// Figure 1 shows: each a dark drum sunk into the wall at the tube's
// middle, its face just proud of the wall, standing for the hole.
func (size veneraSizes) ringHoles(assembly *instrumentAssembly) {
	hole := size.Ring.Hole
	for at := range size.Ring.Holes {
		out := veneraOut(fullTurn * (float64(at) + spaceHalf) /
			float64(size.Ring.Holes))
		from := out.Scale(size.ringOut() - hole).Add(
			maths.Up.Scale(size.Ring.Tube))
		assembly.rodAlong(from, out.Scale(hole+hole*veneraHoleProud), hole,
			veneraHole)
	}
}

// moduleSkirt rings the module's underside with the dark skirt of
// upright slats that Figure 1 shows between the module and the legs,
// centred on the height where the legs meet the module: a short drum
// reaching into the module, with the slats standing on its outside.
func (size veneraSizes) moduleSkirt(assembly *instrumentAssembly) {
	skirt := size.Skirt
	_, top := size.legEnds(spaceOne)
	low := top.Y - skirt.Tall*spaceHalf
	out := skirt.Across * spaceHalf
	assembly.add(veneraTurned(veneraBlock(size.Module.Across*spaceHalf*
		veneraSkirtInside, out, spaceZero, skirt.Tall)), veneraAt(low),
		veneraSkirt)
	for at := range skirt.Slats {
		way := veneraOut(fullTurn * float64(at) / float64(skirt.Slats))
		facing := maths.Orientation{X: maths.Up.Cross(way), Y: maths.Up,
			Z: way}
		assembly.add(boxShape(skirt.Slat, skirt.Tall, skirt.Slat),
			maths.NewSubspace(way.Scale(out).Add(maths.Up.Scale(low)),
				facing, spaceOne), veneraSkirt)
	}
}

// moduleBand paints the band round the module's middle, a little below
// it, standing proud of the insulation: where the museum models carry
// their lettering, drawn here as a darker band with none.
func (size veneraSizes) moduleBand(assembly *instrumentAssembly) {
	radius := size.Module.Across * spaceHalf
	low := size.boxFloor() - size.Module.Band
	assembly.add(veneraTurned(veneraBlock(radius*veneraBandInside,
		radius+size.Module.Proud, spaceZero, size.Module.Band)),
		veneraAt(low), veneraBand)
}

// cameraWindows sets the two camera housings in the module's upper half,
// on opposite sides, each looking out and down toward the ring, with a
// dark window on its end. Each housing is sunk into the module by its
// own radius, so no gap shows round its foot.
func (size veneraSizes) cameraWindows(assembly *instrumentAssembly) {
	radius := size.Module.Across * spaceHalf
	rise := measure.Degrees(veneraWindowRise).InRadians()
	tilt := measure.Degrees(veneraWindowTilt).InRadians()
	housing := size.Module.Housing * spaceHalf
	length := housing + size.Module.Housing*veneraHousingOut
	bevel := housing * veneraChamfer
	glass := size.Module.Window * veneraGlass
	for camera := range veneraWindows {
		out := veneraOut(fullTurn * (float64(camera) + spaceHalf) /
			veneraWindows)
		surface := out.Scale(radius * math.Cos(rise)).Add(maths.Up.Scale(
			size.moduleMiddle() + radius*math.Sin(rise)))
		look := out.Scale(math.Cos(tilt)).Sub(maths.Up.Scale(math.Sin(tilt)))
		foot := surface.Sub(look.Scale(housing))
		assembly.turnedAlong(veneraTurned([][coordinatePair]float64{
			{spaceZero, spaceZero}, {housing, spaceZero},
			{housing, length - bevel}, {housing - bevel, length},
			{spaceZero, length}}), foot, look, veneraModule)
		assembly.turnedAlong(veneraTurned(veneraBlock(spaceZero,
			size.Module.Window*spaceHalf, length-glass, length+glass)),
			foot, look, veneraWindow)
	}
}

// equipment sets the fittings round the module's equator, standing on
// the band: every other one a box, its back sunk into the module, and
// the rest round fittings, short drums out of the module. The cameras'
// housings stand above them.
func (size veneraSizes) equipment(assembly *instrumentAssembly) {
	radius := size.Module.Across * spaceHalf
	box := size.Module.Box
	inner := radius * veneraBoxInside
	for at := range size.Module.Fittings {
		out := veneraOut(fullTurn * (float64(at) + spaceHalf) /
			float64(size.Module.Fittings))
		if at%veneraBoxEvery == spaceZero {
			facing := maths.Orientation{X: maths.Up.Cross(out), Y: maths.Up,
				Z: out}
			assembly.add(boxShape(box.Wide, box.Tall, box.Deep),
				maths.NewSubspace(out.Scale(inner+box.Deep*spaceHalf).Add(
					maths.Up.Scale(size.boxFloor())), facing, spaceOne),
				veneraBox)
			continue
		}
		centre := out.Scale(inner).Add(maths.Up.Scale(size.boxFloor() +
			box.Tall*spaceHalf))
		assembly.rodAlong(centre, out.Scale(box.Deep),
			box.Tall*veneraFittingShare, veneraBox)
	}
}

// dragPlate puts the drag plate on the module's top: a wide disc a
// little dished, its rim higher than its middle, with a lip standing up
// round the rim; under its middle a collar down into the module; and
// stays from the module's shoulder out under the plate.
func (size veneraSizes) dragPlate(assembly *instrumentAssembly) {
	assembly.add(veneraTurned(size.plateProfile()),
		veneraAt(size.moduleTop()), veneraPlate)
	assembly.add(veneraTurned(veneraBlock(spaceZero,
		size.Plate.Hub*spaceHalf, size.moduleMiddle(), size.moduleTop())),
		maths.Local(), veneraModule)
	radius := size.Module.Across * spaceHalf
	rise := measure.Degrees(veneraStayRise).InRadians()
	reach := size.Plate.Across * spaceHalf * veneraStayReach
	for stay := range size.Plate.Stays {
		turn := fullTurn * (float64(stay) + spaceHalf) /
			float64(size.Plate.Stays)
		out := maths.V(math.Sin(turn), spaceZero, -math.Cos(turn))
		from := out.Scale(radius * math.Cos(rise)).Add(maths.Up.Scale(
			size.moduleMiddle() + radius*math.Sin(rise)))
		to := out.Scale(reach).Add(maths.Up.Scale(size.plateUnder(reach) -
			size.Plate.Stay*spaceHalf))
		assembly.rod(from, to, size.Plate.Stay*spaceHalf, veneraStay)
	}
}

// plateUnder is how high the plate's underside is so far out from its
// middle: it rises evenly from the module's top to the dish at the rim.
func (size veneraSizes) plateUnder(out float64) float64 {
	return size.moduleTop() +
		size.Plate.Dish*out/(size.Plate.Across*spaceHalf)
}

// plateProfile is the plate cut across from its middle out, as high as
// the module's top: its underside rising to the dish at the rim, up the
// lip's outside, in across the lip's top, down to the plate's top and
// back in to the middle, counterclockwise as seen from outside.
func (size veneraSizes) plateProfile() [][coordinatePair]float64 {
	rim := size.Plate.Across * spaceHalf
	inside := rim - size.Plate.Rim
	lip := size.Plate.Dish + size.Plate.Lip
	return [][coordinatePair]float64{{spaceZero, spaceZero},
		{rim, size.Plate.Dish}, {rim, lip}, {inside, lip},
		{inside, size.plateUnder(inside) - size.moduleTop() +
			size.Plate.Thick}, {spaceZero, size.Plate.Thick}}
}

// parachuteCan stands the can the parachutes rode in on the plate: a
// dark core, the aerial wound round it, drawn as a stack of flat ribs
// the way the reference sheet's models show it, and on top a cap whose
// flange is as wide as the aerial, with a narrower lid on it.
func (size veneraSizes) parachuteCan(assembly *instrumentAssembly) {
	foot := size.plateTop()
	core := size.Can.Core * spaceHalf
	out := size.Can.Across * spaceHalf
	body := size.Can.Height - size.Can.Cap
	assembly.add(veneraTurned(veneraBlock(spaceZero, core, spaceZero, body)),
		veneraAt(foot), veneraCan)
	start := body * veneraRibsFrom
	pitch := (body - start) / float64(size.Can.Ribs)
	for rib := range size.Can.Ribs {
		low := foot + start + pitch*float64(rib)
		// each rib reaches into the core, so no seam shows between them
		assembly.add(veneraTurned(veneraBlock(core*spaceHalf, out,
			spaceZero, pitch*veneraRibShare)), veneraAt(low), veneraAerial)
	}
	step, lid := size.Can.Cap*veneraCapStep, core*veneraLidShare
	assembly.add(veneraTurned([][coordinatePair]float64{{spaceZero, spaceZero},
		{out, spaceZero}, {out, step}, {lid, step}, {lid, size.Can.Cap},
		{spaceZero, size.Can.Cap}}), veneraAt(foot+body), veneraCan)
}
