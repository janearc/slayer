package shape

import (
	"strings"
	"testing"
)

// groupCountContractsFile is the counts of members a group is asked about,
// and whether each is accepted.
const groupCountContractsFile = "testdata/group-count-contracts-counts.csv"

// TestGroupMemberCountIsTheTwoSlotBudget says how many members a group may
// have: at least one, and at most the 32-bit hierarchy budget at two slots
// each.
func TestGroupMemberCountIsTheTwoSlotBudget(t *testing.T) {
	for _, row := range readSheet(t, groupCountContractsFile) {
		count := cellCount(t, row, "count")
		want := row["accepted"] == "true"
		if got := groupMemberCount(count); got != want {
			t.Fatalf("%d members: accepted %t, want %t", count, got, want)
		}
	}
}

// TestNewMeshGroupCountsBeforeItLooksAtMembers says the count is what
// NewMeshGroup refuses first: an empty list is turned away for its count,
// where a list of one member that is no mesh is turned away for the member.
func TestNewMeshGroupCountsBeforeItLooksAtMembers(t *testing.T) {
	if _, err := NewMeshGroup(nil); err == nil ||
		!strings.Contains(err.Error(), "member count") {
		t.Errorf("no members: error %v, want one about the count", err)
	}
	_, err := NewMeshGroup([]MeshMember{{Mesh: Sphere{Radius: unity}}})
	if err == nil || strings.Contains(err.Error(), "member count") {
		t.Errorf("one sphere: error %v, want one about the member", err)
	}
}
