package hillside

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// recorded reads the measurement behind bytesPerKeptFace from its file.
func recorded(t *testing.T) map[string]string {
	t.Helper()
	file, err := os.Open("testdata/staging-memory.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	found := map[string]string{}
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		key, value, is := strings.Cut(lines.Text(), ": ")
		if is && !strings.HasPrefix(key, "#") {
			found[key] = value
		}
	}
	return found
}

// measuredFace is bytesPerKeptFace written out by hand, so a test's
// expected bytes do not come from the constant under test.
const measuredFace = 1_300

// roundTo is the step, in bytes, that bytesPerKeptFace is rounded up by.
const roundTo = 100

// the constant is the recorded measurement rounded up to a hundred bytes,
// so changing the constant without changing the record, or the record
// without the constant, fails here.
func TestBytesPerKeptFaceIsTheRecordedMeasurement(t *testing.T) {
	record := recorded(t)
	kept, err := strconv.Atoi(record["kept_faces"])
	if err != nil || kept < 1 {
		t.Fatalf("kept_faces in the record: %q: %v", record["kept_faces"], err)
	}
	peak, err := strconv.Atoi(record["peak_bytes"])
	if err != nil || peak < 1 {
		t.Fatalf("peak_bytes in the record: %q: %v", record["peak_bytes"], err)
	}
	measured := peak / kept
	if bytesPerKeptFace < measured || bytesPerKeptFace-measured >= roundTo {
		t.Errorf("bytesPerKeptFace is %d, the record %s says %d bytes a "+
			"face, to round up by under %d", bytesPerKeptFace,
			record["set"], measured, roundTo)
	}
}

// the default of -faces is the memory budget at the measured bytes a
// face, and the flag's help says so in bytes.
func TestTheDefaultFacesAreTheBudgetInBytes(t *testing.T) {
	const budget, perFace, faces = 5_000_000_000, measuredFace, 3_846_153
	if memoryBudget != budget || bytesPerKeptFace != perFace ||
		mostFaces != faces {
		t.Errorf("budget %d, %d a face, %d faces; want %d, %d, %d",
			memoryBudget, bytesPerKeptFace, mostFaces, budget, perFace,
			faces)
	}
	var said bytes.Buffer
	given, _ := parse([]string{"-help"}, &said)
	if given.faces != 0 && given.faces != faces {
		t.Errorf("-faces defaults to %d, not %d", given.faces, faces)
	}
	if !strings.Contains(said.String(), "5000000000 bytes") {
		t.Errorf("the help does not name the budget in bytes: %s",
			said.String())
	}
}

// a scene far past the budget is refused, and the refusal names what the
// scene would keep in faces and bytes, the budget in faces and bytes, and
// the flag that raises it.
func TestASceneFarPastTheBudgetIsRefusedByName(t *testing.T) {
	const perFace, allowed = measuredFace, 3_846_153
	given := mustParse(t)
	world, made := thing.NewWorld(), tally{kept: allowed}
	one, _, err := stock{}.made(given, "can", 0)
	if err != nil {
		t.Fatal(err)
	}
	keeping := allowed + one.stored
	err = stand(world, &made, given, stock{}, "can", 0, maths.Subspace{})
	want := []string{
		fmt.Sprintf("would keep %d faces, about %d bytes", keeping,
			keeping*perFace),
		fmt.Sprintf("budget of %d faces, about %d bytes", allowed,
			allowed*perFace),
		"-faces",
	}
	if !errors.Is(err, errTooMany) {
		t.Fatalf("a scene past the budget: %v, not %v", err, errTooMany)
	}
	for _, each := range want {
		if !strings.Contains(err.Error(), each) {
			t.Errorf("the refusal %q lacks %q", err.Error(), each)
		}
	}
}
