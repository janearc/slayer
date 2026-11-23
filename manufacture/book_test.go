package manufacture

import (
	"strings"
	"testing"
)

// bookTestNumbers is a numbers file of two single numbers.
const bookTestNumbers = `numbers:
  - {name: reach, value: 2.5, unit: metre, says: how far it reaches, kind: picked, by: chosen for the test}
  - {name: legs, value: 3, unit: count, says: how many legs it has, kind: picked, by: chosen for the test}
`

// bookTestPoints is a points file of one point and a table of two rows.
const bookTestPoints = `name,x,y,z,unit,from,says
corner,1,2,3,offset,chosen,a corner
row,1,1,1,offset,chosen,the first row
row,2,2,2,offset,chosen,the second row
`

// bookTestLooks is a looks file of one slot.
const bookTestLooks = `slot,name,red,green,blue,gloss,index,metal,from,says
body,a shell,0.2,0.3,0.4,0.5,1.5,aluminium,chosen,a shell
`

// bookTestRows are the names of the rows of the table in bookTestPoints, one
// for each row.
var bookTestRows = []string{"first", "second"}

// bookTestSlots are the slots of the looks of the test.
var bookTestSlots = map[string]slot{"body": body}

// bookTested is the book of the fixtures with a replacement made in them.
func bookTested(numbersFrom, numbersTo, pointsFrom, pointsTo string,
) (*book, error) {
	numbers := strings.ReplaceAll(bookTestNumbers, numbersFrom, numbersTo)
	points := strings.ReplaceAll(bookTestPoints, pointsFrom, pointsTo)
	return readBook("numbers.yaml", []byte(numbers), "points.csv",
		[]byte(points))
}

// bookAsked asks for every name of the fixtures and returns the asking.
func bookAsked(t *testing.T) *asking {
	t.Helper()
	read, err := bookTested("", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	ask := read.asking()
	ask.number("reach")
	ask.count("legs")
	ask.point("corner")
	ask.table("row")
	return ask
}

func TestABookOfWholeFilesAnswersEveryNameAndRefusesNone(t *testing.T) {
	ask := bookAsked(t)
	if err := ask.refusal(); err != nil {
		t.Fatalf("a book asked for everything it holds refused: %v", err)
	}
	if ask.failed() {
		t.Fatal("every name was held, and the asking says one was not")
	}
}

func TestANameTheBookLacksIsToldByName(t *testing.T) {
	ask := bookAsked(t)
	ask.number("breadth")
	if !ask.failed() {
		t.Fatal("a name the book lacks was not kept to be told")
	}
	err := ask.refusal()
	if err == nil || !strings.Contains(err.Error(), "breadth") {
		t.Fatalf("the refusal %v does not name breadth", err)
	}
}

func TestANameNothingAsksForIsToldByName(t *testing.T) {
	read, err := bookTested("", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	ask := read.asking()
	ask.number("reach")
	refusal := ask.refusal()
	if refusal == nil || !strings.Contains(refusal.Error(), "corner") {
		t.Fatalf("the refusal %v does not name corner, which no one asked "+
			"for", refusal)
	}
	if strings.Contains(refusal.Error(), "reach") {
		t.Fatalf("the refusal %v names reach, which was asked for", refusal)
	}
}

func TestAPointAndATableAreNotTakenForEachOther(t *testing.T) {
	ask := bookAsked(t)
	if len(ask.table("row")) != len(bookTestRows) {
		t.Fatal("a name written twice is not a table of its rows")
	}
	if ask.failed() {
		t.Fatal("the fixtures asked for their own names and failed")
	}
	ask.point("row")
	ask.table("corner")
	if len(ask.missing) != len(bookTestRows) {
		t.Fatalf("a table asked for as a point, and a point as a table, "+
			"were not both refused: %v", ask.missing)
	}
}

func TestABookWithANumberOutOfItsUnitIsRefused(t *testing.T) {
	for _, broken := range []struct {
		why, from, to, pointsFrom, pointsTo string
	}{
		{"too long for a metre", "value: 2.5", "value: 250", "", ""},
		{"not finite", "value: 2.5", "value: .inf", "", ""},
		{"not a number", "value: 2.5", "value: .nan", "", ""},
		{"unknown unit", "unit: metre", "unit: furlong", "", ""},
		{"count that is not whole", "value: 3", "value: 2.5", "", ""},
		{"point out of its unit", "", "", "1,2,3", "1,2,300"},
		{"point that is no number", "", "", "1,2,3", "1,2,three"},
	} {
		if _, err := bookTested(broken.from, broken.to, broken.pointsFrom,
			broken.pointsTo); err == nil {
			t.Errorf("%s: the book was not refused", broken.why)
		}
	}
	if _, err := bookTested("", "", "", ""); err != nil {
		t.Errorf("the whole book was refused: %v", err)
	}
}

func TestABookNamingTheSameThingTwiceIsRefused(t *testing.T) {
	_, err := bookTested("name: legs", "name: corner", "", "")
	if err == nil || !strings.Contains(err.Error(), "corner") {
		t.Fatalf("a name in both files gave %v; want it named", err)
	}
	if _, err := bookTested("name: legs", "name: reach", "", ""); err == nil {
		t.Fatal("a number written twice was not refused")
	}
}

func TestAPointsFileWithAnotherHeaderIsRefused(t *testing.T) {
	_, err := bookTested("", "", "name,x,y,z", "name,x,y")
	if err == nil {
		t.Fatal("a points file of another shape was not refused")
	}
}

func TestLooksAreReadAndDressNewMaterialsEachTime(t *testing.T) {
	read, err := readLooks("looks.csv", []byte(bookTestLooks), bookUnits,
		bookTestSlots)
	if err != nil {
		t.Fatal(err)
	}
	first, second := looks{}, looks{}
	read.dress(first)
	read.dress(second)
	if first[body] == nil || first[body] == second[body] {
		t.Fatal("dressing twice did not give two new materials")
	}
	if first[body].Metal == nil {
		t.Fatal("the metal the row names was not given")
	}
}

func TestLooksThatAreBrokenAreRefused(t *testing.T) {
	row := strings.TrimPrefix(bookTestLooks,
		"slot,name,red,green,blue,gloss,index,metal,from,says\n")
	for _, broken := range []struct{ why, from, to string }{
		{"slot the model lacks", "body,", "wing,"},
		{"metal nobody knows", "aluminium", "unobtainium"},
		{"channel above one", "0.2", "2.5"},
		{"index above its unit", "1.5", "15"},
		{"no word on its source", ",chosen,", ",,"},
		{"slot written twice", row, row + row},
	} {
		raw := strings.ReplaceAll(bookTestLooks, broken.from, broken.to)
		if _, err := readLooks("looks.csv", []byte(raw), bookUnits,
			bookTestSlots); err == nil {
			t.Errorf("%s: the looks were not refused", broken.why)
		}
	}
}
