package cities

import (
	"strings"
	"testing"
)

func TestTheCreatureDataHoldsTheFiveCreaturesInTheOrderItWritesThem(t *testing.T) {
	names, err := GargoyleCreatures()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"allosaurus", "orca", "pliosaur", "squid", "hatzegopteryx"}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("the creatures are %v, want %v", names, want)
	}
}

func TestEveryCreatureHasASizeASpoutAndAHeldPointOfItsOwn(t *testing.T) {
	rows, err := GargoyleCreatureRows()
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range rows {
		if !(each.Scale > 0) || each.Study == "" || each.By == "" {
			t.Errorf("%s: scale %v, study %q, picked by %q", each.Name,
				each.Scale, each.Study, each.By)
		}
		if each.MouthAt() == each.HeldAt() {
			t.Errorf("%s: its spout is the point it is held at", each.Name)
		}
	}
}

func TestACreatureNamedIsFoundAndOneThatIsNotIsRefusedBySayingWhichThereAre(t *testing.T) {
	found, err := CreatureNamed("hatzegopteryx")
	if err != nil || found.Name != "hatzegopteryx" {
		t.Fatalf("%+v, %v", found, err)
	}
	_, err = CreatureNamed("dodo")
	if err == nil || !strings.Contains(err.Error(), "hatzegopteryx") {
		t.Errorf("a dodo was found, or the refusal names no creature: %v", err)
	}
}

func TestEveryCreatureMakesAKindName(t *testing.T) {
	names, _ := GargoyleCreatures()
	for _, name := range names {
		if got := GargoyleKind(name, "granite"); got !=
			"gargoyle-"+name+"-granite" {
			t.Errorf("%s makes the kind %q", name, got)
		}
	}
}
