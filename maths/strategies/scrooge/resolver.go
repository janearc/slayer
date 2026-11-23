package scrooge

// Verdict is what the resolver says of a node at a footprint.
type Verdict int

const (
	// Average is to take the node's aggregate and ask no further.
	Average Verdict = iota
	// Open is to ask the node for its next layer.
	Open
)

// Resolve is whether a node is opened at a footprint: it is, where the
// footprint at its nearest point is narrower than the node is across and
// its next layer's parts are no finer than that width; otherwise it is
// taken as its aggregate. This is the one rule of how much detail a
// picture pays for.
func Resolve(node Node, at Footprint) Verdict {
	made := node.Aggregate()
	width := at.WidthAt(at.Nearest(made.Box))
	if made.Grain <= 0 || made.Size() <= width || made.Grain < width {
		return Average
	}
	return Open
}
