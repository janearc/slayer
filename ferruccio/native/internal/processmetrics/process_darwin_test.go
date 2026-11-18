//go:build darwin && !cgo && (arm64 || amd64)

package processmetrics

import (
	"fmt"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/janearc/slayer/internal/observability"
)

func TestRusageLayoutMatchesAppleSDK(t *testing.T) {
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("Apple SDK compiler unavailable")
	}
	program := `#include <stddef.h>
#include <stdio.h>
#include <sys/resource.h>
#include <libproc.h>
#include <unistd.h>
int main(void) {
  printf("%zu %zu %zu %zu %d %d\n", sizeof(struct rusage_info_v4),
    offsetof(struct rusage_info_v4, ri_resident_size),
    offsetof(struct rusage_info_v4, ri_phys_footprint),
    offsetof(struct rusage_info_v4, ri_lifetime_max_phys_footprint),
    RUSAGE_INFO_V0, RUSAGE_INFO_V4);
  struct rusage_info_v4 info = {0};
  if (proc_pid_rusage(getppid(), RUSAGE_INFO_V4, (rusage_info_t *)&info)) return 2;
  printf("%llu %llu\n", (unsigned long long)info.ri_resident_size,
    (unsigned long long)info.ri_phys_footprint);
  return 0;
}`
	binary := filepath.Join(t.TempDir(), "rusage-abi")
	compile := exec.Command(compiler, "-x", "c", "-o", binary, "-")
	compile.Stdin = strings.NewReader(program)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile SDK ABI probe: %v: %s", err, output)
	}
	output, err := exec.Command(binary).Output()
	if err != nil {
		t.Fatal(err)
	}
	var size, resident, footprint, peak uintptr
	var v0, v4 int
	if _, err := fmt.Sscanf(string(output), "%d %d %d %d %d %d", &size, &resident, &footprint, &peak, &v0, &v4); err != nil {
		t.Fatal(err)
	}
	var layout rusageV4
	if size != unsafe.Sizeof(layout) || resident != unsafe.Offsetof(layout.Resident) ||
		footprint != unsafe.Offsetof(layout.Footprint) || peak != unsafe.Offsetof(layout.Peak) || v0 != 0 || v4 != 4 {
		t.Fatalf("SDK layout %d %d %d %d %d %d differs from Go %d %d %d %d 0 4", size, resident, footprint, peak, v0, v4, unsafe.Sizeof(layout), unsafe.Offsetof(layout.Resident), unsafe.Offsetof(layout.Footprint), unsafe.Offsetof(layout.Peak))
	}
	if !supportedLayout() {
		t.Fatal("the runtime ABI guard rejected the SDK-validated layout")
	}
	var nativeRSS, nativeFootprint uint64
	if _, err := fmt.Sscanf(string(output[strings.IndexByte(string(output), '\n')+1:]), "%d %d", &nativeRSS, &nativeFootprint); err != nil {
		t.Fatal(err)
	}
	var reader nativeReader
	measured, ok := reader.read(4)
	if !ok || !withinMemoryChange(measured.Resident, nativeRSS) || !withinMemoryChange(measured.Footprint, nativeFootprint) {
		t.Fatalf("goffi RSS/footprint %d/%d (%v), native process observation %d/%d", measured.Resident, measured.Footprint, ok, nativeRSS, nativeFootprint)
	}
}

// A live process can change between the independent observations.
func withinMemoryChange(a, b uint64) bool {
	if a == 0 || b == 0 {
		return false
	}
	if a > b {
		a, b = b, a
	}
	return b-a <= max(uint64(16<<20), b/2)
}

func TestFootprintFallbackAndUnavailability(t *testing.T) {
	var calls []int32
	read := func(flavor int32) (rusageV4, bool) {
		calls = append(calls, flavor)
		if flavor == 4 {
			return rusageV4{}, false
		}
		return rusageV4{Resident: 30, Footprint: 40}, true
	}
	got := sample(read)
	if len(calls) != 2 || calls[0] != 4 || calls[1] != 0 || !got.ResidentAvailable || got.ResidentBytes != 30 || !got.FootprintAvailable || got.FootprintBytes != 40 || got.PeakFootprintAvailable {
		t.Fatalf("old API fallback: %+v, calls %v", got, calls)
	}
	got = sample(func(int32) (rusageV4, bool) { return rusageV4{}, false })
	if got.ResidentAvailable || got.FootprintAvailable || got.PeakFootprintAvailable {
		t.Fatalf("failed native sampler invented memory: %+v", got)
	}
	got = sample(func(int32) (rusageV4, bool) { return rusageV4{Resident: 30, Footprint: 40, Peak: 39}, true })
	if !got.FootprintAvailable || got.PeakFootprintAvailable {
		t.Fatalf("peak below current was reported: %+v", got)
	}
}

func TestNoCGONativeMemoryAppearsInTheScrape(t *testing.T) {
	stop := Register()
	defer stop()
	response := httptest.NewRecorder()
	observability.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	for _, want := range []string{"process_resident_memory_available 1\n", "process_physical_footprint_available 1\n", "process_cpu_available 1\n"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("missing %q in scrape:\n%s", want, response.Body.String())
		}
	}
}
