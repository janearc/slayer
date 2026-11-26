package manufacture

import (
	"fmt"
	"testing"
)

// Historic labels identify independently captured witnesses in tests only;
// production callers select a checked species, pose, representation or study.
func referencePinnipedKind(name string) (Kind, error) {
	book := defaultPinnipedProfiles
	switch name {
	case "bay-seal":
		return book.NewPinnipedStudy("harbor-seal", "bay")
	case "harbor-seal":
		return book.NewPinnipedStudy("harbor-seal", "detailed-wet")
	case "harbor-seal-dry":
		return book.NewPinnipedStudy("harbor-seal", "detailed-dry")
	case "harbor-seal-hauled":
		return book.NewPinniped("harbor-seal", "hauled")
	case "harbor-seal-swimming":
		return book.NewPinniped("harbor-seal", "swimming")
	case "harbor-seal-bones":
		return book.NewPinnipedRepresentation("harbor-seal", "", "bones")
	case "hawaiian-monk-seal-hauled":
		return book.NewPinniped("hawaiian-monk-seal", "hauled")
	case "walrus-fresh-5":
		return book.NewPinniped("walrus", "lying")
	case "walrus-fresh-5-hauled-out":
		return book.NewPinniped("walrus", "hauled-out")
	case "walrus-fresh-5-swimming":
		return book.NewPinniped("walrus", "swimming")
	case "walrus-fresh-5-skeleton":
		return book.NewPinnipedRepresentation("walrus", "", "bones")
	case "walrus-fresh-5-volumes":
		return book.NewPinnipedRepresentation("walrus", "", "volumes")
	default:
		return Kind{}, fmt.Errorf("no pinniped reference %q", name)
	}
}

func referenceNativeKind(name string) (Kind, error) {
	if name == "orca-clean-05" {
		return defaultWhaleProfiles.NewOrca("swim", "")
	}
	return referencePinnipedKind(name)
}

func TestFormerPinnipedKindsAreNotRegistered(t *testing.T) {
	for _, name := range []string{
		"bay-seal", "harbor-seal", "harbor-seal-dry", "harbor-seal-bones",
		"harbor-seal-hauled", "harbor-seal-swimming", "hawaiian-monk-seal-hauled",
		"walrus-fresh-5", "walrus-fresh-5-skeleton", "walrus-fresh-5-volumes",
		"walrus-fresh-5-hauled-out", "walrus-fresh-5-swimming",
	} {
		if _, err := KindNamed(name); err == nil {
			t.Errorf("former kind %q still resolves", name)
		}
		if _, err := referencePinnipedKind(name); err != nil {
			t.Errorf("checked replacement for %q: %v", name, err)
		}
	}
	if _, err := KindNamed("can"); err != nil {
		t.Fatalf("valid registered control: %v", err)
	}
}
