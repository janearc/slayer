package cities

import (
	"fmt"
	"slices"
	"strings"

	"github.com/janearc/slayer/typography"
)

// Script is the writing the words of a building's signs are in.
type Script string

// The scripts the sign culture has words for.
const (
	// scriptLatin is the writing a sign has unless another is asked for:
	// capitals, set in go's own fonts.
	scriptLatin Script = "latin"
	// scriptSimplifiedChinese is Chinese in the simplified characters of
	// the mainland, set in files of fonts a shelf keeps, which Lettering's
	// Faces says where to find.
	scriptSimplifiedChinese Script = "simplified-chinese"
)

// scriptSuffix is what a script adds to the name of each element of the
// sign culture that it says over again: the Latin script says them as
// they are named, and every other script says them under its own name,
// the trades, the words, the fonts and the sizes that suit its letters.
var scriptSuffix = map[Script]string{
	scriptLatin:             "",
	scriptSimplifiedChinese: "-hans",
}

// ScriptNames are the scripts a set may ask for, the Latin first.
func ScriptNames() []string {
	return []string{string(scriptLatin), string(scriptSimplifiedChinese)}
}

// signLooks are the makes of a sign, each of which the culture sets in
// fonts of its own.
var signLooks = []string{lookTube, lookChannel, lookLamped, lookBox,
	lookBulb, lookLED, lookValance, lookLitValance}

// tradeMounts are the ways a sign hangs that read the words of a trade,
// in the order a trade a building must carry is given its place: a blade
// sign and an awning hang low, where a street sees them, and a roof sign
// is the last place for it.
var tradeMounts = []MountKind{MountBlade, MountAwning, MountRoof}

// Lettering is what a set says of the writing of the signs of a building.
// The zero value is the signs a building has always had: Latin capitals,
// and the trades the culture draws by weight.
type Lettering struct {
	// Script is the writing the words are in. Nothing is the Latin script.
	Script Script
	// Carries are trades the building must carry a sign of, once each, on
	// the first places it offers that read the words of a trade, whatever
	// the weights of the culture say: each is a trade the culture has
	// words for in the script. A building that offers fewer such places
	// than it is asked to carry trades carries the first of them.
	Carries []string
	// Faces are the files of the fonts the script's words are set in, by
	// the name the culture gives each, as the shelf keeps them. A script
	// that is not Latin sets its words in no font but these.
	Faces map[string]string
}

// script is the script of the lettering, the Latin if it says none.
func (lettering Lettering) script() Script {
	if lettering.Script == "" {
		return scriptLatin
	}
	return lettering.Script
}

// suffix is what the lettering's script adds to the names of elements.
func (lettering Lettering) suffix() string {
	return scriptSuffix[lettering.script()]
}

// Check refuses a lettering whose script is not one there is, whose
// trade is one the script has no words for, or that carries a trade
// twice. The refusal names what there is.
func (lettering Lettering) Check() error {
	drawing, err := newDrawer(signCulture, 0)
	if err != nil {
		return err
	}
	return lettering.refusal(drawing)
}

// refusal is Check's refusal of a lettering, by a drawer of the sign
// culture.
func (lettering Lettering) refusal(drawing *drawer) error {
	if _, known := scriptSuffix[lettering.script()]; !known {
		return fmt.Errorf("cities: no script %q for signs, only %v",
			lettering.Script, ScriptNames())
	}
	suffix := lettering.suffix()
	for index, trade := range lettering.Carries {
		// the Latin words of a trade carry no suffix, so a trade named
		// for another script's words has them under its own name
		if !drawing.has("sign-word-"+trade+suffix) ||
			!drawing.has("sign-short-"+trade+suffix) ||
			(suffix == "" && otherScript(trade)) {
			return fmt.Errorf("cities: signs in the %s script have no "+
				"trade %q to carry, only %v", lettering.script(), trade,
				carryable(drawing, suffix))
		}
		if slices.Contains(lettering.Carries[:index], trade) {
			return fmt.Errorf("cities: the trade %q is carried twice",
				trade)
		}
	}
	return nil
}

// CheckFree refuses a lettering that a sign on its own of a kind cannot
// keep: a sign on its own carries one trade at most, and only of a kind
// that reads the words of a trade, not a screen, whose lines are not a
// trade's.
func (lettering Lettering) CheckFree(kind MountKind) error {
	if len(lettering.Carries) == 0 {
		return nil
	}
	if len(lettering.Carries) > 1 || !slices.Contains(tradeMounts, kind) {
		return fmt.Errorf("cities: a sign on its own carries one trade, "+
			"and only if it is of the kinds %v, not %d on a %q",
			tradeMounts, len(lettering.Carries), kind)
	}
	return nil
}

// carryable is the trades the culture has words for in the script of a
// suffix, by name, in alphabetical order.
func carryable(drawing *drawer, suffix string) []string {
	var trades []string
	for name := range drawing.culture.elements {
		trade, found := strings.CutPrefix(name, "sign-word-")
		if !found || !strings.HasSuffix(trade, suffix) {
			continue
		}
		trade = strings.TrimSuffix(trade, suffix)
		if suffix == "" && otherScript(trade) {
			continue
		}
		trades = append(trades, trade)
	}
	slices.Sort(trades)
	return trades
}

// otherScript is whether the name of a trade ends in the suffix of a
// script that is not the Latin, which makes it a trade of that script and
// not of the Latin.
func otherScript(trade string) bool {
	for _, suffix := range scriptSuffix {
		if suffix != "" && strings.HasSuffix(trade, suffix) {
			return true
		}
	}
	return false
}

