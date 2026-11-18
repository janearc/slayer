package door

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

// censusRead is the parts of a census file the tests read.
type censusRead struct {
	Form    string `json:"form"`
	Counts  string `json:"counts"`
	Receipt struct {
		JobSHA256 string          `json:"job_sha256"`
		Scene     json.RawMessage `json:"scene"`
		SetSHA256 string          `json:"set_sha256"`
		Frame     int             `json:"frame"`
		Frames    int             `json:"frames"`
		Commit    string          `json:"commit"`
	} `json:"receipt"`
	Camera *struct {
		Lens string `json:"lens"`
	} `json:"camera"`
	Geometry struct {
		Total struct {
			Things int `json:"things"`
		} `json:"total"`
	} `json:"geometry"`
}

// the census of a still is written where --census says, with its
// receipt: the job's and the set's sha256, the scene and the camera,
// and its counts said to be staged, not visible.
func TestAssessWritesTheCensus(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	got := assessed(t, directory, path, "--census", "out/still.json")
	if got.code != nothing {
		t.Fatalf("assess --census gave %d: %s", got.code, got.wrongSaid)
	}
	census := filepath.Join(directory, "out", "still.json")
	if !strings.Contains(got.said, "wrote the census "+census) {
		t.Errorf("assess said %q", got.said)
	}
	read := readCensus(t, census)
	jobBytes, setBytes := fileBytes(t, path), fileBytes(t, widow(t))
	if read.Form != censusForm || read.Counts != "staged, not visible" ||
		read.Receipt.JobSHA256 != digest(jobBytes) ||
		read.Receipt.SetSHA256 != digest(setBytes) ||
		read.Receipt.Frame != nothing || read.Receipt.Frames != one ||
		!strings.Contains(string(read.Receipt.Scene), "widow.yaml") {
		t.Errorf("the receipt: %+v", read)
	}
	if read.Camera == nil || read.Geometry.Total.Things == nothing {
		t.Errorf("the camera %+v and things %d", read.Camera,
			read.Geometry.Total.Things)
	}
}

// the anti-test: assess without --census writes nothing and says what
// it said before; with --judge, --census is refused by name.
func TestAssessWithoutTheCensusWritesNothing(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	got := assessed(t, directory, path)
	if got.code != nothing || strings.Contains(got.said, "census") {
		t.Errorf("assess gave %d: %q", got.code, got.said)
	}
	if listed := files(t, directory); len(listed) != one {
		t.Errorf("assess wrote %v", listed)
	}
	judged := assessed(t, directory, widow(t), "--judge", "--census",
		"c.json")
	if judged.code != one || !strings.Contains(judged.wrongSaid,
		"--census does nothing with judge") {
		t.Errorf("--judge --census gave %d: %q", judged.code,
			judged.wrongSaid)
	}
}

// the anti-test: a scene with no staged geometry, frames already
// rendered, is refused by name, writes no census and is let go.
func TestACensusOfPicturesIsRefused(t *testing.T) {
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	opened := NewMocksource(control)
	shop.EXPECT().Source(gomock.Any(), gomock.Any()).Return(opened, nil)
	opened.EXPECT().Tone().AnyTimes()
	opened.EXPECT().Close()
	shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(nil, nil)
	shop.EXPECT().Titler(gomock.Any()).Return(nil, false, nil)
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "assess", widow(t),
		"--census", "c.json")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"frames already rendered") {
		t.Errorf("an uncounted source gave %d: %q", got.code,
			got.wrongSaid)
	}
	if listed := files(t, directory); len(listed) != nothing {
		t.Errorf("a refused census wrote %v", listed)
	}
}

// readCensus is the census file at path, read.
func readCensus(t *testing.T, path string) censusRead {
	t.Helper()
	var read censusRead
	if err := json.Unmarshal(fileBytes(t, path), &read); err != nil {
		t.Fatal(err)
	}
	return read
}

// fileBytes is the bytes of the file at path.
func fileBytes(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
