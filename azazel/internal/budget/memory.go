// Package budget decides if declared work fits host capacity.
// It does no I/O and grants no authority to start a process.
package budget

// byteCount is a count of bytes, independent of physical mass units.
type byteCount uint64

// noBytes is defined as the empty byte count; it grants no work.
const noBytes byteCount = 0

// memoryBudget is a configured ceiling and memory already promised to jobs.
type memoryBudget struct {
	limit    byteCount
	reserved byteCount
}

// memoryRequest is a job's explicit memory ceiling, which must be positive.
type memoryRequest struct {
	amount byteCount
}

// admits checks if the request fits after existing reservations.
// It bounds the request first so subtraction cannot wrap.
func (budget memoryBudget) admits(request memoryRequest) bool {
	if request.amount == noBytes || request.amount > budget.limit {
		return false
	}
	return budget.reserved <= budget.limit-request.amount
}
