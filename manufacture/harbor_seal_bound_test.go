package manufacture

import "testing"

func TestTheSealOnItsBonesHasItsOwnKinds(t *testing.T) {
	for _, name := range []string{"harbor-seal-hauled",
		"harbor-seal-swimming"} {
		kind, err := referencePinnipedKind(name)
		if err != nil {
			t.Fatal(err)
		}
		row, found := kind.how.(sealKind)
		if !found || row.form != sealLoft {
			t.Fatalf("%s is not the seal built on its bones", name)
		}
		if _, err := row.loft.Build(407); err != nil {
			t.Fatalf("%s's pose: %v", name, err)
		}
	}
}

func TestTheBaySealIsNotTheSealOnItsBones(t *testing.T) {
	bay, err := referencePinnipedKind("bay-seal")
	if err != nil {
		t.Fatal(err)
	}
	if row, ok := bay.how.(sealKind); !ok || row.form != sealBay {
		t.Fatal("bay-seal is no longer the bay scene's own seal")
	}
	if _, found := table()["bay-seal-swimming"]; found {
		t.Fatal("the seal on its bones still has a bay seal's name")
	}
}
