package manufacture

import "testing"

// These integer branch counts are authored specimen choices, not field data.
// Exercise every promised count and forbid an extra one across seeded draws.
func TestRedwoodFamilyKeepsItsAuthoredLowestWhorlRange(t *testing.T) {
	for name, row := range map[string]conifer{
		"coast-redwood": redwoodRow(), "giant-sequoia": sequoiaRow(),
	} {
		t.Run(name, func(t *testing.T) {
			seen := map[int]bool{}
			for number := uint64(0); number < 64; number++ {
				count := row.lowest.count(Sow(number).draw(picking, 3, 0))
				if count < 3 || count > 5 {
					t.Fatalf("seed %d: lowest whorl has %d branches, want 3..5", number, count)
				}
				seen[count] = true
			}
			for count := 3; count <= 5; count++ {
				if !seen[count] {
					t.Errorf("seeded family never produces %d lowest-whorl branches", count)
				}
			}
		})
	}
}
