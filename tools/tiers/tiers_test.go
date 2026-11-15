package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseListTakesNamesBeforeTheirOkLine(t *testing.T) {
	got, err := parseList([]byte("TestA\nTestB\nok  \tp/one\t0.01s\n?   \tp/two\t[no test files]\nTestC\nok  \tp/three\t0.02s\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got["p/one"]) != 2 || len(got["p/three"]) != 1 || len(got["p/two"]) != 0 {
		t.Errorf("parsed %v", got)
	}
}

// anti-test of TestParseListTakesNamesBeforeTheirOkLine
func TestParseListRefusesAFailedPackage(t *testing.T) {
	if _, err := parseList([]byte("FAIL\tp/one [build failed]\n")); err == nil {
		t.Error("a failed package was read as an empty one")
	}
}

func TestCompareIsQuietWhenTheTiersAddUp(t *testing.T) {
	saved := names{"p": {"TestA", "TestB"}}
	all := names{"p": {"TestB", "TestA"}}
	plain := names{"p": {"TestA"}}
	if got := compare(saved, all, plain); len(got) != 0 {
		t.Errorf("problems %v", got)
	}
}

// anti-test of TestCompareIsQuietWhenTheTiersAddUp
func TestCompareNamesALostAnAddedAndAMisplacedTest(t *testing.T) {
	saved := names{"p": {"TestA", "TestB"}}
	all := names{"p": {"TestA", "TestNew"}}
	plain := names{"p": {"TestA", "TestStray"}}
	got := strings.Join(compare(saved, all, plain), "\n")
	for _, want := range []string{"lost: TestB", "added: TestNew", "plain: TestStray"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in %q", want, got)
		}
	}
}

// anti-test of TestCompareIsQuietWhenTheTiersAddUp
func TestCompareNamesADuplicate(t *testing.T) {
	saved := names{"p": {"TestA"}}
	all := names{"p": {"TestA", "TestA"}}
	got := strings.Join(compare(saved, all, names{}), "\n")
	if !strings.Contains(got, "duplicated") {
		t.Errorf("no duplicate in %q", got)
	}
}

func TestFormatAndReadSavedAgree(t *testing.T) {
	n := names{"p/b": {"TestY", "TestX"}, "p/a": {"TestZ"}}
	dir := t.TempDir()
	file := dir + "/names.txt"
	if err := writeFile(file, format(n)); err != nil {
		t.Fatal(err)
	}
	got, err := readSaved(file)
	if err != nil {
		t.Fatal(err)
	}
	if count(got) != 3 || len(compare(n, got, got)) != 0 {
		t.Errorf("read back %v", got)
	}
}

// anti-test of TestFormatAndReadSavedAgree
func TestReadSavedRefusesALineWithNoTab(t *testing.T) {
	file := t.TempDir() + "/names.txt"
	if err := writeFile(file, "p TestA\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := readSaved(file); err == nil {
		t.Error("a line with no tab was read")
	}
}

func writeFile(file, text string) error { return os.WriteFile(file, []byte(text), 0o644) }

func TestReadHotTakesTheWordBeforeTheComment(t *testing.T) {
	file := t.TempDir() + "/hot.txt"
	if err := writeFile(file, "# why\n\nexample.com/a   # one\nexample.com/b\n"); err != nil {
		t.Fatal(err)
	}
	got, err := readHot(file)
	if err != nil || len(got) != 2 || got[0] != "example.com/a" || got[1] != "example.com/b" {
		t.Errorf("read %v, %v", got, err)
	}
}

// anti-test of TestReadHotTakesTheWordBeforeTheComment
func TestReadHotRefusesTwoWordsAndAnEmptyList(t *testing.T) {
	file := t.TempDir() + "/hot.txt"
	if err := writeFile(file, "example.com/a example.com/b\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := readHot(file); err == nil {
		t.Error("two words on a line were read")
	}
	if err := writeFile(file, "# nothing\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := readHot(file); err == nil {
		t.Error("an empty list was read")
	}
}

func TestDrawArgsRunOnlyTheNamedTestsWithTheTag(t *testing.T) {
	got := strings.Join(drawArgs("p/one", []string{"TestA", "Example_b"}), " ")
	want := "test -count=1 -tags draw -p 1 -timeout 60m -run ^(TestA|Example_b)$ p/one"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// anti-test of TestDrawArgsRunOnlyTheNamedTestsWithTheTag
func TestDrawArgsRunNothingForAPackageWithNoDrawTests(t *testing.T) {
	if got := drawArgs("p/one", nil); got != nil {
		t.Errorf("a package with no draw tests would run: %q", got)
	}
}
