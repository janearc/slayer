package searching

// FlowAt keeps the final reveal through the coda and soundtrack release.
// Explicit shots take precedence; the old ground coda is not an implicit ending
// for a full cut that has already reached its authored flow scene.
func (edit *Edit) FlowAt(at float64) bool {
	return edit.Plan.FlowStudy != nil && (!edit.Plan.FullCut || at >= edit.FlowStart)
}
