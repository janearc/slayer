package lantern

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A played lantern is a beamed one whose parts answer to channels: the
// star's brightness, colour and breath; the beams' brightness, width,
// cone, spin and precession; the glare; and each shell's tumble, breath,
// band and opening. The channels are data, in zappy-channels.yaml.
//
// It is placed once; Hold sets every channel for the moment about to be
// drawn and changes only the parts whose channels changed, so a frame
// costs what it changed.
//
// With every channel at rest the world holds exactly what Place put in
// it, so the picture is the beamed lantern's own.

//go:embed zappy-channels.yaml
var channelsRaw []byte

// refusal is a refusal of this package, held as a constant so that no one
// can assign it.
type refusal string

// Error is the refusal's words.
func (r refusal) Error() string { return string(r) }

// ErrChannel is a channel a played lantern has not, or a value outside
// one's range.
const ErrChannel refusal = "lantern: no such channel value"

// Channel is one thing music can move in a played lantern: its name, what
// it does, its unit, the least and most it may be, and its rest, the
// value that draws the lantern as it is defined.
type Channel struct {
	Name, Says, Unit  string
	Least, Most, Rest float64
}

// written is a channel as zappy-channels.yaml writes it.
type written struct {
	Name  string  `yaml:"name"`
	Says  string  `yaml:"says"`
	Unit  string  `yaml:"unit"`
	Least float64 `yaml:"least"`
	Most  float64 `yaml:"most"`
	Rest  string  `yaml:"rest"`
	Shell bool    `yaml:"shell"`
}

// the channels a played lantern has, by name
const (
	starBrightness = "star-brightness"
	starKelvin     = "star-kelvin"
	starBreath     = "star-breath"
	beamBrightness = "beam-brightness"
	beamWidth      = "beam-width"
	beamCone       = "beam-cone"
	beamSpin       = "beam-spin"
	beamPrecess    = "beam-precess"
	glare          = "glare"
	tumble         = "tumble"
	breath         = "breath"
	band           = "band"
	open           = "open"
	allShells      = "shells-"
	oneShell       = "shell-%d-"
)

// Played is a beamed lantern placed in a world with its channels.
type Played struct {
	made     Lantern
	clock    *time.Duration
	world    *thing.World
	channels []Channel
	rest     map[string]float64
	held     map[string]float64
	star     int
	starRest *thing.Thing
	lamps    int
	lampRest []thing.Lamp
	shells   []playedShell
	// remeshed are the sheets a shell's band and opening have made, kept
	// so a value held is made once
	remeshed map[remesh][]shape.Shape
}

// remesh is one shell's band and opening, the key its sheets are kept by.
type remesh struct {
	number        int
	share, opened float64
}

// playedShell is one shell as placed: its orbit and number, its things'
// places in the world and what Place made of them, and its sizes.
type playedShell struct {
	number, count     int
	ring              orbit
	turning           spin
	things            []int
	rest              []*thing.Thing
	out, side, radius float64
}

// Play places the beamed lantern in the world as Place does, its beams
// read at whatever time clock holds, and gives it its channels, at rest.
func (beamed Beamed) Play(world *thing.World,
	clock *time.Duration) (*Played, error) {
	made, err := beamed.Lantern(clock)
	if err != nil {
		return nil, err
	}
	things, lamps := len(world.Things), len(world.Lamps)
	if _, err := made.Place(world); err != nil {
		return nil, err
	}
	played := &Played{made: made, clock: clock, world: world,
		star: things, starRest: world.Things[things], lamps: lamps,
		lampRest: append([]thing.Lamp(nil), world.Lamps[lamps:]...)}
	played.channels, played.rest = channelsOf(beamed)
	played.held = played.restingValues()
	played.shells = played.shellsOf(things + 1)
	return played, nil
}

// shellsOf is each shell as Place put it, its things from first on.
func (played *Played) shellsOf(first int) []playedShell {
	made := played.made
	sheets := 1
	if made.Lining != nil {
		sheets = 2
	}
	firstOut, firstBand, _ := made.shelled(0)
	shells := make([]playedShell, made.Orbits)
	for number := range made.Orbits {
		out, side, _ := made.shelled(number)
		one := playedShell{number: number, ring: orbitOf(number,
			made.Orbits), out: out, side: side, radius: made.Radius,
			count: int(math.Round(float64(made.Panels) * out / firstOut *
				firstBand / side))}
		for sheet := range sheets {
			at := first + number*sheets + sheet
			one.things = append(one.things, at)
			one.rest = append(one.rest, played.world.Things[at])
		}
		one.turning = played.world.Things[one.things[0]].Motion.(spin)
		shells[number] = one
	}
	return shells
}

