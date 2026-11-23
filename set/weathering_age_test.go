package set

import (
	"os"
	"strings"
	"testing"
)

// accretedSet is the set whose weathering age follows its block.
const accretedSet = "../sets/superblock-accreted.yaml"

// followsText is how that set writes the age it follows.
const followsText = "        from: block\n        planned: 5\n" +
	"        accreted: 45\n"

// follows is the age block of a set, each key written only where it is
// given.
func follows(from, planned, accreted string) string {
	var text string
	for key, value := range map[string]string{"from": from,
		"planned": planned, "accreted": accreted} {
		if value != "" {
			text += "        " + key + ": " + value + "\n"
		}
	}
	return text
}

// readAccreted is the accreted set's text, and the set read from it with
// the replacement made.
func readAccreted(t *testing.T, was, now string) (Weathering, error) {
	t.Helper()
	raw, err := os.ReadFile(accretedSet)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, was) {
		t.Fatalf("the accreted set has no %q to change", was)
	}
	edited := strings.Replace(text, was, now,
		1)
	read, err := Read(accretedSet, []byte(edited))
	if err != nil {
		return Weathering{}, err
	}
	world, _ := read.World()
	layer, _ := world.Weathering()
	return layer, nil
}

func TestAnAgeThatFollowsAnAccretionIsRead(t *testing.T) {
	layer, err := readAccreted(t, followsText, followsText)
	if err != nil {
		t.Fatal(err)
	}
	thing, planned, accreted, follows := layer.AgeFollows()
	if thing != "block" || planned != 5 || accreted != 45 || !follows {
		t.Errorf("the age follows %q from %v to %v, %v", thing, planned,
			accreted, follows)
	}
	if layer.Age() != 0 {
		t.Errorf("an age that follows has %v years of its own", layer.Age())
	}
}

func TestAnAgeThatIsANumberFollowsNothing(t *testing.T) {
	layer, err := readAccreted(t, followsText, "")
	if err == nil {
		t.Fatalf("a weathering with no age was read: %v", layer)
	}
	layer, err = readAccreted(t, "      age:\n"+followsText, "      age: 12\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, follows := layer.AgeFollows(); follows || layer.Age() != 12 {
		t.Errorf("an age of 12 follows %v and reads %v", follows, layer.Age())
	}
}

func TestAnAgeThatFollowsAnAccretionRefusesWhatIsWrongWithIt(t *testing.T) {
	flawed := map[string]string{
		"no thing":      follows("", "5", "45"),
		"no planned":    follows("block", "", "45"),
		"no accreted":   follows("block", "5", ""),
		"backwards":     follows("block", "45", "5"),
		"negative":      follows("block", "-1", "45"),
		"another key":   followsText + "        oldest: 90\n",
		"a nameless":    follows(`""`, "5", "45"),
		"words for age": follows("block", "old", "45"),
	}
	for name, text := range flawed {
		if _, err := readAccreted(t, followsText, text); err == nil {
			t.Errorf("%s: %q was read", name, text)
		}
	}
}
