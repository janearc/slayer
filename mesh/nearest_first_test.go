package mesh

import "testing"

// TestNearestIsSafeBeforeAnyRay says that a mesh asked its nearest point
// before any ray has met it grows its tree for that and answers.
func TestNearestIsSafeBeforeAnyRay(t *testing.T) {
	ungrown, err := New(cubeCorners(), cubeFaces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	corners := cubeCorners()
	if got := ungrown.Distance(corners[first]); got != nought {
		t.Fatalf("distance to corner = %v", got)
	}
}
