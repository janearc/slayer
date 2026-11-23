package show

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// fourFourLines is the sha256 of cueshow's lines for testdata's sheet,
// every twentieth of a second from nought to ten, at 24 frames a
// second. Measured at b9e5ba20, before the meter was the score's.
const fourFourLines = "9d7a52b4e4b4c0aeba4ad10700ece6ec" +
	"971f2ab2f329ca80b5a861f58220c9fd"

// fourFourStep is the moments' spacing, in hundredths of a second.
const fourFourStep = 5

// fourFourEnd is the last moment, in hundredths of a second.
const fourFourEnd = 1000

// hundredthsPerSecond turns hundredths into seconds.
const hundredthsPerSecond = 100.0

// fourFourRate is the frame rate the moments are drawn for.
const fourFourRate = 24

// a score in four, its meter unsaid, says the very lines it said before
// the meter was read from the score.
func TestFourFourLinesAreUnchanged(t *testing.T) {
	digest := sha256.New()
	sheet := testSheet(t)
	for hundredths := 0; hundredths <= fourFourEnd; hundredths +=
		fourFourStep {
		moment, err := sheet.At(float64(hundredths)/hundredthsPerSecond,
			fourFourRate)
		if err != nil {
			t.Fatal(err)
		}
		lineBytes := []byte(moment.Lines())
		digest.Write(lineBytes)
	}
	if got := hex.EncodeToString(digest.Sum(nil)); got != fourFourLines {
		t.Errorf("four-four lines hash %s, not %s", got, fourFourLines)
	}
}
