package session

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

func TestSequenceReportsSchemaCompilationAndTypedDecodeDrift(t *testing.T) {
	path, _ := sequenceFixture(t)
	beforeSchema, beforeError := sequenceSchema, sequenceSchemaError
	t.Cleanup(func() { sequenceSchema, sequenceSchemaError = beforeSchema, beforeError })
	_, compileErr := yamlfile.NewSchema("https://example.test/broken.json", []byte("{"))
	if compileErr == nil {
		t.Fatal("invalid schema compiled")
	}
	sequenceSchemaError = compileErr
	if _, err := readSequence(path); !errors.Is(err, compileErr) {
		t.Fatalf("schema compilation cause: %v", err)
	}
	sequenceSchemaError = nil
	// Deliberately replace only this test's schema with a permissive real
	// Puzzlemouth schema. This checks the typed-decoder boundary if schema
	// and Go fields drift; production schema refusals are checked separately.
	loose, err := yamlfile.NewSchema("https://example.test/loose.json", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	sequenceSchema = loose
	sequenceWrite(t, path, []byte("fps: {}"))
	if _, err := readSequence(path); err == nil || !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("typed schema drift: %v", err)
	}
}

func TestSequenceRefusesAnUnresolvableRelativePlanBeforeReading(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("PWD", "/")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Error(err)
		}
	})
	_, err := readSequence("plan.yaml")
	var pathError *os.PathError
	var systemError *os.SyscallError
	resolving := errors.As(err, &pathError) && pathError.Op == "stat"
	resolving = resolving || errors.As(err, &systemError) && systemError.Syscall == "getwd"
	if !errors.Is(err, os.ErrPermission) || !resolving {
		t.Fatalf("unresolvable cwd: %v", err)
	}
}
