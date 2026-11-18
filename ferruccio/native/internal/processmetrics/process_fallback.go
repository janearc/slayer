//go:build !darwin || cgo || (!arm64 && !amd64)

package processmetrics

// Register does nothing where this adapter is unnecessary or unsupported.
func Register() func() { return func() {} }
