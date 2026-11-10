package gpu

import "github.com/janearc/slayer/maths"

// rowPlan resolves caller field order against the shared ABI declaration once.
// Hot loops can pass stack values without constructing/checking a map per row.
type rowPlan struct {
	structure structure
	sources   []int
}

// planRows refuses missing, duplicate or unknown fields as a code fault.
// Reordering the ABI declaration still writes each supplied field by its name.
func planRows(made structure, names ...string) rowPlan {
	if len(names) != len(made.fields) {
		made.wrong("row plan has %d fields, needs %d", len(names), len(made.fields))
	}
	plan := rowPlan{structure: made, sources: make([]int, len(made.fields))}
	used := make([]bool, len(names))
	for index, member := range made.fields {
		found := -1
		for source, name := range names {
			if member.name == name {
				if found >= 0 || used[source] {
					made.wrong("duplicate row-plan field %s", name)
				}
				found, used[source] = source, true
			}
		}
		if found < 0 {
			made.wrong("no row-plan value for %s", member.name)
		}
		plan.sources[index] = found
	}
	return plan
}

// write retains declaration order, scalar representation and zero padding.
func (plan rowPlan) write(buffer *packer, values ...[4]float64) {
	if len(values) != len(plan.sources) {
		plan.structure.wrong("row has %d values, needs %d", len(values), len(plan.sources))
	}
	for index, member := range plan.structure.fields {
		for _, value := range values[plan.sources[index]] {
			buffer.word(member.bits(value))
		}
	}
}

// vectorRow is one explicit float4 value, with no borrowed slice or allocation.
func vectorRow(point maths.Vec, fourth float64) [4]float64 {
	return [4]float64{point.X, point.Y, point.Z, fourth}
}
