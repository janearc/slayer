package sources

import "io"

// progress is told how a download is going: the key of the source, the
// bytes received so far, and the bytes expected, which is the length
// the server announced, or zero or less when it announced none. It is
// called as bytes arrive, so it must be quick.
type progress func(key string, received, total int64)

// Counting is the shelf telling a function how each download goes. A
// nil function is none, which is how a shelf starts, so other callers
// see no change.
func (shelf Shelf) Counting(progress progress) Shelf {
	shelf.progress = progress
	return shelf
}

// counted is a reader that tells a function how many bytes have passed
// through it, and how many are expected.
type counted struct {
	from     io.Reader
	received int64
	total    int64
	report   func(received, total int64)
}

// Read reads from the wrapped reader and reports the bytes it gave.
func (reader *counted) Read(into []byte) (int, error) {
	got, err := reader.from.Read(into)
	if got > 0 {
		reader.received += int64(got)
		reader.report(reader.received, reader.total)
	}
	return got, err
}

// watched is a body read through a counter that reports to a function
// for a source's key. With no function it is the body itself.
func (shelf Shelf) watched(key string, body io.Reader,
	total int64) io.Reader {
	if shelf.progress == nil {
		return body
	}
	report := func(received, expected int64) {
		shelf.progress(key, received, expected)
	}
	report(0, total)
	return &counted{from: body, total: total, report: report}
}
