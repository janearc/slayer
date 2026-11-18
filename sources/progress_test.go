package sources

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

// heard is one call to a progress function.
type heard struct {
	key             string
	received, total int64
}

// A counting reader reports every byte that passes, in order, with the
// total it was given, and does not change the bytes.
func TestACountingReaderReportsTheBytesThatPass(t *testing.T) {
	var calls []heard
	reader := &counted{from: iotest.OneByteReader(strings.NewReader("abcd")),
		total: 4, report: func(received, total int64) {
			calls = append(calls, heard{received: received, total: total})
		}}
	got, err := io.ReadAll(reader)
	if err != nil || string(got) != "abcd" {
		t.Fatalf("read %q, %v", got, err)
	}
	if len(calls) != 4 || calls[3] != (heard{received: 4, total: 4}) {
		t.Errorf("heard %v", calls)
	}
}

// A shelf told to count hears its source's key, a start at zero, and the
// whole size at the end, with the length the server announced.
func TestAShelfThatCountsHearsEachDownload(t *testing.T) {
	server, _ := served(t)
	var calls []heard
	shelf := NewShelf(t.TempDir(), server.Client()).Counting(
		func(key string, received, total int64) {
			calls = append(calls, heard{key, received, total})
		})
	_, err := shelf.Fetch(context.Background(), one(t, server, ""),
		time.Now())
	size := int64(len(paper))
	if err != nil || len(calls) < 2 || calls[0] != (heard{"paper", 0, size}) ||
		calls[len(calls)-1] != (heard{"paper", size, size}) {
		t.Errorf("heard %v, %v", calls, err)
	}
}

// With no length announced the total is not a positive number, so the
// display knows to say only the bytes received.
func TestAnUnannouncedLengthIsNotPositive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		answer http.ResponseWriter, request *http.Request) {
		answer.(http.Flusher).Flush()
		answer.Write([]byte(paper))
	}))
	defer server.Close()
	var last heard
	shelf := NewShelf(t.TempDir(), server.Client()).Counting(
		func(key string, received, total int64) {
			last = heard{key, received, total}
		})
	if _, err := shelf.Fetch(context.Background(), one(t, server, ""),
		time.Now()); err != nil {
		t.Fatal(err)
	}
	if last.total > 0 || last.received != int64(len(paper)) {
		t.Errorf("heard %v", last)
	}
}

// Anti-test: a shelf that is not told to count is the shelf it was, and
// fetches as before.
func TestAShelfThatDoesNotCountFetchesAsBefore(t *testing.T) {
	server, _ := served(t)
	shelf := NewShelf(t.TempDir(), server.Client())
	if shelf.progress != nil {
		t.Error("a new shelf counts")
	}
	said, err := shelf.Fetch(context.Background(), one(t, server, ""),
		time.Now())
	if err != nil || !strings.Contains(said, "fetched, not pinned") {
		t.Errorf("said %q, %v", said, err)
	}
}
