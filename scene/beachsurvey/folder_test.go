package beachsurvey

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// needsSurveys skips a test that reads the retained surveys where the
// data folder is not named.
func needsSurveys(t *testing.T) {
	t.Helper()
	if os.Getenv(dataRoot) == "" {
		t.Skipf("%s is not set, and the surveys are in it", dataRoot)
	}
}

// anyWater is an extinction for the tests that never reach the water.
var anyWater = maths.RGB(.31, .22, .17)

// with no data folder named, nothing is listed and a survey is refused
// in words that name the variable to set.
func TestSurveysNeedTheDataFolderNamed(t *testing.T) {
	t.Setenv(dataRoot, "")
	if sites := Sites(); len(sites) != 0 {
		t.Errorf("sites with no data folder: %v", sites)
	}
	_, _, _, err := NewWithWater("wategos", anyWater)
	if err == nil || !strings.Contains(err.Error(), dataRoot) {
		t.Errorf("a survey with no data folder: %v, want it to name %s",
			err, dataRoot)
	}
	if _, err := heights("wategos"); err == nil ||
		!strings.Contains(err.Error(), dataRoot) {
		t.Errorf("heights with no data folder: %v, want it to name %s",
			err, dataRoot)
	}
}

// a data folder that holds no survey lists none, and says a survey asked
// of it is unavailable.
func TestAnEmptyDataFolderHoldsNoSurvey(t *testing.T) {
	t.Setenv(dataRoot, t.TempDir())
	if sites := Sites(); len(sites) != 0 {
		t.Errorf("sites in an empty data folder: %v", sites)
	}
	_, _, _, err := NewWithWater("wategos", anyWater)
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Errorf("a survey from an empty folder: %v, want unavailable", err)
	}
}

// the sites are the surveys the folder holds, in name order, and a file
// that is no survey's page is not one of them.
func TestSitesAreWhatTheFolderHolds(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, surveyFolder)
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"b-cove.json", "a-bay.json", "a-bay.bin.gz"} {
		err := os.WriteFile(filepath.Join(folder, name), nil, 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(dataRoot, root)
	want := []string{"a-bay", "b-cove"}
	if sites := Sites(); !slices.Equal(sites, want) {
		t.Errorf("sites are %v, want %v", sites, want)
	}
}

// a survey's name cannot climb out of the folder.
func TestASurveyNameCannotLeaveTheFolder(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, surveyFolder)
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.json")
	if err := os.WriteFile(outside, []byte(`{"ID":"../outside"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(dataRoot, root)
	if _, _, _, err := NewWithWater("../outside", anyWater); err == nil {
		t.Error("a survey outside the folder was read")
	}
}
