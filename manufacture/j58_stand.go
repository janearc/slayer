package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// An engine test stand for the J58, as its own kind: a cradle on
// flexures over a pad, holding a j58 by its four mount pins, a load cell
// between the cradle and a post fixed to the pad, and a bellmouth on the
// engine's face. Every size of it is picked, in j58-stand.yaml; where
// the engine and its pins are is the engine's own mount, and the engine
// is the j58 kind's, placed and not told what holds it.
//
// The stand's own space has its pad's middle on the ground at the
// origin and y up; the engine lies in it as the j58 kind lays it, its
// front toward plus x, its axis at the stand's height.

//go:embed j58-stand.yaml
var j58StandRaw []byte

// j58StandSizes is the stand's sizes, in metres, its counts, and the
// look of each of its parts.
type j58StandSizes struct {
	Pad      struct{ Long, Wide, Thick float64 }
	Axis     float64
	Rail     struct{ Wide, Tall, Out, Margin float64 }
	Flexures struct {
		Tall, Thick float64
		Count       int
	}
	Post, Pin, Reach float64
	Cell             struct{ Radius, Long, Post float64 }
	Bell             struct {
		Lip, Long, Wall, Leg, Splay float64
		Steps                       int
	}
	Looks map[string]j58StandLook
	Read  string
	// dressed is what it wears, made once when the file is read.
	dressed looks
}

// j58StandLook is what one of the stand's parts is made of: a material
// slayer knows, or none for a plain colour, its finish, its colour when
// painted or plain, and what it is said to be.
type j58StandLook struct {
	Material, Finish, Alloy string
	Colour                  []float64
}

// The stand's own looks, after the engine's.
const (
	standPad slot = j58Slots + iota
	standCradle
	standFlexure
	standPin
	standCell
	standPost
	standBell
)

// standNames is what each of the stand's looks is called, which is also
// how its look is found in the sizes file.
var standNames = map[slot]string{
	standPad:     "pad",
	standCradle:  "cradle",
	standFlexure: "flexure",
	standPin:     "mount pin",
	standCell:    "load cell",
	standPost:    "thrust post",
	standBell:    "bellmouth",
}

const (
	standPlain     = "none"    // defined: the material a plain colour is made of
	standPainted   = "painted" // defined: the finish book's name for paint
	standChannels  = 3         // defined: red, green and blue
	standLegAt     = 0.5       // picked: the bellmouth's legs meet it this share of the way from its lip to the face
	standLegSplay  = 0.3       // picked: metres each leg's foot stands further out than its top, so it leans
	standLeastFlex = 2         // defined: two flexures are the fewest that hold a rail level
)

var j58StandRead, j58StandError = readJ58Stand(j58StandRaw, j58Read)

// readJ58Stand reads the stand's sizes and checks them against the
// engine it holds, so that making it afterwards cannot fail.
func readJ58Stand(raw []byte, engine j58Sizes) (j58StandSizes, error) {
	var read j58StandSizes
	if err := yamlfile.Read("j58-stand.yaml", raw, &read); err != nil {
		return j58StandSizes{}, err
	}
	if err := read.sound(); err != nil {
		return j58StandSizes{}, err
	}
	if err := read.fits(engine); err != nil {
		return j58StandSizes{}, err
	}
	dressed, err := read.looks()
	if err != nil {
		return j58StandSizes{}, err
	}
	read.dressed = dressed
	return read, nil
}

// lengths is every length the file gives, by name, for sound to check.
func (stand j58StandSizes) lengths() map[string]float64 {
	return map[string]float64{"pad long": stand.Pad.Long,
		"pad wide": stand.Pad.Wide, "pad thick": stand.Pad.Thick,
		"axis": stand.Axis, "rail wide": stand.Rail.Wide,
		"rail tall": stand.Rail.Tall, "rail out": stand.Rail.Out,
		"rail margin": stand.Rail.Margin, "flexure tall": stand.Flexures.Tall,
		"flexure thick": stand.Flexures.Thick, "post": stand.Post,
		"pin": stand.Pin, "reach": stand.Reach, "cell radius": stand.Cell.Radius,
		"cell long": stand.Cell.Long, "cell post": stand.Cell.Post,
		"bell lip": stand.Bell.Lip, "bell long": stand.Bell.Long,
		"bell wall": stand.Bell.Wall, "bell leg": stand.Bell.Leg,
		"bell splay": stand.Bell.Splay}
}

