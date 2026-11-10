package publication

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

// TestLocalWriteKeepsBytesAndHandle needs a real file to establish exact
// native bytes, offset preservation and a still-open handle. No flush.
func TestLocalWriteKeepsBytesAndHandle(t *testing.T) {
	examples := payloadExamples(t)
	for _, sample := range examples.LocalWrite.cases() {
		t.Run(sample.name, sample.checkWrite)
	}
}

// checkWrite keeps byte, offset and still-open-handle assertions together.
func (sample nativePayloadCase) checkWrite(t *testing.T) {
	file := nativeTemporaryFile(t)
	prefix := []byte("before:")
	if _, err := file.Write(prefix); err != nil {
		t.Fatal(err)
	}
	written := (localTemporary{file: file}).Write(writeSpec{
		body: sample.body,
	})
	if written.err != nil || written.written != len(sample.body) {
		t.Fatalf("write: %+v, want %d bytes", written, len(sample.body))
	}
	want := append([]byte(nil), prefix...)
	want = append(want, sample.body...)
	assertNativeBytes(t, file.Name(), want)
	suffix := []byte(":after")
	if _, err := file.Write(suffix); err != nil {
		t.Fatalf("handle no longer writable: %v", err)
	}
	assertNativeBytes(t, file.Name(), append(want, suffix...))
}

// assertNativeBytes reads only the tiny fixture whose handle we own.
func assertNativeBytes(t *testing.T, name string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file bytes: %x, want %x", got, want)
	}
}

// TestLocalWriteRetainsInvalidHandleErrors is the anti-test: a refused
// native write cannot claim successful bytes or discard its error.
func TestLocalWriteRetainsInvalidHandleErrors(t *testing.T) {
	t.Run("already closed", checkLocalWriteAlreadyClosed)
	t.Run("absent", checkLocalWriteAbsent)
}

// checkLocalWriteAlreadyClosed keeps the native closed-handle refusal.
func checkLocalWriteAlreadyClosed(t *testing.T) {
	file := nativeTemporaryFile(t)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	written := (localTemporary{file: file}).Write(writeSpec{
		body: []byte("refuse"),
	})
	if written.written != noWrittenBytes ||
		!errors.Is(written.err, os.ErrClosed) {
		t.Fatalf("closed write: %+v, want zero and closed error", written)
	}
}

// checkLocalWriteAbsent keeps the missing-handle refusal and zero byte count.
func checkLocalWriteAbsent(t *testing.T) {
	written := (localTemporary{}).Write(writeSpec{body: []byte("refuse")})
	if written.written != noWrittenBytes ||
		!errors.Is(written.err, os.ErrInvalid) {
		t.Fatalf("absent write: %+v, want zero and invalid error", written)
	}
}
