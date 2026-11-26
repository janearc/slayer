package manufacture

import "testing"

func TestSealLoftedKindNeedsASeed(t *testing.T) {
	if _, err := (sealKind{form: sealLoft}).made(seed{}); err == nil {
		t.Fatal("an unsown seed made a seal")
	}
}
func TestFormerMonkSwimmingKindIsAbsent(t *testing.T) {
	if _, err := KindNamed("hawaiian-monk-seal-swimming"); err == nil {
		t.Fatal("a monk seal kind nobody made was found")
	}
}
