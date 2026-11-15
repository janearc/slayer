//go:build darwin

package metal

/*
#include "library.h"
*/
import "C"

// cachedLibraries is how many distinct sources the process's shader
// cache holds, for the test that shows a second open compiles nothing.
func cachedLibraries() int { return int(C.slayer_shared_library_count()) }
