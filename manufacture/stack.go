package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// A stack is a craft that is a body of revolution: a capsule, a service
// module, a stage, or a launcher made of them. It is data. stacks.yaml
// lists each as its sections from the bottom up, with whatever stands
// beside it, as a launcher's boosters do, and this file turns any such
// list into a thing standing on its own zero.

//go:embed stacks.yaml
var stacksRaw []byte

// stackSection is one length of a stack: how wide it is at its bottom
// and at its top, how long it is, what it is made of and how that was
// finished, and where its numbers were read. A painted section names
// its paint, one of the swatches, and no other section names one.
type stackSection struct {
	Part                   string
	Bottom, Top, Length    float64
	Material, Finish, Read string
	Paint                  string
	// made is the section's look, made once when the file is read and
	// held to be sound, so that building a stack afterwards has nothing
	// left to refuse.
	made *thing.Material
}

// painted is the finish the book calls paint.
const painted = "painted"

// look is a section's look: its material under its finish, or, painted,
// under paint of the swatch it names. A painted section with no swatch
// the swatches know, and a paint named on a section that is not painted,
// are refused, saying whose section it was.
func (each stackSection) look(whose string) (*thing.Material, error) {
	part := whose + "'s " + each.Part
	if each.Finish != painted {
		if each.Paint != "" {
			return nil, fmt.Errorf("manufacture: %s names the paint %q and "+
				"is not painted", part, each.Paint)
		}
		return surfaceOf(part, each.Material, each.Finish)
	}
	colour, known := swatches.Named(each.Paint)
	if !known {
		return nil, fmt.Errorf("manufacture: %s is painted %q, which is no "+
			"swatch", part, each.Paint)
	}
	kind, err := materials.ByName(each.Material)
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", part, err)
	}
	made, err := thing.Made(kind, thing.Painted(swatches.Light(colour)))
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", part, err)
	}
	return made, nil
}

// stackBeside is so many of one smaller stack standing round the main
// one, evenly, each against its foot's side: a launcher's boosters.
// Raised is how far above the main stack's foot their own feet are.
type stackBeside struct {
	Part     string
	Count    int
	Raised   float64
	Sections []stackSection
}

// stackRow is one craft: its name, what it is, its sections from the
// bottom up, and what stands beside it.
type stackRow struct {
	Name, Says string
	Sections   []stackSection
	Beside     []stackBeside
}

// stack is the maker of one such craft.
type stack struct{ row stackRow }

// stacksRead is every craft the file lists, read and checked once.
var stacksRead, stacksError = readStacks(stacksRaw)

// sound is whether a length or a width is a number above nought that
// can be built.
func sound(value float64) bool {
	return value > spaceZero && !math.IsInf(value, spaceOne)
}

// soundSections checks one run of sections, the main stack's or a
// booster's: at least one, each with a part named, somewhere its numbers
// were read, a width at each end and a length above nought, and a look
// for its material and finish.
func soundSections(whose string, sections []stackSection) error {
	if len(sections) == spaceZero {
		return fmt.Errorf("manufacture: stacks: %s has no sections", whose)
	}
	for at := range sections {
		each := &sections[at]
		if each.Part == "" || each.Read == "" {
			return fmt.Errorf("manufacture: stacks: %s has a section with "+
				"no part named, or nowhere its numbers were read", whose)
		}
		part := whose + "'s " + each.Part
		if !sound(each.Bottom) || !sound(each.Top) || !sound(each.Length) {
			return fmt.Errorf("manufacture: stacks: %s is %v wide at its "+
				"bottom, %v at its top and %v long", part, each.Bottom,
				each.Top, each.Length)
		}
		look, err := each.look(whose)
		if err != nil {
			return err
		}
		look.Name = each.Part
		each.made = look
	}
	return nil
}

