package observability

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

func TestMemoryStageWritesAnObservation(t *testing.T) {
	var out bytes.Buffer
	if err := WriteMemoryStage(&out, "before-pool"); err != nil {
		t.Fatal(err)
	}
	var got MemoryStage
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Stage != "before-pool" || got.PID <= 0 || got.ObservedAt.IsZero() {
		t.Fatalf("stage identity was not recorded: %+v", got)
	}
	if got.HeapAlloc == 0 || got.HeapSys < got.HeapAlloc || got.Sys < got.HeapSys {
		t.Fatalf("impossible Go memory observation: %+v", got)
	}
	if got.FootprintAvailable && got.Footprint == 0 {
		t.Fatalf("available footprint was empty: %+v", got)
	}
	if got.PeakAvailable && (!got.FootprintAvailable || got.Peak < got.Footprint) {
		t.Fatalf("invalid lifetime peak: %+v", got)
	}
}

func TestMemoryStageReportsWriteFailure(t *testing.T) {
	writer := mocks.NewMockWriter(gomock.NewController(t))
	writer.EXPECT().Write(gomock.Any()).Return(0, errors.New("no space")).
		AnyTimes()
	if err := WriteMemoryStage(writer, "after-pool"); err == nil {
		t.Fatal("failed writer was accepted")
	}
}
