package telemetry

/*
#include <mach/mach.h>
#include <stddef.h>
#include <stdint.h>

typedef struct {
    uint64_t page_bytes;
    uint64_t free_pages;
    uint64_t active_pages;
    uint64_t inactive_pages;
    uint64_t wired_pages;
    uint64_t speculative_pages;
    uint64_t compressor_pages;
    int compressor_known;
} azazel_vm_pages;

static int azazel_read_vm_pages(azazel_vm_pages *out) {
    vm_statistics64_data_t info = {0};
    mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
    mach_port_t host = mach_host_self();
    if (host == MACH_PORT_NULL) return -1;
    vm_size_t page_bytes = 0;
    kern_return_t page_result = host_page_size(host, &page_bytes);
    kern_return_t stats_result = host_statistics64(host, HOST_VM_INFO64,
        (host_info64_t)&info, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (page_result != KERN_SUCCESS || stats_result != KERN_SUCCESS ||
        page_bytes == 0 || count < HOST_VM_INFO64_REV0_COUNT) return -1;
    out->page_bytes = page_bytes;
    out->free_pages = info.free_count;
    out->active_pages = info.active_count;
    out->inactive_pages = info.inactive_count;
    out->wired_pages = info.wire_count;
    out->speculative_pages = info.speculative_count;
    const size_t compressor_end = offsetof(vm_statistics64_data_t,
        compressor_page_count) + sizeof(info.compressor_page_count);
    const size_t words = (compressor_end + sizeof(integer_t) - 1) / sizeof(integer_t);
    if (count >= words) {
        out->compressor_known = 1;
        out->compressor_pages = info.compressor_page_count;
    }
    return 0;
}
*/
import "C"

import "fmt"

// vmPageStates is a host snapshot. Speculative pages are already in Free.
// Compressor is physical memory used to hold compressed pages, not the
// uncompressed size, swap size or a separately additive total.
type vmPageStates struct {
	PageBytes, Free, Active, Inactive, Wired, Speculative, Compressor uint64
	CompressorKnown                                                   bool
}

// vm reads the kernel's page counts and compressor state for the host.
func (nativeHost) vm() (vmPageStates, error) {
	var native C.azazel_vm_pages
	if C.azazel_read_vm_pages(&native) != 0 {
		return vmPageStates{}, fmt.Errorf("host VM page states unavailable")
	}
	return vmPageStates{
		PageBytes: uint64(native.page_bytes), Free: uint64(native.free_pages),
		Active: uint64(native.active_pages), Inactive: uint64(native.inactive_pages),
		Wired: uint64(native.wired_pages), Speculative: uint64(native.speculative_pages),
		Compressor: uint64(native.compressor_pages), CompressorKnown: native.compressor_known != 0,
	}, nil
}
