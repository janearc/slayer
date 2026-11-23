package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
)

// signCulture is the culture the words, colours and sizes of signs are
// drawn from.
const signCulture = "sign"

// clumpsLevel is the detail of a building seen from far, a little nearer
// than its hull, which, like the hull, draws a sign as a glowing panel
// and not as letters.
const clumpsLevel Detail = "clumps"

// SignLevel is the detail a sign is drawn at, given where it hangs, in the
// frame of the building that offers the place, and how big it is across.
// It is how a sign is made to follow the camera: far signs stay a glowing
// panel and the letters unfurl as the camera comes near.
type SignLevel func(at maths.Vec, size float64) Detail

// ConstantLevel is a SignLevel that gives one detail wherever a sign hangs.
func ConstantLevel(level Detail) SignLevel {
	return func(maths.Vec, float64) Detail { return level }
}

// signer draws the signs of one building: the culture's draws for a seed,
// the words already said, so that no two signs of a building say the same,
// the make of each sign it has drawn, and the fonts it has loaded. Its
// lettering is the script the words are in, the suffix that script adds to
// the culture's names, the places given to the trades the building must
// carry, by index, and which letters each face has refused.
type signer struct {
	drawing   *drawer
	levelOf   SignLevel
	used      []string
	made      []string
	faces     map[string]*typography.Font
	lettering Lettering
	suffix    string
	owing     map[int]string
	refused   map[string]map[rune]bool
}

// newSigner is a signer of a seed, which draws each sign's detail by a
// level, in Latin capitals.
func newSigner(seed Seed, levelOf SignLevel) (*signer, error) {
	return newLetteredSigner(seed, levelOf, Lettering{})
}

// newLetteredSigner is a signer of a seed, which draws each sign's detail
// by a level, in the script of a lettering. It refuses a lettering the
// culture cannot keep.
func newLetteredSigner(seed Seed, levelOf SignLevel,
	lettering Lettering) (*signer, error) {
	if levelOf == nil {
		return nil, fmt.Errorf("cities: signs are drawn at a detail, and " +
			"none was given")
	}
	drawing, err := newDrawer(signCulture, seed)
	if err != nil {
		return nil, err
	}
	if err := lettering.refusal(drawing); err != nil {
		return nil, err
	}
	return &signer{drawing: drawing, levelOf: levelOf,
		faces: map[string]*typography.Font{}, lettering: lettering,
		suffix: lettering.suffix(), refused: map[string]map[rune]bool{}}, nil
}

// signs are the signs a building carries, as parts in the building's
// frame: on each place it offers, a sign of that kind by the chance the
// culture gives, drawn by the seed. The words of no two signs of a
// building are the same. They are Latin capitals.
func signs(from Mounting, seed Seed, levelOf SignLevel) ([]placedElement, error) {
	return SignsIn(from, seed, levelOf, Lettering{})
}

// SignsIn is Signs in the script of a lettering, and with the trades the
// lettering says the building must carry, each on a place that is made a
// sign whatever chance the culture gives it. A lettering that says nothing
// gives the signs Signs gives, part for part.
func SignsIn(from Mounting, seed Seed, levelOf SignLevel,
	lettering Lettering) ([]placedElement, error) {
	parts, _, err := signsSaying(from, seed, levelOf, lettering)
	return parts, err
}

// signsSaying is SignsIn and the words the signs said, in the order they
// were hung.
func signsSaying(from Mounting, seed Seed, levelOf SignLevel,
	lettering Lettering) ([]placedElement, []string, error) {
	mounts, err := from.SignMounts()
	if err != nil {
		return nil, nil, err
	}
	signing, err := newLetteredSigner(seed, levelOf, lettering)
	if err != nil {
		return nil, nil, err
	}
	signing.owing = owing(mounts, lettering.Carries)
	var parts []placedElement
	for index, mount := range mounts {
		share := "sign-" + string(mount.Kind) + "-share"
		_, owes := signing.owing[index]
		if owes || signing.drawing.chance(share, place{Opening: index}) {
			parts = append(parts, signing.hang(mount, index)...)
		}
	}
	return parts, signing.used, signing.drawing.trouble
}

// FreeSign is a sign of a kind on its own, standing at the origin of its
// design with its foot on the ground, in the room the culture gives a sign
// that hangs on nothing. A blade sign hangs from a bracket at the origin,
// and an awning from the wall at the origin, over the ground. The design's
// name says the kind and how the sign is made. It refuses a kind that is
// not a way a sign hangs. Its words are Latin capitals.
func FreeSign(kind MountKind, seed Seed, level Detail) (Design, error) {
	return FreeSignIn(kind, seed, level, Lettering{})
}

