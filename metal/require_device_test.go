//go:build darwin

package metal

import (
	"errors"
	"testing"
)

// requireDevice follows the package convention: only a genuinely missing
// Metal device skips a positive hardware check. Compiler and native-query
// errors on an available device still fail.
func requireDevice(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
}
