package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// shoebox is the shoebox set file, as written.
func shoebox(test *testing.T) string {
	test.Helper()
	source, err := os.ReadFile("testdata/shoebox.yaml")
	if err != nil {
		test.Fatal(err)
	}
	return string(source)
}

func TestTheShoeboxReads(test *testing.T) {
	made, err := set.Read("shoebox.yaml", []byte(shoebox(test)))
	if err != nil {
		test.Fatal(err)
	}
	later := strings.Join(made.Unrendered(), "\n")
	for _, kept := range []string{"shoebox.yaml:29:9 autogen",
		"shoebox.yaml:30:9 autogen_parameters"} {
		if !strings.Contains(later, kept) {
			test.Errorf("%q is not among the unrendered:\n%s", kept, later)
		}
	}
}

func TestAMistakeIsRefusedWhereItIsWritten(test *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"  light: ambient", "  lihgt: ambient",
			"shoebox.yaml:3:3: lihgt is not a key here"},
		{"  light: ambient", "  light: spotlight",
			"shoebox.yaml:3:10: light takes [ambient moon sun]"},
		{"lat: 37.78507614135742", "lat: 91",
			"shoebox.yaml:5:10: lat takes a number from -90 to 90"},
		{"timeofday: 1548", "timeofday: 1575",
			"shoebox.yaml:7:14: timeofday takes a time as hhmm"},
		{"point_to: the box", "point_to: the shed",
			`camera shoot_1 wants "the shed"`},
		{"      house:", "      shed:",
			"shoebox.yaml:27:7: shed takes [house]"},
	} {
		written := strings.Replace(shoebox(test), mistake.was, mistake.now, 1)
		_, err := set.Read("shoebox.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			test.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

func TestAFileThatIsNotKeysIsRefused(test *testing.T) {
	if _, err := set.Read("list.yaml", []byte("- one\n- two\n")); err == nil {
		test.Error("a list was read as a set")
	}
}
