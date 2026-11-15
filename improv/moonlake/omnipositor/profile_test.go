package omnipositor

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestProfileWritesOrderedTrace(t *testing.T) {
	profile, err := NewProfile(2)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Unix(10, 0)
	if err := profile.Record(ProfileSpan{Frame: 2, Stage: "mist", Start: start.Add(time.Second), Duration: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if err := profile.Record(ProfileSpan{Frame: 2, Stage: "matte", Start: start, Duration: time.Microsecond}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := profile.WriteTrace(&output); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Events []traceEvent `json:"traceEvents"`
	}
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Events) != 2 || decoded.Events[0].Name != "matte" || decoded.Events[1].Name != "mist" {
		t.Fatalf("trace order: %+v", decoded.Events)
	}
}

func TestProfileRefusesInvalidAndExcessSpans(t *testing.T) {
	profile, err := NewProfile(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := profile.Record(ProfileSpan{Frame: -1, Stage: "draw", Start: time.Now()}); err == nil {
		t.Fatal("negative frame accepted")
	}
	span := ProfileSpan{Frame: 0, Stage: "draw", Start: time.Now()}
	if err := profile.Record(span); err != nil {
		t.Fatal(err)
	}
	if err := profile.Record(span); err == nil {
		t.Fatal("span limit ignored")
	}
}