// channelsOf is a beamed lantern's channels, from zappy-channels.yaml,
// each shell's once for all and once for each, and their rests.
func channelsOf(beamed Beamed) ([]Channel, map[string]float64) {
	var file struct {
		Channels []written `yaml:"channels"`
	}
	if err := yaml.Unmarshal(channelsRaw, &file); err != nil {
		panic(err)
	}
	own := map[string]float64{"kelvin": beamed.Kelvin,
		"beam-width": beamed.Beams.Width, "beam-cone": beamed.Beams.Cone,
		"glare": beamed.Glare}
	var channels []Channel
	rest := map[string]float64{}
	add := func(one written, name string) {
		value, err := restOf(one.Rest, own)
		if err != nil {
			panic(err)
		}
		channels = append(channels, Channel{Name: name, Says: one.Says,
			Unit: one.Unit, Least: one.Least, Most: one.Most, Rest: value})
		rest[name] = value
	}
	for _, one := range file.Channels {
		if !one.Shell {
			add(one, one.Name)
			continue
		}
		add(one, allShells+one.Name)
		for number := range beamed.Orbits {
			add(one, fmt.Sprintf(oneShell, number+1)+one.Name)
		}
	}
	return channels, rest
}

// restOf is a written rest: a number, or the name of one of the lantern's
// own values.
func restOf(said string, own map[string]float64) (float64, error) {
	if value, found := own[said]; found {
		return value, nil
	}
	var value float64
	if _, err := fmt.Sscanf(said, "%g", &value); err != nil {
		return 0, fmt.Errorf("lantern: a channel rests at %q, which is "+
			"neither a number nor one of zappy's values", said)
	}
	return value, nil
}

// Channels are the played lantern's channels, in the order declared.
func (played *Played) Channels() []Channel {
	return append([]Channel(nil), played.channels...)
}

// restingValues is every channel at its rest.
func (played *Played) restingValues() map[string]float64 {
	held := make(map[string]float64, len(played.rest))
	for name, value := range played.rest {
		held[name] = value
	}
	return held
}

// Hold sets every channel for the moment about to be drawn: those given
// to their values, the rest to their rests. A channel the lantern has not
// and a value out of a channel's range are refused by name, and nothing
// is changed.
func (played *Played) Hold(values map[string]float64) error {
	held := played.restingValues()
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := played.check(name, values[name]); err != nil {
			return err
		}
		held[name] = values[name]
	}
	played.held = held
	played.holdStar()
	played.holdLamps()
	return played.holdShells()
}

// check is why a value cannot be held on a channel, or nothing.
func (played *Played) check(name string, value float64) error {
	for _, one := range played.channels {
		if one.Name != name {
			continue
		}
		if !(value >= one.Least && value <= one.Most) {
			return fmt.Errorf("%w: %s is %v, outside %v to %v",
				ErrChannel, name, value, one.Least, one.Most)
		}
		return nil
	}
	return fmt.Errorf("%w: %q is not one of zappy's channels; they are "+
		"%s", ErrChannel, name, strings.Join(played.names(), ", "))
}

// names are the channels' names, in the order declared.
func (played *Played) names() []string {
	names := make([]string, len(played.channels))
	for i, one := range played.channels {
		names[i] = one.Name
	}
	return names
}

// resting is whether each named channel is at its rest.
func (played *Played) resting(names ...string) bool {
	for _, name := range names {
		if played.held[name] != played.rest[name] {
			return false
		}
	}
	return true
}

// lantern is the lantern as the channels now have it: its star's radius,
// colour and glare.
func (played *Played) lantern() Lantern {
	made := played.made
	made.Radius *= played.held[starBreath]
	made.Kelvin = played.held[starKelvin]
	made.Glare = played.held[glare]
	return made
}

// holdStar puts the star as the channels have it: as Place made it at
// rest, else grown by its breath and glowing by its brightness.
func (played *Played) holdStar() {
	if played.resting(starBrightness, starBreath) {
		played.world.Things[played.star] = played.starRest
		return
	}
	star := *played.starRest
	glowing := *star.Material
	glowing.Glow = glowing.Glow.Scale(played.held[starBrightness])
	star.Material = &glowing
	star.Shape = shape.Sphere{Radius: played.lantern().Radius}
	played.world.Things[played.star] = &star
}

// holdLamps puts the star's light and the beams as the channels have
// them: as Place made them at rest, else made again.
func (played *Played) holdLamps() {
	lamps := played.lamps
	if played.resting(starBrightness, starKelvin, starBreath) {
		played.world.Lamps[lamps] = played.lampRest[0]
	} else {
		made := played.lantern()
		light := made.light()
		light.Light = light.Light.Scale(played.held[starBrightness])
		played.world.Lamps[lamps] = light.When(0)
	}
	if played.resting(starKelvin, starBreath, glare, beamBrightness,
		beamWidth, beamCone, beamSpin, beamPrecess) {
		played.world.Lamps[lamps+1] = played.lampRest[1]
		return
	}
	played.world.Lamps[lamps+1] = played.beams()
}