// FreeSignIn is FreeSign in the script of a lettering. A lettering that
// carries a trade makes the sign that trade's; CheckFree says what it may
// carry.
func FreeSignIn(kind MountKind, seed Seed, level Detail,
	lettering Lettering) (Design, error) {
	signing, err := newLetteredSigner(seed, ConstantLevel(level), lettering)
	if err != nil {
		return Design{}, err
	}
	drawing := signing.drawing
	if err := lettering.CheckFree(kind); err != nil {
		return Design{}, err
	}
	mount := signMount{Kind: kind, Out: maths.ZAxis, Along: maths.XAxis,
		Wide: drawing.fixed("sign-free-wide"),
		High: drawing.fixed("sign-free-high")}
	switch kind {
	case MountRoof:
		mount.Reach = drawing.fixed("sign-roof-set-back")
	case MountScreen:
		mount.Reach = drawing.fixed("sign-screen-reach")
	case MountBlade:
		mount.Reach = drawing.fixed("sign-blade-reach")
		mount.Point = maths.V(0, mount.High, 0)
	case MountAwning:
		mount.Reach = drawing.fixed("sign-awning-reach-most")
		mount.High = drawing.fixed("sign-awning-high")
	default:
		return Design{}, fmt.Errorf("cities: no sign hangs as a %q, only "+
			"as %q, %q, %q or %q", kind, MountRoof, MountScreen, MountBlade,
			MountAwning)
	}
	signing.owing = owing([]signMount{mount}, lettering.Carries)
	parts := signing.hang(mount, 0)
	if drawing.trouble != nil {
		return Design{}, drawing.trouble
	}
	return Design{Name: "sign " + string(kind) + " " + signing.made[0],
		Elements: parts}, nil
}

// face is the font of a name, loaded once.
func (signing *signer) face(name string) *typography.Font {
	if face, loaded := signing.faces[name]; loaded {
		return face
	}
	face, err := signing.load(name)
	signing.drawing.fail(err)
	signing.faces[name] = face
	return face
}

// hang is the sign for one place, the index of the place in the building's
// offer being what the draws are for, as parts in the building's frame.
func (signing *signer) hang(mount signMount, index int) []placedElement {
	job := signing.job(mount, index)
	if signing.drawing.trouble != nil {
		return nil
	}
	signing.made = append(signing.made, job.look)
	var parts []placedElement
	switch mount.Kind {
	case MountRoof:
		parts = job.roof(mount)
	case MountScreen:
		parts = job.screen(mount)
	case MountAwning:
		parts = job.awning(mount)
	default:
		parts = job.blade(mount)
	}
	return within(parts, mount.space())
}

// signJob is everything needed to draw one sign: the draws it has made
// for its make, trade, colour, font and metal, the wear it shows, and the
// detail it is drawn at.
//
// The paint is the board of a painted sign, and the
// lettering what its letters are flat in, when they are. The face is the
// font its look is set in, faceName its name, and spares the names of the
// other fonts the look may be set in, for a word the face cannot draw.
//
// The said are the words the sign has said so far, shared by the jobs made of
// this one. The cap is how tall a letter is, and the rise how far it is set
// above its baseline, both shares of the em.
type signJob struct {
	signing   *signer
	drawing   *drawer
	at        place
	level     Detail
	used      *[]string
	look      string
	trade     option
	neon      option
	face      *typography.Font
	faceName  string
	spares    []string
	said      *[]string
	steel     *thing.Material
	frame     *thing.Material
	paint     *thing.Material
	lettering *thing.Material
	spot      wornSpot
	cap       float64
	rise      float64
}

// job is the job of a sign at a place of the building's offer, with its
// make, trade, colour, font and metal drawn for it, and its wear.
func (signing *signer) job(mount signMount, index int) signJob {
	drawing := signing.drawing
	at := place{Opening: index}
	kind := "sign-make-" + string(mount.Kind)
	look := drawing.pick(kind, signing.inScript(kind), at).Name
	return signJob{signing: signing, drawing: drawing, at: at,
		level: signing.levelOf(mount.Point, math.Max(mount.Wide, mount.High)),
		used:  &signing.used, look: look, said: new([]string),
		trade: signing.trade(at, index),
		neon:  drawing.pick("sign-neon", drawing.options("sign-neon"), at),
		steel: paint("sign body", drawing.pick("sign-body-colour",
			drawing.options("sign-body-colour"), at)),
		frame: paint("sign frame", drawing.pick("sign-frame-colour",
			drawing.options("sign-frame-colour"), at)),
		spot: drawing.wear(look, at),
		cap:  drawing.fixed("sign-cap-share" + signing.suffix),
		rise: signing.rise()}.faced(look).painted()
}

