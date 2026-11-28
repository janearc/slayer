package preview

import "testing"

func TestFullRenderRequestUsesQueueCeiling(t *testing.T) {
	plan, tools := queuedPlan(t)
	plan.Profile.Full = true
	request, err := plan.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	if request.TimeoutSeconds != renderTimeoutSeconds {
		t.Fatalf("full render timeout = %d; want %d", request.TimeoutSeconds, renderTimeoutSeconds)
	}
}

func TestPreviewRequestUsesQueueCeiling(t *testing.T) {
	plan, tools := queuedPlan(t)
	request, err := plan.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	if request.TimeoutSeconds != renderTimeoutSeconds {
		t.Fatalf("preview timeout = %d; want %d", request.TimeoutSeconds, renderTimeoutSeconds)
	}
}

func TestGPUPreviewAllowsHoursOfRendering(t *testing.T) {
	plan, tools := queuedPlan(t)
	plan.Profile.GPU = true
	plan.Profile.CPURenderer = false
	request, err := plan.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	if request.Lane != "gpu" || request.TimeoutSeconds != renderTimeoutSeconds {
		t.Fatalf("GPU city preview request = lane %q, timeout %d", request.Lane, request.TimeoutSeconds)
	}
}
