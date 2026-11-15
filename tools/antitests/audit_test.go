package main

import (
	"bytes"
	"strings"
	"testing"
)

func audit(t *testing.T, dir string) report {
	t.Helper()
	rep, err := auditDir("testdata/" + dir)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestCleanPackageIsAnsweredOneToOne(t *testing.T) {
	rep := audit(t, "clean")
	if rep.tests != 3 || rep.antis != 1 || rep.positives != 2 ||
		rep.named != 2 || rep.ratio() != 1 {
		t.Errorf("clean counted %+v", rep)
	}
}

// anti-test of TestCleanPackageIsAnsweredOneToOne
func TestGapsAreCountedNotHidden(t *testing.T) {
	rep := audit(t, "gaps")
	// Answered and AlsoAnswered are named; Alone and LooseMarker are not
	if rep.positives != 4 || rep.named != 2 || rep.ratio() != 0.5 {
		t.Errorf("gaps counted %+v", rep)
	}
	if len(rep.unanswered) != 2 {
		t.Errorf("unanswered %v, want Alone and LooseMarker", rep.unanswered)
	}
}

func TestOneFunctionMayCarryTwoMarkers(t *testing.T) {
	rep := audit(t, "gaps")
	if rep.antis != 1 || rep.named < 2 {
		t.Errorf("two markers on one function counted %+v", rep)
	}
}

// anti-test of TestOneFunctionMayCarryTwoMarkers
func TestAMarkerWithWordsAfterItNamesNothing(t *testing.T) {
	rep := audit(t, "gaps")
	if rep.antis != 1 {
		t.Errorf("anti-tests %d, the loose marker must not count", rep.antis)
	}
	if len(rep.missing) != 0 {
		t.Errorf("a loose marker is not a missing name: %v", rep.missing)
	}
}

func TestAMarkerInsideATestNamesItsPositive(t *testing.T) {
	rep := audit(t, "clean")
	if len(rep.unanswered) != 0 {
		t.Errorf("unanswered %v", rep.unanswered)
	}
}

// anti-test of TestAMarkerInsideATestNamesItsPositive
func TestAnInlineMarkerDoesNotMakeItsTestAnAntiTest(t *testing.T) {
	rep := audit(t, "clean")
	if rep.antis != 1 {
		t.Errorf("anti-tests %d, want only the one above a function",
			rep.antis)
	}
}

func TestMissingNameIsAnErrorAndSeen(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"testdata/missing"}, &out, &errOut)
	if code != exitBelow || !strings.Contains(out.String(),
		"names TestNotThere") {
		t.Errorf("code %d, output %q", code, out.String())
	}
}

// anti-test of TestMissingNameIsAnErrorAndSeen
func TestNoMissingNameIsNoError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"testdata/clean"}, &out, &errOut); code != 0 {
		t.Errorf("code %d, output %q", code, out.String())
	}
}

func TestOldMarkersAreCountedWithTheirPlace(t *testing.T) {
	rep := audit(t, "old")
	if len(rep.old) != 2 || rep.old[0].line != 7 || rep.old[1].line != 10 {
		t.Errorf("old markers %v", rep.old)
	}
}

// anti-test of TestOldMarkersAreCountedWithTheirPlace
func TestOldMarkersNameNoTest(t *testing.T) {
	rep := audit(t, "old")
	if rep.antis != 0 || rep.named != 0 || len(rep.missing) != 0 {
		t.Errorf("old markers answered something: %+v", rep)
	}
}

func TestMinHoldsAPackageToARatio(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-min", "0.4", "testdata/gaps"}, &out,
		&errOut); code != 0 {
		t.Errorf("0.5 is not below 0.4: code %d", code)
	}
}

// anti-test of TestMinHoldsAPackageToARatio
func TestAPackageBelowMinFails(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-min", "0.6", "testdata/gaps"}, &out,
		&errOut); code != exitBelow {
		t.Errorf("0.5 is below 0.6: code %d", code)
	}
}

func TestVerboseListsUnansweredPositives(t *testing.T) {
	var out, errOut bytes.Buffer
	run([]string{"-v", "testdata/gaps"}, &out, &errOut)
	if !strings.Contains(out.String(),
		"unanswered: testdata/gaps/gaps_test.go:") {
		t.Errorf("output %q", out.String())
	}
}

// anti-test of TestVerboseListsUnansweredPositives
func TestQuietListsNoPositives(t *testing.T) {
	var out, errOut bytes.Buffer
	run([]string{"testdata/gaps"}, &out, &errOut)
	if strings.Contains(out.String(), "unanswered") {
		t.Errorf("output %q", out.String())
	}
}

func TestTotalLineSumsThePackages(t *testing.T) {
	var out, errOut bytes.Buffer
	run([]string{"testdata/clean", "testdata/gaps"}, &out, &errOut)
	if !strings.Contains(out.String(), "total: tests 8, anti-tests 2, "+
		"positives 6, named 4") {
		t.Errorf("output %q", out.String())
	}
}

// anti-test of TestTotalLineSumsThePackages
func TestNoPackagesIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(nil, &out, &errOut); code != exitUsage {
		t.Errorf("code %d", code)
	}
}

func TestEllipsisWalksBelowAndSkipsTestdata(t *testing.T) {
	dirs, err := expand([]string{"./..."})
	if err != nil || len(dirs) != 1 || dirs[0] != "." {
		t.Errorf("dirs %v, err %v", dirs, err)
	}
}

// anti-test of TestEllipsisWalksBelowAndSkipsTestdata
func TestNamedDirectoryIsTakenAsGiven(t *testing.T) {
	dirs, _ := expand([]string{"testdata/clean"})
	if len(dirs) != 1 || dirs[0] != "testdata/clean" {
		t.Errorf("dirs %v", dirs)
	}
}

func TestNameOfATest(t *testing.T) {
	if !isTestName("TestX") || !isTestName("Test_x") {
		t.Error("TestX and Test_x are tests")
	}
}

// anti-test of TestNameOfATest
func TestNameOfNearlyATest(t *testing.T) {
	if isTestName("Testx") || isTestName("Benchmark") || isTestName("test") {
		t.Error("Testx, Benchmark and test are not tests")
	}
}