// sound refuses a size at nought or below, too few flexures or corners,
// and sizes read nowhere.
func (stand j58StandSizes) sound() error {
	for name, each := range stand.lengths() {
		if !sound(each) {
			return fmt.Errorf("manufacture: the j58 stand's %s is %v", name,
				each)
		}
	}
	if stand.Flexures.Count < standLeastFlex || stand.Bell.Steps < j58Least {
		return fmt.Errorf("manufacture: the j58 stand has %d flexures a "+
			"rail and %d steps on its bellmouth", stand.Flexures.Count,
			stand.Bell.Steps)
	}
	if stand.Read == "" {
		return fmt.Errorf("manufacture: the j58 stand's sizes were read " +
			"nowhere")
	}
	return nil
}

// fits refuses a stand that cannot hold its engine: a cradle reaching
// the engine, posts inside its envelope, pins longer than the lugs they
// go into, a bellmouth narrower than the face or touching the pad, and
// a pad too small for what stands on it.
func (stand j58StandSizes) fits(engine j58Sizes) error {
	under := stand.Axis - engine.Envelope*spaceHalf
	if under <= stand.Flexures.Tall+stand.Rail.Tall ||
		stand.Rail.Out <= stand.Post*spaceHalf ||
		stand.Reach >= engine.Mounts.Lug {
		return fmt.Errorf("manufacture: the j58 stand's cradle, its axis "+
			"%v up and its rails %v out, does not clear the engine",
			stand.Axis, stand.Rail.Out)
	}
	if stand.Bell.Lip <= engine.Radii.Face ||
		stand.Bell.Lip+stand.Bell.Wall >= stand.Axis {
		return fmt.Errorf("manufacture: the j58 stand's bellmouth, %v at "+
			"its lip, does not fit between the face and the pad",
			stand.Bell.Lip)
	}
	front := engine.Length*spaceHalf + stand.Bell.Long
	side := stand.railOut(engine) + stand.Post*spaceHalf
	if front >= stand.Pad.Long*spaceHalf || side >= stand.Pad.Wide*spaceHalf ||
		stand.cellFront(engine)+stand.Cell.Post >= stand.Pad.Long*spaceHalf {
		return fmt.Errorf("manufacture: the j58 stand's pad, %v by %v, is "+
			"too small for the engine on it", stand.Pad.Long, stand.Pad.Wide)
	}
	return nil
}

// looks is what the stand wears: each part's material, painted or in its
// finish, or a plain colour where slayer has no such material.
func (stand j58StandSizes) looks() (looks, error) {
	dressed := looks{}
	for where, name := range standNames {
		look, given := stand.Looks[name]
		if !given || look.Alloy == "" {
			return nil, fmt.Errorf("manufacture: the j58 stand's %s has no "+
				"look, or says nothing of what it is", name)
		}
		made, err := standLook(name, look)
		if err != nil {
			return nil, err
		}
		made.Name = name
		dressed[where] = made
	}
	return dressed, nil
}

// standLook is one look: plain, painted, or a finish on a metal.
func standLook(name string, look j58StandLook) (*thing.Material, error) {
	whose := "the j58 stand's " + name
	if look.Material != standPlain && look.Finish != standPainted {
		return surfaceOf(whose, look.Material, look.Finish)
	}
	if len(look.Colour) != standChannels || !share(look.Colour[0]) ||
		!share(look.Colour[1]) || !share(look.Colour[2]) {
		return nil, fmt.Errorf("manufacture: %s's colour %v is not three "+
			"shares of nought to one", whose, look.Colour)
	}
	colour := maths.RGB(look.Colour[0], look.Colour[1], look.Colour[2])
	if look.Material == standPlain {
		return plain(name, colour.R, colour.G, colour.B), nil
	}
	kind, err := materials.ByName(look.Material)
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", whose, err)
	}
	return thing.Made(kind, thing.Painted(colour))
}

// place is the engine's own space in the stand's: laid as the j58 kind
// lays it, its axis the stand's height above the pad.
func (stand j58StandSizes) place(engine j58Sizes) maths.Subspace {
	return engine.lyingAt(stand.Pad.Thick + stand.Axis)
}

// railOut is how far either side of the engine's axis the rails stand.
func (stand j58StandSizes) railOut(engine j58Sizes) float64 {
	return engine.Mounts.Reach + stand.Rail.Out
}

// railFoot is how high the rails' underside is, on their flexures.
func (stand j58StandSizes) railFoot() float64 {
	return stand.Pad.Thick + stand.Flexures.Tall
}

// railTop is how high the rails' top is, where the posts stand.
func (stand j58StandSizes) railTop() float64 {
	return stand.railFoot() + stand.Rail.Tall
}

