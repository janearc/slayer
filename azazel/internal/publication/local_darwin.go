package publication

import "os"

// localTemporary owns one native file handle for the private publisher.
type localTemporary struct {
	file *os.File
}

// Close makes one native close attempt and retains its error.
func (temporary localTemporary) Close() error {
	return temporary.file.Close()
}

// Write calls the native file once and retains its count and error.
func (temporary localTemporary) Write(request writeSpec) writeResult {
	written, err := temporary.file.Write(request.body)
	return writeResult{written: written, err: err}
}

// A native temporary provides the full publisher handle boundary.
var _ temporary = localTemporary{}
