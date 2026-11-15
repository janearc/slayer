package diskcache

import (
	"strings"
	"testing"
)

func TestNewKindDeclaresAKind(t *testing.T) {
	made := testKind(t, "kind-declared")
	found, err := KindNamed("kind-declared")
	if err != nil || found != made || made.Cap() != testCap ||
		made.Why() != "a test's" || made.Name() != "kind-declared" {
		t.Fatalf("declared %v, found %v, %v", made, found, err)
	}
}

// anti-test of TestNewKindDeclaresAKind
func TestNewKindRefusesNamesThatAreNotWords(t *testing.T) {
	for _, name := range []string{"", "Upper", "a/b", "-lead", "trail-",
		"two words", "dot.ted"} {
		if _, err := NewKind(name, testCap, "why"); err == nil {
			t.Errorf("kind %q was declared", name)
		}
	}
}

// anti-test of TestNewKindDeclaresAKind
func TestNewKindRefusesNoCapAndNoReason(t *testing.T) {
	for _, each := range []struct {
		cap int64
		why string
	}{{0, "why"}, {-1, "why"}, {testCap, " "}} {
		if _, err := NewKind("kind-uncapped", each.cap, each.why); err == nil {
			t.Errorf("cap %d, reason %q was declared", each.cap, each.why)
		}
	}
}

func TestNewKindTakesTheSameDeclarationTwice(t *testing.T) {
	testKind(t, "kind-twice")
	if _, err := NewKind("kind-twice", testCap, "a test's"); err != nil {
		t.Fatalf("the same declaration twice was refused: %v", err)
	}
}

// anti-test of TestNewKindTakesTheSameDeclarationTwice
func TestNewKindRefusesOneNameWithTwoCaps(t *testing.T) {
	testKind(t, "kind-two-caps")
	if _, err := NewKind("kind-two-caps", testCap+1, "a test's"); err == nil {
		t.Fatal("one name was declared with two caps")
	}
}

// anti-test of TestMustKindDeclaresAGoodOne
func TestMustKindPanicsOnAMistake(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a bad declaration did not panic")
		}
	}()
	MustKind("Bad Name", testCap, "why")
}

func TestMustKindDeclaresAGoodOne(t *testing.T) {
	if made := MustKind("kind-must", testCap, "why"); made.Name() != "kind-must" {
		t.Fatalf("declared %v", made)
	}
}

func TestKindsAreInNameOrder(t *testing.T) {
	testKind(t, "kind-order-b")
	testKind(t, "kind-order-a")
	var names []string
	for _, each := range Kinds() {
		names = append(names, each.Name())
	}
	listed := strings.Join(names, " ")
	if !strings.Contains(listed, "kind-order-a kind-order-b") {
		t.Fatalf("kinds %s", listed)
	}
}

// anti-test of TestNewKindDeclaresAKind
func TestKindNamedRefusesAnUndeclaredName(t *testing.T) {
	testKind(t, "kind-known")
	_, err := KindNamed("kind-never-declared")
	if err == nil || !strings.Contains(err.Error(), "kind-known") {
		t.Fatalf("an undeclared kind was found, or the kinds unsaid: %v",
			err)
	}
}

// anti-test of TestKindsAreInNameOrder
func TestKindsDoNotListARefusedDeclaration(t *testing.T) {
	if _, err := NewKind("kind-refused", 0, "why"); err == nil {
		t.Fatal("a kind of no cap was declared")
	}
	for _, each := range Kinds() {
		if each.Name() == "kind-refused" {
			t.Fatal("a refused declaration was listed")
		}
	}
}
