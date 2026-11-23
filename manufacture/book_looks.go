package manufacture

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// bookMetals are the metals a looks file may name, by the word it writes.
var bookMetals = map[string]func() *thing.Metal{
	"aluminium": thing.Aluminium,
}

// bookLookColumns are the numeric columns of a looks file and the unit each
// is judged against.
var bookLookColumns = map[string]string{
	"red": "share", "green": "share", "blue": "share",
	"gloss": "share", "index": "index",
}

// readLookNumbers are the numbers of a row of a looks file by column, each
// judged against its unit, or every reason the row is refused.
func readLookNumbers(file string, row map[string]string,
	units bookUnitsByName,
) (map[string]float64, error) {
	var err error
	read := map[string]float64{}
	for column, unit := range bookLookColumns {
		value, readErr := readFloat(row[column])
		err = errors.Join(err, readErr)
		if judged := units.judge(unit, value); readErr == nil &&
			judged != nil {
			err = errors.Join(err, fmt.Errorf("%s: %s %s %w", file,
				row["name"], column, judged))
		}
		read[column] = value
	}
	return read, err
}

// A bookLook is one row of a looks file, checked: what a slot of a model is
// dressed in.
type bookLook struct {
	place  slot
	name   string
	colour maths.Light
	gloss  float64
	index  float64
	metal  string
}

// material is a new material of the row, which its caller owns.
func (look bookLook) material() *thing.Material {
	material := thing.Polished(look.name, look.colour, look.gloss,
		look.index)
	if look.metal != "" {
		material.Metal = bookMetals[look.metal]()
	}
	return material
}

// bookLooks are the rows of a looks file in the order written.
type bookLooks []bookLook

// dress puts a new material of each row into the slot the row names.
func (all bookLooks) dress(into looks) {
	for _, look := range all {
		into[look.place] = look.material()
	}
}

// readLooks is the rows of a looks file, or every reason it is broken. The
// slots are the words the model gives its slots; a row naming another is
// refused.
func readLooks(file string, raw []byte, units bookUnitsByName,
	slots map[string]slot,
) (bookLooks, error) {
	rows, err := readRows(file, raw, []string{"slot", "name", "red",
		"green", "blue", "gloss", "index", "metal", "from", "says"})
	var read bookLooks
	dressed := looks{}
	for _, row := range rows {
		number, refusal := readLookNumbers(file, row, units)
		refusal = errors.Join(refusal, lookRefusal(file, row, slots,
			dressed))
		err = errors.Join(err, refusal)
		if refusal == nil {
			dressed[slots[row["slot"]]] = nil
			look := bookLook{name: row["name"], metal: row["metal"]}
			look.place = slots[row["slot"]]
			look.colour = maths.RGB(number["red"], number["green"],
				number["blue"])
			look.gloss, look.index = number["gloss"], number["index"]
			read = append(read, look)
		}
	}
	return read, err
}

// lookRefusal is why the row of a looks file is not one the model can use:
// a slot it does not have or has already dressed, a metal nobody knows, or
// no word on where it came from.
func lookRefusal(file string, row map[string]string, slots map[string]slot,
	made looks,
) error {
	place, known := slots[row["slot"]]
	var err error
	if !known {
		err = fmt.Errorf("%s: %q is not a slot of this model", file,
			row["slot"])
	} else if _, dressed := made[place]; dressed {
		err = fmt.Errorf("%s: slot %s is written twice", file, row["slot"])
	}
	if _, ok := bookMetals[row["metal"]]; !ok && row["metal"] != "" {
		err = errors.Join(err, fmt.Errorf("%s: %s names metal %q, which "+
			"the book does not know", file, row["name"], row["metal"]))
	}
	if row["from"] == "" || row["says"] == "" || row["name"] == "" {
		err = errors.Join(err, fmt.Errorf("%s: a row needs a name, where "+
			"it came from, and what it says", file))
	}
	return err
}