// painted is the job with the paint and ink its make needs: a painted
// board and letters in a colour against it, from the dark and light
// paints, or for an awning's valance letters in a light paint, or for a lit
// box letters in the dark metal.
func (job signJob) painted() signJob {
	drawing := job.drawing
	dark, light := drawing.options("sign-paint-dark"),
		drawing.options("sign-paint-light")
	switch job.look {
	case lookLamped:
		board, ink := dark, light
		if drawing.chance("sign-paint-light-board", job.at) {
			board, ink = light, dark
		}
		job.paint = paint("sign board paint", drawing.pick("sign-paint-dark",
			board, job.at))
		job.lettering = paint("sign letter paint", drawing.pick(
			"sign-paint-light", ink, job.at))
	case lookValance:
		job.lettering = paint("sign letter paint", drawing.pick(
			"sign-paint-light", light, job.at))
	default:
		job.lettering = job.steel
	}
	return job
}

// tubed is the job with its letters pillowed tube neon, in the culture's
// tube font: a screen's letters are always so, whatever the make of the
// sign.
func (job signJob) tubed() signJob {
	job.look = lookTube
	return job.faced(lookTube)
}

// lettered is whether the sign is drawn with letters. Far off, at the hull
// and the clumps, it is one glowing panel.
func (job signJob) lettered() bool {
	return job.level != hullLevel && job.level != clumpsLevel
}

// word is words of a kind that no sign of the building says yet, drawn
// from the culture's list for a place, which are then said.
func (job signJob) word(kind string, at place) string {
	name := "sign-" + kind + job.signing.suffix
	one := job.drawing.pick(name, job.drawing.options(name), at, *job.used...)
	*job.used = append(*job.used, one.Name)
	*job.said = append(*job.said, one.Name)
	return one.Name
}

// tradeWord is a word of a kind, a long one or a short one, from the
// list of the sign's trade, that no sign of the building says yet.
func (job signJob) tradeWord(kind string) string {
	name := "sign-" + kind + "-" + job.trade.Name + job.signing.suffix
	one := job.drawing.pick(name, job.drawing.options(name), job.at,
		*job.used...)
	*job.used = append(*job.used, one.Name)
	*job.said = append(*job.said, one.Name)
	return one.Name
}

// tradeHasAnother is whether the sign's trade has a word of a kind other
// than one given, which a second line of the same trade could say.
func (job signJob) tradeHasAnother(kind, word string) bool {
	name := "sign-" + kind + "-" + job.trade.Name + job.signing.suffix
	for _, one := range job.drawing.options(name) {
		if one.Name != word {
			return true
		}
	}
	return false
}

// shining is a material that glows a colour, a number of times brighter
// than the colour, named for what it is and the colour's name. It is dark
// in itself: a lit tube's glass gives back little of what falls on it.
func (job signJob) shining(of, colour string, light maths.Light,
	scale string) *thing.Material {
	return &thing.Material{Name: "cities sign " + of + " " + colour,
		Colour: maths.Grey(job.drawing.fixed("sign-unlit")),
		Glow:   light.Scale(job.drawing.fixed(scale))}
}

// glow is the material of lit letters of a colour: the glow of the sign's
// make.
func (job signJob) glow(neon option) *thing.Material {
	return job.shining(job.look, neon.Name, colourOf(neon),
		"sign-"+job.look+"-glow")
}

// board is the material of a sign's board: dark metal for a sign with
// letters, paint for a painted one, and for one without letters, from far
// off, a panel that glows the colour of the sign.
func (job signJob) board() *thing.Material {
	switch {
	case !job.lettered():
		return job.shining("panel", job.neon.Name, colourOf(job.neon),
			"sign-panel-glow")
	case job.look == lookLamped:
		return job.paint
	}
	return job.steel
}

// lamps are the lights a sign throws on what is near it, in the colour of
// the sign: as many as its width has room for, capped, in a row across it
// at a height and a distance out of its frame, each a small lit box of the
// glass given. The luminance is the sign's, shared among its lamps, and
// among the sides it has, which this row is one of.
func (job signJob) lamps(across float64, at maths.Vec, luminance float64,
	sides int, glass *thing.Material, colour maths.Light) []placedElement {
	drawing := job.drawing
	count := int(math.Ceil(across / drawing.fixed("sign-lamp-spacing")))
	count = max(1, min(count, int(drawing.fixed("sign-lamp-most"))))
	size := drawing.fixed("sign-lamp-size")
	lamp := &Lamp{Luminance: luminance / float64(count*sides), Tint: colour}
	var parts []placedElement
	for index := range count {
		x := at.X + across*(float64(index)+1.0/half)/float64(count) -
			across/half
		parts = append(parts, placedElement{Element: solid{Name: "sign lamp",
			Form: Box{size, size, size}, Material: glass, Lamp: lamp},
			Place: standingAt(maths.V(x-size/half, at.Y-size/half,
				at.Z-size/half))})
	}
	return parts
}
