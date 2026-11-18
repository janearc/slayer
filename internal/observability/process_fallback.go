//go:build !darwin

package observability

// Other builds do not claim a process observation they cannot make.
func processUsage() (rss uint64, rssOK bool, cpuSeconds float64, cpuOK bool) {
	return 0, false, 0, false
}

// processFootprint reports no footprint, as these builds claim none.
func processFootprint() (current uint64, currentOK bool, peak uint64, peakOK bool) {
	return 0, false, 0, false
}
