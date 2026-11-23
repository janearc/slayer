package session

import (
	"bytes"
	"testing"
)

// an ask says progress where it is told, keeps the rows it was given
// beside it, and the zero ask says none.
func TestAnAskSaysWhereItIsTold(t *testing.T) {
	var said bytes.Buffer
	ask := Ask{}.Keeping(nil).Saying(&said)
	if ask.said != &said {
		t.Errorf("the ask says on %v; want the writer it was given",
			ask.said)
	}
	if (Ask{}).said != nil {
		t.Error("the zero ask says progress somewhere")
	}
}
