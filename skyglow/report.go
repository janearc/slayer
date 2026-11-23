package skyglow

import (
	"fmt"
	"strings"
)

// Summary is the report as one line a progress report can carry: what was
// solved, on what grid, how much of the scattered light each order held,
// and what the orders past the last are likely to hold.
func (report Report) Summary() string {
	var orders strings.Builder
	for index, share := range report.Shares {
		if index > 0 {
			orders.WriteString(" ")
		}
		fmt.Fprintf(&orders, "%.3f", share)
	}
	return fmt.Sprintf("skyglow: %d sources (%d scattered by the tracer), "+
		"%dx%dx%d nodes of %.1f by %.1f by %.1f m, shares by order %s, "+
		"tail %.3f of the whole, peak mean radiance %.3g %.3g %.3g",
		report.Sources, report.Traced, report.Nodes[0], report.Nodes[1],
		report.Nodes[2], report.Spacing[0], report.Spacing[1],
		report.Spacing[2], orders.String(), report.Tail, report.Peak.R,
		report.Peak.G, report.Peak.B)
}
