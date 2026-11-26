package branching

import (
	"errors"
	"reflect"
	"testing"
)

type fragment struct{ size, depth int }

func TestExpandBuildsAFragmentTreeInBreadthFirstOrder(t *testing.T) {
	// A fragment can split unequally; smaller fragments become leaves.
	nodes, leaves, err := Expand([]fragment{{8, 0}}, 7,
		func(f fragment, _ int) (int, error) {
			if f.size < 3 {
				return 0, nil
			}
			return 2, nil
		},
		func(f fragment, _ int, child int) (fragment, error) {
			if child == 0 {
				return fragment{f.size / 2, f.depth + 1}, nil
			}
			return fragment{f.size - f.size/2 - 1, f.depth + 1}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	want := []fragment{{8, 0}, {4, 1}, {3, 1}, {2, 2}, {1, 2}, {1, 2}, {1, 2}}
	if !reflect.DeepEqual(nodes, want) || !reflect.DeepEqual(leaves, []int{3, 4, 5, 6}) {
		t.Fatalf("nodes %v leaves %v", nodes, leaves)
	}
}

func TestExpandRefusesInvalidOrExcessiveTrees(t *testing.T) {
	branches := func(f fragment, _ int) (int, error) { return f.size, nil }
	child := func(f fragment, _, _ int) (fragment, error) { return fragment{}, nil }
	for _, tc := range []struct {
		name     string
		roots    []fragment
		limit    int
		branches func(fragment, int) (int, error)
		child    func(fragment, int, int) (fragment, error)
	}{
		{"no budget", []fragment{{1, 0}}, 0, branches, child},
		{"too many roots", []fragment{{1, 0}, {1, 0}}, 1, branches, child},
		{"missing count", []fragment{{1, 0}}, 2, nil, child},
		{"missing child", []fragment{{1, 0}}, 2, branches, nil},
		{"negative count", []fragment{{-1, 0}}, 2, branches, child},
		{"budget", []fragment{{2, 0}}, 2, branches, child},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Expand(tc.roots, tc.limit, tc.branches, tc.child); err == nil {
				t.Fatal("accepted invalid expansion")
			}
		})
	}
}

func TestExpandPreservesErrorsAndDoesNotChangeRoots(t *testing.T) {
	roots := []fragment{{2, 0}}
	stop := errors.New("stop")
	_, _, err := Expand(roots, 3, func(fragment, int) (int, error) { return 1, nil }, func(fragment, int, int) (fragment, error) { return fragment{}, stop })
	if !errors.Is(err, stop) {
		t.Fatalf("lost error: %v", err)
	}
	if !reflect.DeepEqual(roots, []fragment{{2, 0}}) {
		t.Fatal("changed caller roots")
	}
	_, _, err = Expand(roots, 3, func(fragment, int) (int, error) { return 0, stop }, func(fragment, int, int) (fragment, error) { return fragment{}, nil })
	if !errors.Is(err, stop) {
		t.Fatalf("lost count error: %v", err)
	}
}

// A caller can use a semantic path instead of the traversal index as identity.
func TestExpandAllowsCallerOwnedPaths(t *testing.T) {
	type node struct{ path string }
	grow := func(roots []node) map[string]bool {
		t.Helper()
		nodes, _, err := Expand(roots, 6,
			func(n node, _ int) (int, error) {
				if len(n.path) == 1 {
					return 2, nil
				}
				return 0, nil
			},
			func(n node, _, child int) (node, error) {
				if child == 0 {
					return node{n.path + "L"}, nil
				}
				return node{n.path + "R"}, nil
			})
		if err != nil {
			t.Fatal(err)
		}
		paths := make(map[string]bool, len(nodes))
		for _, n := range nodes {
			paths[n.path] = true
		}
		return paths
	}
	a, b := grow([]node{{"A"}, {"B"}}), grow([]node{{"B"}, {"A"}})
	if !reflect.DeepEqual(a, b) || !a["AL"] || !a["AR"] || !a["BL"] || !a["BR"] {
		t.Fatalf("paths %v %v", a, b)
	}
}

// A rejected generation cannot allocate any of its children.
func TestExpandChecksBudgetBeforeBuildingChildren(t *testing.T) {
	built := 0
	nodes, leaves, err := Expand([]int{1}, 1,
		func(int, int) (int, error) { return 1, nil },
		func(int, int, int) (int, error) { built++; return 2, nil })
	if !errors.Is(err, ErrBudget) || built != 0 || nodes != nil || leaves != nil {
		t.Fatalf("nodes %v leaves %v built %d error %v", nodes, leaves, built, err)
	}
}
