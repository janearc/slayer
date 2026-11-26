package manufacture

import (
	birds "github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/theropods/birds"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

var birdRead = func() *birds.Book {
	book, err := birds.Read("birds.yaml", birdRaw)
	if err != nil {
		panic(err)
	}
	return book
}()
var birdStances = []string{birds.Standing, birds.Gliding, birds.Bones, birds.Outlined}

func birdThings(key, stance string, from seed, at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	bird, err := birdRead.New(key, stance)
	if err != nil {
		return nil, maths.Box{}, err
	}
	return bird.Build(from, at)
}

func TestBirdSpeciesAreNotRegistryAliases(t *testing.T) {
	rows := table()
	for _, key := range birdRead.Species() {
		for _, stance := range birdStances {
			if _, ok := rows["bird-"+key+"-"+stance]; ok {
				t.Errorf("registered old alias bird-%s-%s", key, stance)
			}
		}
	}
	if _, err := KindNamed("bird-grey-heron-standing"); err == nil {
		t.Fatal("old bird kind still resolves")
	}
}

func TestEveryBirdIsMadeInEveryStance(t *testing.T) {
	book, err := ReadBirdProfiles("birds.yaml", birdRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range birdRead.Species() {
		for _, stance := range birdStances {
			row, err := book.NewBird(key, stance)
			if err != nil {
				t.Fatal(err)
			}
			made, err := row.how.made(sow(7))
			if err != nil {
				t.Fatalf("%s %s: %v", key, stance, err)
			}
			if len(made.native) == 0 || made.extent == nil {
				t.Errorf("%s %s: nothing made", key, stance)
			}
		}
	}
}

func TestAnUnsownBirdIsRefused(t *testing.T) {
	if _, err := (birdKind{species: "grey-heron"}).made(seed{}); err == nil {
		t.Error("a bird was made from a seed never sown")
	}
}