// readStacks reads the craft a file lists and checks all of each, so
// that making one afterwards cannot fail: a name, no name twice, sound
// sections, and for whatever stands beside it a part named, a count
// above nought, a height that is a number not below nought, and sound
// sections of its own. Nonsense is refused, naming the craft.
func readStacks(raw []byte) ([]stackRow, error) {
	var read struct{ Stacks []stackRow }
	if err := yamlfile.Read("stacks.yaml", raw, &read); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, row := range read.Stacks {
		if row.Name == "" || seen[row.Name] {
			return nil, fmt.Errorf("manufacture: stacks: %q is no name, "+
				"or is a name twice", row.Name)
		}
		seen[row.Name] = true
		if err := soundSections(row.Name, row.Sections); err != nil {
			return nil, err
		}
		for _, each := range row.Beside {
			if each.Part == "" || each.Count <= spaceZero ||
				!(each.Raised >= spaceZero) || math.IsInf(each.Raised, spaceOne) {
				return nil, fmt.Errorf("manufacture: stacks: %s has %d of "+
					"%q beside it, raised %v", row.Name, each.Count,
					each.Part, each.Raised)
			}
			whose := row.Name + "'s " + each.Part
			if err := soundSections(whose, each.Sections); err != nil {
				return nil, err
			}
		}
	}
	return read.Stacks, nil
}

// stackRows supplies every craft the file lists, by name. A file that
// cannot be read is a broken build, and stops it here, as the other
// embedded data files do: a craft that quietly was no kind would be
// worse.
func stackRows() map[string]maker {
	if stacksError != nil {
		panic(stacksError)
	}
	rows := map[string]maker{}
	for _, row := range stacksRead {
		rows[row.Name] = stack{row: row}
	}
	return rows
}

// sizeOf is how wide a run of sections is at its widest, and how tall.
func sizeOf(sections []stackSection) (wide, tall float64) {
	for _, each := range sections {
		wide = math.Max(wide, math.Max(each.Bottom, each.Top))
		tall += each.Length
	}
	return wide, tall
}

// standoff is how far from the main stack's axis a thing beside it
// stands: the main stack's half width at its foot and the thing's own
// half width, so the two touch.
func (row stack) standoff(beside stackBeside) float64 {
	wide, _ := sizeOf(beside.Sections)
	return (row.row.Sections[spaceZero].Bottom + wide) * spaceHalf
}

// extent is a stack's box: as wide as its widest section and whatever
// stands beside it, and as tall as the tallest of them.
func (row stack) extent() maths.Box {
	wide, tall := sizeOf(row.row.Sections)
	out := wide * spaceHalf
	for _, each := range row.row.Beside {
		width, height := sizeOf(each.Sections)
		out = math.Max(out, row.standoff(each)+width*spaceHalf)
		tall = math.Max(tall, each.Raised+height)
	}
	return maths.SegmentBox(maths.V(-out, spaceZero, -out),
		maths.V(out, tall, out), spaceZero)
}

// made keeps quick as a stack's default detail.
func (row stack) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// stood turns a run of sections one on the next at a place, each a
// closed length about its own axis in its own look, the looks numbered
// from first, and gives back the number after the last. Each thing made
// has its own copy of a section's look.
func (row stack) stood(assembly *instrumentAssembly,
	sections []stackSection, at maths.Vec, first int,
) int {
	var foot float64
	for index, each := range sections {
		look := *each.made
		where := slot(first + index)
		assembly.object.dressed[where] = &look
		head := foot + each.Length
		assembly.add(turnedProfile{points: [][2]float64{
			{spaceZero, foot}, {each.Bottom * spaceHalf, foot},
			{each.Top * spaceHalf, head}, {spaceZero, head}}},
			maths.NewSubspace(at, maths.Identity, spaceOne), where)
		foot = head
	}
	return first + len(sections)
}

// madeAt builds the stack at a detail: its own sections on its axis,
// and each thing beside it so many times round, evenly.
func (row stack) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: looks{}}}
	next := row.stood(assembly, row.row.Sections,
		maths.V(spaceZero, spaceZero, spaceZero), spaceZero)
	for _, each := range row.row.Beside {
		out := row.standoff(each)
		after := next
		for one := range each.Count {
			round := fullTurn * float64(one) / float64(each.Count)
			at := maths.V(out*math.Cos(round), each.Raised,
				-out*math.Sin(round))
			after = row.stood(assembly, each.Sections, at, next)
		}
		next = after
	}
	object, err := assembly.finish(row.extent())
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}
