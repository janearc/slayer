package diskcache

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Kind is one kind of thing kept: its folder's name, its cap in bytes
// and why the cap is what it is.
type Kind struct {
	name string
	cap  int64
	why  string
}

// declared is every kind made in this process, by name.
var declared = struct {
	sync.Mutex
	kinds map[string]Kind
}{kinds: map[string]Kind{}}

// NewKind is a kind of a name, lower-case words joined by hyphens, with
// a cap of more than nothing and a reason for it; it is declared, so
// the verb lists it. A name declared already with another cap or reason
// is refused: one name is one folder, kept to one rule.
func NewKind(name string, cap int64, why string) (Kind, error) {
	if name == "" || strings.Trim(name, kindLetters) != "" ||
		strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return Kind{}, fmt.Errorf("diskcache: a kind's name is lower-case "+
			"words joined by hyphens, not %q", name)
	}
	if cap <= nothing || strings.TrimSpace(why) == "" {
		return Kind{}, fmt.Errorf("diskcache: kind %s needs a cap of more "+
			"than nothing and a reason for it", name)
	}
	made := Kind{name: name, cap: cap, why: why}
	declared.Lock()
	defer declared.Unlock()
	if before, has := declared.kinds[name]; has && before != made {
		return Kind{}, fmt.Errorf("diskcache: kind %s is declared already "+
			"with another cap or reason", name)
	}
	declared.kinds[name] = made
	return made, nil
}

// MustKind is NewKind for a package's own declaration, whose arguments
// are constants: a refusal there is a mistake in the program, found by
// its first test, so it panics.
func MustKind(name string, cap int64, why string) Kind {
	made, err := NewKind(name, cap, why)
	if err != nil {
		panic(err)
	}
	return made
}

// Kinds is every kind declared, in name order.
func Kinds() []Kind {
	declared.Lock()
	defer declared.Unlock()
	made := make([]Kind, nothing, len(declared.kinds))
	for _, each := range declared.kinds {
		made = append(made, each)
	}
	slices.SortFunc(made, func(one, other Kind) int {
		return strings.Compare(one.name, other.name)
	})
	return made
}

// KindNamed is the kind declared by a name, or a refusal naming the
// kinds there are.
func KindNamed(name string) (Kind, error) {
	declared.Lock()
	made, has := declared.kinds[name]
	declared.Unlock()
	if !has {
		var names []string
		for _, each := range Kinds() {
			names = append(names, each.name)
		}
		return Kind{}, fmt.Errorf("diskcache: no kind %q; the kinds are %s",
			name, strings.Join(names, ", "))
	}
	return made, nil
}

// Name is the kind's name, which is its folder's.
func (kind Kind) Name() string { return kind.name }

// Cap is how many bytes the kind keeps before it removes the least
// recently used.
func (kind Kind) Cap() int64 { return kind.cap }

// Why is the reason for the cap.
func (kind Kind) Why() string { return kind.why }
