package main

import (
	"testing"

	"go.uber.org/mock/gomock"
)

// TestProofStopLeavesAnUnexpiredPreviewAlone does not inspect or cancel normal work.
func TestProofStopLeavesAnUnexpiredPreviewAlone(t *testing.T) {
	preview := NewMockProofPreview(gomock.NewController(t))
	stop := &proofStop{}
	if expired, complete := stop.poll(preview); expired || complete {
		t.Fatalf("unexpired stop: %t %t", expired, complete)
	}
}

// TestProofStopRequestsOnceAndWaitsForOwnedCleanup preserves queue cancellation time.
func TestProofStopRequestsOnceAndWaitsForOwnedCleanup(t *testing.T) {
	preview := NewMockProofPreview(gomock.NewController(t))
	gomock.InOrder(
		preview.EXPECT().StopPreview(),
		preview.EXPECT().Running().Return(true),
		preview.EXPECT().Running().Return(true),
		preview.EXPECT().Running().Return(false),
	)
	stop := &proofStop{}
	stop.expire()
	for attempt := 0; attempt < 2; attempt++ {
		if expired, complete := stop.poll(preview); !expired || complete {
			t.Fatalf("cleanup still running: %t %t", expired, complete)
		}
	}
	if expired, complete := stop.poll(preview); !expired || !complete {
		t.Fatalf("cleanup completed: %t %t", expired, complete)
	}
	if expired, complete := stop.poll(preview); !expired || !complete {
		t.Fatalf("completion did not remain stable: %t %t", expired, complete)
	}
}

// TestProofStopCompletesImmediatelyWhenNoPreviewIsRunning avoids a false cleanup wait.
func TestProofStopCompletesImmediatelyWhenNoPreviewIsRunning(t *testing.T) {
	preview := NewMockProofPreview(gomock.NewController(t))
	gomock.InOrder(preview.EXPECT().StopPreview(), preview.EXPECT().Running().Return(false))
	stop := &proofStop{}
	stopped := make(chan struct{})
	go func() { stop.expire(); close(stopped) }()
	<-stopped
	if expired, complete := stop.poll(preview); !expired || !complete {
		t.Fatalf("idle cleanup: %t %t", expired, complete)
	}
}
