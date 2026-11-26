package ferns

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func specimen() Profile {
	return Profile{Kind: "fern", Status: "picked", By: "study", Date: "2026",
		FrondLength: FloatRange{.4, .9}, StalkRadius: FloatRange{.005, .008},
		Fronds: CountRange{6, 9}, Leaflets: CountRange{12, 24},
		Colour:  maths.RGB(.2, .5, .12),
		Anatomy: Anatomy{.03, 3, .15, .2, .22, .85, 3, .01}}
}

func TestBookCopiesAndChecksSpecies(t *testing.T) {
	var absent *Book
	if _, ok := absent.Get("fern"); ok || len(absent.Species()) != 0 {
		t.Fatal("nil book reports a species")
	}
	if _, err := Read("fern.yaml", nil); err == nil {
		t.Fatal("accepted an empty species book")
	}
	rows := []Profile{specimen()}
	book, err := newBook("fern.yaml", rows)
	if err != nil {
		t.Fatal(err)
	}
	rows[0].Colour.R = .9
	got, ok := book.Get("fern")
	if !ok || got.Colour.R != .2 {
		t.Fatalf("profile or lookup changed: %+v", got)
	}
	if _, ok := book.Get("absent"); ok {
		t.Fatal("found a missing profile")
	}
	got.Colour.R = .8
	again, _ := book.Get("fern")
	if again.Colour.R != .2 {
		t.Fatal("a returned profile changed its book")
	}
	if names := book.Species(); len(names) != 1 || names[0] != "fern" {
		t.Fatalf("species: %v", names)
	}
	if _, err := newBook("fern.yaml", []Profile{specimen(), specimen()}); err == nil {
		t.Fatal("accepted duplicate species")
	}
	invalid := specimen()
	invalid.FrondLength.Minimum = math.Inf(1)
	if _, err := newBook("fern.yaml", []Profile{invalid}); err == nil {
		t.Fatal("accepted nonfinite fern")
	}
	invalid = specimen()
	invalid.Fronds.Maximum, invalid.Leaflets.Maximum = 65, 65
	if _, err := newBook("fern.yaml", []Profile{invalid}); err == nil {
		t.Fatal("accepted foliage beyond the construction budget")
	}
}

func TestGrowthHabitKeepsFernLeafletsPairedAndTapered(t *testing.T) {
	habit := specimen().Anatomy.habit()
	if err := habit.Sound(); err != nil {
		t.Fatal(err)
	}
	if habit.Turn.InRadians() != math.Pi || habit.Fewest != 1 ||
		habit.Reach != .22 || habit.Shortening != .85 {
		t.Fatalf("fern frond habit changed: %+v", habit)
	}
}
