package terrain

// patchCandidate is one visible leaf whose refinement has not been considered.
type patchCandidate struct {
	patch   Patch
	score   float64
	visible bool
}

// patchHeap resolves equal priorities by source rectangle, keeping plans
// stable.
type patchHeap []patchCandidate

// Len counts the pending patch candidates.
func (front patchHeap) Len() int { return len(front) }

// Swap exchanges two pending candidates for heap maintenance.
func (front patchHeap) Swap(one, other int) { front[one], front[other] = front[other], front[one] }

// Less puts higher scores first, breaking ties by source rectangle.
func (front patchHeap) Less(one, other int) bool {
	if front[one].score != front[other].score {
		return front[one].score > front[other].score
	}
	return patchBefore(front[one].patch, front[other].patch)
}

// Push appends a candidate before the heap restores priority order.
func (front *patchHeap) Push(value any) { *front = append(*front, value.(patchCandidate)) }

// Pop removes the last candidate after the heap has moved it there.
func (front *patchHeap) Pop() any {
	last := len(*front) - 1
	value := (*front)[last]
	*front = (*front)[:last]
	return value
}

// patchBefore defines source-row/source-column order independently of a map.
func patchBefore(one, other Patch) bool {
	if one.MinRow != other.MinRow {
		return one.MinRow < other.MinRow
	}
	if one.MinColumn != other.MinColumn {
		return one.MinColumn < other.MinColumn
	}
	if one.MaxRow != other.MaxRow {
		return one.MaxRow < other.MaxRow
	}
	if one.MaxColumn != other.MaxColumn {
		return one.MaxColumn < other.MaxColumn
	}
	return one.Level < other.Level
}
