package shape

import "testing"

func TestGroupMemberCountChecksCapacityBeforeAllocatingMemberStorage(t *testing.T) {
	const maximum = 1_073_741_823 // Existing two-slots-per-member 32-bit budget.
	for _, row := range []struct {
		count int
		want  bool
	}{{0, false}, {1, true}, {maximum, true}, {maximum + 1, false}} {
		if got := groupMemberCount(row.count); got != row.want {
			t.Fatalf("%d members: accepted %t, want %t", row.count, got, row.want)
		}
	}
}