// beams are the beams made again as the channels have them.
func (played *Played) beams() thing.Lamp {
	made := played.lantern()
	polar := *made.Beams
	polar.Width = radians(played.held[beamWidth])
	polar.Cone = radians(played.held[beamCone])
	polar.SpinPhase = played.held[beamSpin]
	polar.PrecessPhase = played.held[beamPrecess]
	made.Beams = &polar
	lit := made.beams()
	lit.Light = lit.Light.Scale(played.held[beamBrightness])
	return lit.Beams(played.clock)
}

// shellValue is a shell's value of a channel: its own when off its rest,
// else all the shells'.
func (played *Played) shellValue(number int, name string) float64 {
	own := fmt.Sprintf(oneShell, number+1) + name
	if !played.resting(own) {
		return played.held[own]
	}
	return played.held[allShells+name]
}

// holdShells puts each shell as the channels have it: as Place made it
// at rest; turned on and grown when its tumble or breath is off rest;
// made again when its band or opening is.
func (played *Played) holdShells() error {
	for _, one := range played.shells {
		phase := played.shellValue(one.number, tumble)
		scale := played.shellValue(one.number, breath)
		share := played.shellValue(one.number, band)
		opened := played.shellValue(one.number, open)
		things := make([]*thing.Thing, len(one.rest))
		copy(things, one.rest)
		if share != played.rest[allShells+band] ||
			opened != played.rest[allShells+open] {
			if err := played.remade(one, things, share,
				opened); err != nil {
				return err
			}
		}
		if phase != played.rest[allShells+tumble] ||
			scale != played.rest[allShells+breath] {
			moved := playedSpin{spin: one.turning, phase: phase,
				scale: scale}
			for i := range things {
				turned := *things[i]
				turned.Motion = moved
				things[i] = &turned
			}
		}
		for i, at := range one.things {
			played.world.Things[at] = things[i]
		}
	}
	return nil
}

// remade is a shell's sheets made again at a share of its band and with
// its panels opened as asked, into things. Making them costs about half a
// second a shell, so what is made is kept: a value held costs it once.
func (played *Played) remade(one playedShell, things []*thing.Thing,
	share, opened float64) error {
	key := remesh{number: one.number, share: share, opened: opened}
	shapes, found := played.remeshed[key]
	if !found {
		var err error
		if shapes, err = played.sheetsOf(one, len(things), share,
			opened); err != nil {
			return err
		}
		if played.remeshed == nil {
			played.remeshed = map[remesh][]shape.Shape{}
		}
		played.remeshed[key] = shapes
	}
	for i := range things {
		sheet := *things[i]
		sheet.Shape = shapes[i]
		things[i] = &sheet
	}
	return nil
}

// sheetsOf is a shell's sheets made at a share of its band and with its
// panels opened as asked.
func (played *Played) sheetsOf(one playedShell, count int, share,
	opened float64) ([]shape.Shape, error) {
	side := one.side * share * one.radius
	covered := func(radius float64) float64 {
		return (1 - opened) * 2 * math.Pi * radius / float64(one.count)
	}
	outs := []float64{one.out, one.out - liningGap}[:count]
	shapes := make([]shape.Shape, len(outs))
	for i, out := range outs {
		shell := out * one.radius
		panels, err := ringMeshOf(one.number, one.count,
			played.made.Wear, func(index int) ([4]maths.Vec, maths.Vec) {
				return one.ring.panelCovering(one.number, index,
					one.count, shell, side, covered)
			})
		if err != nil {
			return nil, err
		}
		shapes[i] = panels
	}
	return shapes, nil
}

// liningGap is how far inside its skin a lining sheet sits, in radii of
// the star, as placeRing sets it.
const liningGap = 0.02

// playedSpin is a shell's turning with a phase added to its tumble and
// grown about the star by a scale.
type playedSpin struct {
	spin
	phase, scale float64
}

// At is where the shell is at a time: turned as its spin turns it, its
// tumble moved on by the phase, and grown by the scale.
func (turning playedSpin) At(at time.Duration) maths.Subspace {
	base := turning.spin
	round := maths.About(base.pole, angleAt(at, base.turn))
	over := maths.About(base.across, angleAt(at, base.tumble)+
		2*math.Pi*turning.phase)
	return maths.Subspace{Origin: base.at, Orientation: over.Then(round),
		Scale: turning.scale}
}
