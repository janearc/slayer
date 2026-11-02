package main

import (
	"strings"
	"testing"
)

const small = `package p

func f(a int) bool {
	s := "a < b" // x == y
	return a < 3 && true
}
`

func TestEveryOperatorWordAndNumberCanChange(t *testing.T) {
	var got []string
	for _, one := range changes([]byte(small), 0, 1<<30) {
		got = append(got, one.from+">"+one.to)
	}
	want := []string{"<><=", "3>4", "&&>||", "true>false"}
	if len(got) != len(want) {
		t.Fatalf("changes %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("change %d is %s, want %s", index, got[index],
				want[index])
		}
	}
}

func TestAChangeIsMadeWhereItWas(t *testing.T) {
	all := changes([]byte(small), 5, 5)
	made := string(all[0].apply([]byte(small)))
	if want := "return a <= 3 && true"; !strings.Contains(made, want) {
		t.Errorf("made\n%s\nwant %q in it", made, want)
	}
}

func TestOnlyTheLinesAskedForChange(t *testing.T) {
	if got := changes([]byte(small), 3, 4); len(got) != 0 {
		t.Errorf("lines 3 to 4 change %d things, want none", len(got))
	}
}
