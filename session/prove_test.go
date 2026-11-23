package session

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/fermat"
)

// -pixel is taken once or many times, in order, and -many with it;
// both are kept as asked.
func TestThePixelsAndHowManyAreTaken(t *testing.T) {
	given := mustParse(t, "-prove", "a.pfm", "-pixel", "3,4", "-pixel",
		" 0 , 0", "-many", "5")
	if want := (named{{3, 4}, {0, 0}}); !reflect.DeepEqual(given.pixels,
		want) || given.many != 5 {
		t.Errorf("pixels %v and many %d, not %v and 5", given.pixels,
			given.many, want)
	}
	if said := given.pixels.String(); said != "3,4;0,0" {
		t.Errorf("the pixels say %q", said)
	}
	if plain := mustParse(t); plain.pixels != nil ||
		plain.many != fermat.Many {
		t.Errorf("unasked, pixels %v and many %d", plain.pixels, plain.many)
	}
}

// a pixel or a count that cannot be proved is refused by its flag: one that is
// not X,Y, one outside the picture, either without -prove, a count less than
// zero, and a proof of nothing at all.
func TestWhatCannotBeProvedIsRefusedAtTheFlags(t *testing.T) {
	refused := map[string][][]string{
		"-pixel": {{"-prove", "a.pfm", "-pixel", "3"},
			{"-prove", "a.pfm", "-pixel", "a,b"},
			{"-prove", "a.pfm", "-pixel", "96,0"},
			{"-prove", "a.pfm", "-pixel", "0,-1"},
			{"-pixel", "3,4"}},
		"-many": {{"-prove", "a.pfm", "-many", "-1"},
			{"-prove", "a.pfm", "-many", "0"},
			{"-many", "5"}},
	}
	for flag, cases := range refused {
		for _, arguments := range cases {
			_, err := parse(append([]string{"-w", "96", "-h", "60"},
				arguments...), io.Discard)
			if err == nil || !strings.Contains(err.Error(), flag[1:]) {
				t.Errorf("%v: %v, naming %s", arguments, err, flag)
			}
		}
	}
	_, err := parse([]string{"-prove", "a.pfm", "-many", "0", "-pixel",
		"1,1"}, io.Discard)
	if err != nil {
		t.Errorf("a named pixel and no picks was refused: %v", err)
	}
	if _, err := parse([]string{"-pixel", "3,4"}, io.Discard); !errors.Is(err,
		errAsked) {
		t.Errorf("-pixel without -prove: %v, not %v", err, errAsked)
	}
}
