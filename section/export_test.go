package section

import "github.com/janearc/slayer/maths"

// Keeps is keeps, for the tests of package section_test.
func (change Change) Keeps(sight Sight) bool { return change.keeps(sight) }

// Meets is meets, for the tests of package section_test.
func (sight Sight) Meets(centre maths.Vec, radius float64) bool {
	return sight.meets(centre, radius)
}
