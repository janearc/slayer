package diskcache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sumOf is the sha256 of text, in hex.
func sumOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func TestKeyRecordSaysEveryPartInOrder(t *testing.T) {
	kind := testKind(t, "key-record")
	key := NewKey(kind).Text("width", "640").Bytes("set", []byte("a")).
		Digest("dem", sumOf("dem"))
	want := "kind key-record\nwidth 640\nset " + sumOf("a") + "\ndem " +
		sumOf("dem") + "\n"
	if key.Err() != nil || key.record() != want {
		t.Fatalf("record %q, %v; want %q", key.record(), key.Err(), want)
	}
	if key.Name() != sumOf(want) || key.Kind() != kind {
		t.Fatalf("name %s is not the record's sha256", key.Name())
	}
}

func TestKeyNameChangesWithOneByte(t *testing.T) {
	kind := testKind(t, "key-byte")
	one := NewKey(kind).Bytes("set", []byte("abc"))
	other := NewKey(kind).Bytes("set", []byte("abd"))
	if one.Name() == other.Name() {
		t.Fatal("one byte changed and the key did not")
	}
}

func TestKeyIsNotChangedInPlace(t *testing.T) {
	// Three parts, so the base's list has room to spare: a part said by
	// reaching into that room would show in both keys made from the base.
	base := NewKey(testKind(t, "key-place")).Text("a", "1").Text("b", "2").
		Text("c", "3")
	one, other := base.Text("d", "4"), base.Text("d", "5")
	if base.record() != "kind key-place\na 1\nb 2\nc 3\n" ||
		one.Err() != nil || other.Err() != nil {
		t.Fatalf("base %q, one %q, other %q", base.record(), one.record(),
			other.record())
	}
	// anti-test of TestKeyIsNotChangedInPlace
	// Two keys made from one base differ; if they shared the base's list
	// the second would have overwritten the first and the two would be one.
	if one.Name() == other.Name() || one.record() == other.record() {
		t.Fatal("two keys made from one base are one key")
	}
}

// anti-test of TestKeyRecordSaysEveryPartInOrder
func TestKeyRefusesAPartSaidTwice(t *testing.T) {
	key := NewKey(testKind(t, "key-twice")).Text("a", "1").Text("a", "1")
	if key.Err() == nil || !strings.Contains(key.Err().Error(), "twice") {
		t.Fatalf("a part said twice was taken: %v", key.Err())
	}
}

func TestKeyKeepsTheFirstRefusal(t *testing.T) {
	key := NewKey(testKind(t, "key-first")).Text("", "1").Text("a", "\n")
	if key.Err() == nil || !strings.Contains(key.Err().Error(), "word") {
		t.Fatalf("the first refusal was not kept: %v", key.Err())
	}
}

// anti-test of TestKeyRecordSaysEveryPartInOrder
func TestKeyRefusesALineBreakInText(t *testing.T) {
	key := NewKey(testKind(t, "key-break")).Text("a", "one\ntwo")
	if key.Err() == nil {
		t.Fatal("a value of two lines was taken")
	}
}

// anti-test of TestKeyRecordSaysEveryPartInOrder
func TestKeyRefusesANameOfTwoWords(t *testing.T) {
	key := NewKey(testKind(t, "key-words")).Text("two words", "1")
	if key.Err() == nil {
		t.Fatal("a name of two words was taken")
	}
}

// anti-test of TestKeyRecordSaysEveryPartInOrder
func TestKeyRefusesADigestThatIsNotOne(t *testing.T) {
	for _, sum := range []string{"abc", strings.Repeat("g", digestLength),
		sumOf("x") + "0"} {
		key := NewKey(testKind(t, "key-digest")).Digest("dem", sum)
		if key.Err() == nil {
			t.Errorf("digest %q was taken", sum)
		}
	}
}

func TestFileDigestIsTheBytesSha256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("bytes"), fileMode); err != nil {
		t.Fatal(err)
	}
	sum, err := FileDigest(path)
	if err != nil || sum != sumOf("bytes") {
		t.Fatalf("digest %s, %v", sum, err)
	}
}

// anti-test of TestFileDigestIsTheBytesSha256
func TestFileDigestRefusesAFileThatIsNotThere(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	if _, err := FileDigest(gone); err == nil {
		t.Fatal("a missing file had a digest")
	}
}

// anti-test of TestFileDigestIsTheBytesSha256
func TestFileDigestRefusesAFolder(t *testing.T) {
	if _, err := FileDigest(t.TempDir()); err == nil {
		t.Fatal("a folder had a digest")
	}
}

func TestBuildIsTheRunningProgramsSha256(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want, err := FileDigest(path)
	got, built := Build()
	if err != nil || built != nil || got != want {
		t.Fatalf("build %s, %v; want %s, %v", got, built, want, err)
	}
}

func TestBuiltIsTheBuildAsAPart(t *testing.T) {
	program, err := Build()
	key := NewKey(testKind(t, "key-built")).Built()
	if err != nil || key.Err() != nil ||
		key.record() != "kind key-built\nbuild "+program+"\n" {
		t.Fatalf("record %q, %v, %v", key.record(), key.Err(), err)
	}
}

// anti-test of TestBuiltIsTheBuildAsAPart
func TestBuiltIsSaidOnlyOnce(t *testing.T) {
	key := NewKey(testKind(t, "key-built-twice")).Built().Built()
	if key.Err() == nil {
		t.Fatal("the build was said twice")
	}
}

// anti-test of TestKeyNameChangesWithOneByte
func TestKeyNameIsTheSameForTheSameParts(t *testing.T) {
	kind := testKind(t, "key-same")
	one := NewKey(kind).Text("width", "640").Bytes("set", []byte("abc"))
	other := NewKey(kind).Text("width", "640").Bytes("set", []byte("abc"))
	if one.Name() != other.Name() || one.record() != other.record() {
		t.Fatalf("the same parts gave %s and %s", one.Name(), other.Name())
	}
}

// anti-test of TestKeyRecordSaysEveryPartInOrder
func TestKeyRecordKeepsTheOrderSaid(t *testing.T) {
	kind := testKind(t, "key-order")
	one := NewKey(kind).Text("a", "1").Text("b", "2")
	other := NewKey(kind).Text("b", "2").Text("a", "1")
	if one.record() == other.record() || one.Name() == other.Name() {
		t.Fatal("two orders of the same parts gave one key")
	}
}

// anti-test of TestKeyKeepsTheFirstRefusal
func TestKeyKeepsTheFirstRefusalWhicheverItIs(t *testing.T) {
	key := NewKey(testKind(t, "key-first-other")).Text("a", "\n").Text("", "1")
	if key.Err() == nil || !strings.Contains(key.Err().Error(), "break") {
		t.Fatalf("the first refusal, a line break, was not kept: %v",
			key.Err())
	}
}

// anti-test of TestBuildIsTheRunningProgramsSha256
// anti-test of TestBuiltIsTheBuildAsAPart
func TestBuildSaysWhenTheProgramCannotBeFound(t *testing.T) {
	was := build
	defer func() { build = was }()
	build = func() (string, error) { return "", errors.New("not found") }
	key := NewKey(testKind(t, "key-unbuilt")).Built()
	if _, err := Build(); err == nil {
		t.Fatal("a build that cannot be found had a digest")
	}
	if key.Err() == nil {
		t.Fatal("a key without its build was not refused")
	}
}