// pins is where the engine's mount pins stand, in the stand's space.
func (stand j58StandSizes) pins(engine j58Sizes) []maths.Vec {
	place := stand.place(engine)
	var pins []maths.Vec
	for _, pin := range engine.mount().Pins {
		pins = append(pins, place.Out(pin))
	}
	return pins
}

// railEnds is how far forward and how far aft the rails run: past the
// foremost and the aftmost posts by the margin.
func (stand j58StandSizes) railEnds(engine j58Sizes) (float64, float64) {
	front, back := math.Inf(-spaceOne), math.Inf(spaceOne)
	for _, pin := range stand.pins(engine) {
		front, back = math.Max(front, pin.X), math.Min(back, pin.X)
	}
	return front + stand.Rail.Margin, back - stand.Rail.Margin
}

// cellFront is where the thrust post's face stands, ahead of the rails'
// front by the load cell's length.
func (stand j58StandSizes) cellFront(engine j58Sizes) float64 {
	front, _ := stand.railEnds(engine)
	return front + stand.Cell.Long
}

// extent is the stand's box: the pad's plan, and as high as the
// engine's envelope or the bellmouth's lip, whichever is higher. The
// lip is the bellmouth's widest: its wall stands aft of it there.
func (stand j58StandSizes) extent(engine j58Sizes) maths.Box {
	top := stand.Pad.Thick + stand.Axis + math.Max(engine.Envelope*
		spaceHalf, stand.Bell.Lip)
	long, wide := stand.Pad.Long*spaceHalf, stand.Pad.Wide*spaceHalf
	return maths.Box{Min: maths.V(-long, spaceZero, -wide),
		Max: maths.V(long, top, wide)}
}

// build puts every part of the stand into an assembly.
func (stand j58StandSizes) build(assembly *instrumentAssembly,
	engine j58Sizes,
) {
	assembly.add(boxShape(stand.Pad.Long, stand.Pad.Thick, stand.Pad.Wide),
		maths.Local(), standPad)
	stand.cradle(assembly, engine)
	stand.posts(assembly, engine)
	stand.thrust(assembly, engine)
	stand.bellmouth(assembly, engine)
}

// standAt is a part's space standing at a point, unturned.
func standAt(at maths.Vec) maths.Subspace {
	return maths.NewSubspace(at, maths.Identity, spaceOne)
}

// cradle lays the two rails on their flexures, and the cross beams
// between them at each end and under each pair of pins.
func (stand j58StandSizes) cradle(assembly *instrumentAssembly,
	engine j58Sizes,
) {
	rail, out := stand.Rail, stand.railOut(engine)
	front, back := stand.railEnds(engine)
	for _, side := range []float64{-out, out} {
		assembly.add(boxShape(front-back, rail.Tall, rail.Wide), standAt(
			maths.V((front+back)*spaceHalf, stand.railFoot(), side)),
			standCradle)
		for flexure := range stand.Flexures.Count {
			share := float64(flexure) / float64(stand.Flexures.Count-spaceOne)
			along := back + rail.Wide + (front-back-rail.Wide-rail.Wide)*share
			assembly.add(boxShape(stand.Flexures.Thick, stand.Flexures.Tall,
				rail.Wide), standAt(maths.V(along, stand.Pad.Thick, side)),
				standFlexure)
		}
	}
	beams := []float64{front - rail.Wide*spaceHalf, back + rail.Wide*spaceHalf}
	for _, pin := range stand.pins(engine) {
		if pin.Z > spaceZero {
			beams = append(beams, pin.X)
		}
	}
	for _, along := range beams {
		assembly.add(boxShape(rail.Wide, rail.Tall, out+out-rail.Wide),
			standAt(maths.V(along, stand.railFoot(), spaceZero)), standCradle)
	}
}

// posts stands a post on a rail beside each of the engine's pins, up to
// just over it, and a pin from the post into the engine's lug.
func (stand j58StandSizes) posts(assembly *instrumentAssembly,
	engine j58Sizes,
) {
	out := stand.railOut(engine)
	for _, pin := range stand.pins(engine) {
		side := math.Copysign(spaceOne, pin.Z)
		assembly.add(boxShape(stand.Post, pin.Y+stand.Post*spaceHalf-
			stand.railTop(), stand.Post), standAt(maths.V(pin.X,
			stand.railTop(), side*out)), standCradle)
		assembly.rod(maths.V(pin.X, pin.Y, pin.Z-side*stand.Reach),
			maths.V(pin.X, pin.Y, side*out), stand.Pin, standPin)
	}
}

