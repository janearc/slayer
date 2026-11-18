package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/render/workload"
)

func TestReportLoadsExactStoredProfileWithExplicitHostInputs(t *testing.T) {
	cache, identityPath := preflightFixture(t)
	store, err := workload.Open(cache, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	physical, peak := uint64(100), uint64(120)
	sample := preflightObservation()
	sample.Physical, sample.PhysicalPeak = &physical, &peak
	identity := preflightIdentity()
	if err := store.Record(identity, []workload.Observation{sample}); err != nil {
		t.Fatal(err)
	}
	args := preflightArgs(cache, identityPath)
	var out, errs bytes.Buffer
	if code := run(args, &out, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	var got workload.EstimateReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != workload.EstimateWithinCapacity || got.RequiredBytes == nil || *got.RequiredBytes != 790 ||
		got.PerProcessBytes == nil || *got.PerProcessBytes != peak || got.Identity != identity ||
		got.AdditionalProcesses != 2 || got.FramesPerProcess != 8 {
		t.Fatalf("report = %+v; stderr %s", got, errs.String())
	}
}

func TestReportMissAndInvalidIdentityStayUnknown(t *testing.T) {
	cache, identityPath := preflightFixture(t)
	var out, errs bytes.Buffer
	if code := run(preflightArgs(cache, identityPath), &out, &errs); code != 0 {
		t.Fatalf("cache miss exit %d: %s", code, errs.String())
	}
	var missing workload.EstimateReport
	if err := json.Unmarshal(out.Bytes(), &missing); err != nil {
		t.Fatal(err)
	}
	if missing.Status != workload.EstimateUnknown || missing.RequiredBytes != nil || !hasReason(missing.Unknown, "no observations") {
		t.Fatalf("cache miss produced a number: %+v", missing)
	}

	if err := os.WriteFile(identityPath, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run(preflightArgs(cache, identityPath), &out, &errs); code != 0 {
		t.Fatalf("invalid identity exit %d: %s", code, errs.String())
	}
	var invalid struct {
		Status  workload.EstimateStatus `json:"status"`
		Unknown []string                `json:"unknown"`
	}
	if err := json.Unmarshal(out.Bytes(), &invalid); err != nil {
		t.Fatal(err)
	}
	if invalid.Status != workload.EstimateUnknown || len(invalid.Unknown) == 0 {
		t.Fatalf("invalid identity metadata was hidden: %+v", invalid)
	}
}

func TestReportCorruptProfileStaysUnknown(t *testing.T) {
	cache, identityPath := preflightFixture(t)
	if err := os.WriteFile(filepath.Join(cache, "workload-profiles.bolt"), []byte("not a bolt database"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if code := run(preflightArgs(cache, identityPath), &out, &errs); code != 0 {
		t.Fatalf("corrupt profile exit %d: %s", code, errs.String())
	}
	var got struct {
		Status  workload.EstimateStatus `json:"status"`
		Unknown []string                `json:"unknown"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != workload.EstimateUnknown || len(got.Unknown) == 0 {
		t.Fatalf("corrupt profile metadata was hidden: %+v", got)
	}
}

func TestReportRejectsBadFlagsAndReadOnlyMissCreatesNoCache(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "not-created")
	identityPath := filepath.Join(t.TempDir(), "identity.json")
	if err := os.WriteFile(identityPath, identityBytes(t), 0600); err != nil {
		t.Fatal(err)
	}
	args := preflightArgs(cache, identityPath)
	args[len(args)-1] = "not-an-integer"
	var out, errs bytes.Buffer
	if code := run(args, &out, &errs); code == 0 || !strings.Contains(errs.String(), "unsigned integer") {
		t.Fatalf("bad numeric flag exit %d, stdout %q, stderr %q", code, out.String(), errs.String())
	}

	args = preflightArgs(cache, identityPath)
	out.Reset()
	errs.Reset()
	if code := run(args, &out, &errs); code != 0 {
		t.Fatalf("missing cache exit %d: %s", code, errs.String())
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("read-only preflight miss created cache directory; stat: %v", err)
	}
}

func preflightArgs(cache, identityFile string) []string {
	return []string{"report", "--cache-home", cache, "--identity-file", identityFile,
		"--capacity-bytes", "1000", "--current-use-bytes", "500",
		"--processes", "2", "--frames-per-process", "8", "--headroom-bytes", "50"}
}

func preflightFixture(t *testing.T) (string, string) {
	t.Helper()
	cache, identityPath := t.TempDir(), filepath.Join(t.TempDir(), "identity.json")
	if err := os.WriteFile(identityPath, identityBytes(t), 0600); err != nil {
		t.Fatal(err)
	}
	return cache, identityPath
}

func identityBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(preflightIdentity())
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func preflightIdentity() workload.Identity {
	return workload.Identity{Source: strings.Repeat("a", 64), Build: strings.Repeat("b", 64),
		Device: strings.Repeat("c", 64), Render: strings.Repeat("d", 64), Probe: strings.Repeat("e", 64)}
}

func preflightObservation() workload.Observation {
	return workload.Observation{Run: "preflight-run", Sample: "cold", Frame: 0,
		At: time.Unix(1_700_000_000, 0), Cache: "cold", Wall: time.Second}
}

func hasReason(reasons []string, fragment string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, fragment) {
			return true
		}
	}
	return false
}
