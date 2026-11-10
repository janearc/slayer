//ninja:vow dojo unknown private choice names must be refused
package manufacture

import (
	"fmt"
	"strings"
	"testing"
)

func TestGroundObservatoryUnknownChoiceKeepsItsName(test *testing.T) {
	name := "absent-ground-choice"
	defer assertGroundRefusal(test, name)
	groundValue(name)
}

func TestGroundObservatoryChoicesRefuseMalformedData(test *testing.T) {
	defer assertGroundRefusal(test, "ground-choices.yaml")
	readGroundChoices([]byte("numbers: ["))
}

func assertGroundRefusal(test *testing.T, context string) {
	failure := recover()
	test.Helper()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), context) {
		test.Fatal("choice refusal lost its context", context, failure)
	}
}