// thrust sets the load cell along the axis from the cradle's front beam
// to the thrust post, which stands on the pad and not on the cradle.
func (stand j58StandSizes) thrust(assembly *instrumentAssembly,
	engine j58Sizes,
) {
	front, _ := stand.railEnds(engine)
	middle := stand.railFoot() + stand.Rail.Tall*spaceHalf
	assembly.rod(maths.V(front, middle, spaceZero), maths.V(
		stand.cellFront(engine), middle, spaceZero), stand.Cell.Radius,
		standCell)
	assembly.add(boxShape(stand.Cell.Post, stand.railTop()-stand.Pad.Thick,
		stand.Cell.Post), standAt(maths.V(stand.cellFront(engine)+
		stand.Cell.Post*spaceHalf, stand.Pad.Thick, spaceZero)), standPost)
}

// bellAt is a point of the bellmouth's inside a share of the way from
// its lip to the face, as a radius and a distance aft of the face, and
// the way out of its wall there.
func (stand j58StandSizes) bellAt(engine j58Sizes,
	share float64,
) ([coordinatePair]float64, [coordinatePair]float64) {
	bell, face := stand.Bell, engine.Radii.Face
	turn := math.Pi * spaceHalf * (spaceOne - share)
	flare := bell.Lip - face
	normal := math.Hypot(bell.Long*math.Cos(turn), flare*math.Sin(turn))
	return [coordinatePair]float64{face + flare*(spaceOne-math.Cos(turn)),
		-bell.Long * math.Sin(turn)}, [coordinatePair]float64{
		bell.Long * math.Cos(turn) / normal, flare * math.Sin(turn) / normal}
}

// bellOutside is a point of the bellmouth's outside, a wall out from its
// inside along the way out.
func (stand j58StandSizes) bellOutside(engine j58Sizes,
	share float64,
) [coordinatePair]float64 {
	inside, out := stand.bellAt(engine, share)
	return [coordinatePair]float64{inside[spaceZero] +
		out[spaceZero]*stand.Bell.Wall, inside[spaceOne] +
		out[spaceOne]*stand.Bell.Wall}
}

// bellProfile is the bellmouth cut across: its outside from the lip to
// the face, then its inside back to the lip, counterclockwise as seen
// from outside. Its inside meets the face at the face's radius.
func (stand j58StandSizes) bellProfile(engine j58Sizes) [][coordinatePair]float64 {
	steps := float64(stand.Bell.Steps)
	lip, _ := stand.bellAt(engine, spaceZero)
	loop := [][coordinatePair]float64{lip}
	for step := range stand.Bell.Steps + spaceOne {
		loop = append(loop, stand.bellOutside(engine, float64(step)/steps))
	}
	for step := stand.Bell.Steps; step > spaceZero; step-- {
		inside, _ := stand.bellAt(engine, float64(step)/steps)
		loop = append(loop, inside)
	}
	return loop
}

// bellmouth sets the bellmouth on the engine's face and its two legs,
// each from under its curve, either side, down and out to the pad.
func (stand j58StandSizes) bellmouth(assembly *instrumentAssembly,
	engine j58Sizes,
) {
	place := stand.place(engine)
	build := j58Build{assembly: assembly, place: place}
	build.turned(stand.bellProfile(engine), standBell)
	outside := stand.bellOutside(engine, standLegAt)
	for _, side := range []float64{-spaceOne, spaceOne} {
		turn := math.Pi + side*stand.Bell.Splay*maths.Degree
		top := place.Out(j58Radial(turn).Scale(outside[spaceZero]).Add(
			maths.Up.Scale(outside[spaceOne])))
		foot := maths.V(top.X, stand.Pad.Thick, top.Z+math.Copysign(
			standLegSplay, top.Z))
		assembly.rod(top, foot, stand.Bell.Leg, standBell)
	}
}

// j58Stand is the maker of the test stand.
type j58Stand struct{}

// made keeps quick as the stand's default detail.
func (row j58Stand) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt builds the stand at a detail, its repeated flat parts shared.
func (row j58Stand) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	stand, engine := j58StandRead, j58Read
	assembly := stand.stood(engine, detail)
	if err := assembly.shareRepeats(); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	object, err := assembly.finish(stand.extent(engine))
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// stood is the stand's assembly at a detail with the engine on it, held
// by its mount: the engine is built at the place the stand gives its own
// space, and wears its own looks beside the stand's. Every part is
// placed and none yet shared.
func (stand j58StandSizes) stood(engine j58Sizes,
	detail string,
) *instrumentAssembly {
	dressed := engine.dressedCopy()
	for where, look := range stand.dressed {
		own := *look
		dressed[where] = &own
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: dressed}}
	stand.build(assembly, engine)
	engine.build(assembly, stand.place(engine))
	return assembly
}