// ScriptFaces are the names of the fonts the signs of a script are set
// in, which the culture says and a shelf keeps as files. The Latin script
// is set in go's own fonts, which are no files, and has none.
func ScriptFaces(script Script) ([]string, error) {
	suffix, known := scriptSuffix[script]
	if !known {
		return nil, fmt.Errorf("cities: no script %q for signs, only %v",
			script, ScriptNames())
	}
	if suffix == "" {
		return nil, nil
	}
	drawing, err := newDrawer(signCulture, 0)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, look := range signLooks {
		for _, one := range drawing.options("sign-" + look + "-font" +
			suffix) {
			if !slices.Contains(names, one.Name) {
				names = append(names, one.Name)
			}
		}
	}
	return names, drawing.trouble
}

// owing is the places of a building's offer that are given to the trades
// it must carry: for each trade in turn, the first place of the kinds that
// read a trade, taking the kinds in the order of tradeMounts, that no trade
// has yet. A trade for which there is no place is given none.
func owing(mounts []signMount, carries []string) map[int]string {
	given := map[int]string{}
	for _, kind := range tradeMounts {
		for index, mount := range mounts {
			if len(given) < len(carries) && mount.Kind == kind {
				given[index] = carries[len(given)]
			}
		}
	}
	return given
}

// load is the font of a name: one of go's own for the Latin script, and
// for any other the file the lettering gives it.
func (signing *signer) load(name string) (*typography.Font, error) {
	if signing.suffix == "" {
		return typography.Named(name, "")
	}
	path, given := signing.lettering.Faces[name]
	if !given {
		return nil, fmt.Errorf("cities: the signs are in the %s script, "+
			"set in the face %s, and no file was given for it",
			signing.lettering.script(), name)
	}
	return typography.Load(path)
}

// sets is whether the face of a name can draw every letter of the words.
// A face may refuse a letter, as typography refuses a glyph whose contours
// cross or touch, and many of the letters of a script of thousands are
// drawn so; the verdict on each letter is kept, face by face.
func (signing *signer) sets(name, words string) bool {
	face := signing.face(name)
	if face == nil {
		return false
	}
	refused, found := signing.refused[name]
	if !found {
		refused = map[rune]bool{}
		signing.refused[name] = refused
	}
	for _, letter := range words {
		verdict, known := refused[letter]
		if !known {
			_, err := typography.Outlines(face, typography.Text{
				Words: string(letter), Size: 1,
				Tolerance: signing.tolerance(face)})
			verdict = err != nil
			refused[letter] = verdict
		}
		if verdict {
			return false
		}
	}
	return true
}

// curveUnits is the units of an em the culture's curve tolerance is said
// in, which are go's fonts': a face of another em has the tolerance in
// proportion, as far from the curve as it is of go's.
const curveUnits = 2048

// tolerance is how far a straight step of a letter's outline may stray
// from the curve, in the units of a face: the culture's, scaled from the
// em of go's fonts to the face's own.
func (signing *signer) tolerance(face *typography.Font) float64 {
	return signing.drawing.fixed("sign-curve-tolerance") * face.Units() /
		curveUnits
}

// tolerance is the curve tolerance in the units of the sign's face.
func (job signJob) tolerance() float64 {
	return job.signing.tolerance(job.faceFor())
}

// faceFor is the face the sign is set in: its own, or, in a script that
// has refusals, the first of the other faces its look may be set in that
// can draw every letter of all the words the sign says, so that one sign
// is of one face. If none can, it is the sign's own, and drawing the words
// says which letter it refuses. It is asked after the words are said.
func (job signJob) faceFor() *typography.Font {
	if job.signing.suffix == "" {
		return job.face
	}
	words := strings.Join(*job.said, "")
	for _, name := range append([]string{job.faceName}, job.spares...) {
		if job.signing.sets(name, words) {
			return job.signing.face(name)
		}
	}
	return job.face
}

// faced is the job with the face its look is set in, drawn from the
// culture's fonts for the look in the script of the building, and the
// names of the other faces the look may be set in.
func (job signJob) faced(look string) signJob {
	font := "sign-" + look + "-font" + job.signing.suffix
	options := job.drawing.options(font)
	job.faceName = job.drawing.pick(font, options, job.at).Name
	job.face = job.signing.face(job.faceName)
	job.spares = nil
	for _, one := range options {
		if one.Name != job.faceName {
			job.spares = append(job.spares, one.Name)
		}
	}
	return job
}

// inScript is the options of an element in the script of the building:
// those of the element's name with the script's suffix if the culture says
// the element again for the script, and otherwise the element's own. A
// script says again only what its letters make differ.
func (signing *signer) inScript(name string) []option {
	if signing.suffix != "" && signing.drawing.has(name+signing.suffix) {
		return signing.drawing.options(name + signing.suffix)
	}
	return signing.drawing.options(name)
}

// trade is the trade of the sign at a place: the one the building must
// carry there, if it must, and otherwise one drawn by weight from the
// trades of its script. The draw is of the same stream whatever the
// script, so that a sign is for the same trade in every script that has
// the trades of the culture.
func (signing *signer) trade(at place, index int) option {
	if name, owes := signing.owing[index]; owes {
		return option{Name: name}
	}
	drawing := signing.drawing
	return drawing.pick("sign-trade", drawing.options("sign-trade"+
		signing.suffix), at)
}

// rise is how far a script's letters are set above their baseline, as a
// share of the em, so that the ink of a letter stands on the line the sign
// is laid out by. Latin capitals stand on their baseline and are set
// where they are; the characters of the simplified Chinese reach a tenth
// of an em below it.
func (signing *signer) rise() float64 {
	if signing.suffix == "" {
		return 0
	}
	return signing.drawing.fixed("sign-rise" + signing.suffix)
}
