package trace

import (
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// letGoAsked is a let-go a shot sets: the quasar slice's reach, picked.
const letGoAsked = 60

func TestAFieldWithNoLetGoKeepsTheTracersOwn(t *testing.T) {
	field := placedField{source: &thing.Field{Mass: 1, Reach: 40}}
	if field.letGo() != loose {
		t.Fatalf("a field with no let-go lets go at %v, not %v",
			field.letGo(), loose)
	}
}

func TestAFieldsOwnLetGoStandsOverTheTracers(t *testing.T) {
	field := placedField{source: &thing.Field{Mass: 1, Reach: 40,
		LetGo: letGoAsked}}
	if field.letGo() != letGoAsked {
		t.Fatalf("a field asking %v lets go at %v", letGoAsked,
			field.letGo())
	}
	negative := placedField{source: &thing.Field{Mass: 1, LetGo: -1}}
	if negative.letGo() != loose {
		t.Fatal("a negative let-go was taken")
	}
}
