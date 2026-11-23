package preview

import "testing"

// TestFullRenderRequestHasNoFiveMinuteLimit checks that a full render is sent
// with the longest limit the request schema accepts.
func TestFullRenderRequestHasNoFiveMinuteLimit(t *testing.T) {
	plan, tools := queuedPlan(t)
	plan.Profile.Full = true
	request, err := plan.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	if request.TimeoutSeconds != fullTimeoutSeconds || request.TimeoutSeconds <= previewTimeoutSeconds {
		t.Fatalf("full render timeout = %d; want %d", request.TimeoutSeconds, fullTimeoutSeconds)
	}
}

// TestPreviewRequestKeepsItsLimit is the anti-test: a preview keeps the short
// limit.
func TestPreviewRequestKeepsItsLimit(t *testing.T) {
	plan, tools := queuedPlan(t)
	request, err := plan.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	if request.TimeoutSeconds != previewTimeoutSeconds {
		t.Fatalf("preview timeout = %d; want %d", request.TimeoutSeconds, previewTimeoutSeconds)
	}
}
