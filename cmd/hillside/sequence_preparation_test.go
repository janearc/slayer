package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/scene/thing"
)

// preparationBed observes ownership at the staging boundary. Real solver
// arithmetic and device cleanup have their separate CPU/Metal checks.
type preparationBed struct{ closed, stepped int }

func (bed *preparationBed) At(float64) ([]fuelbed.Cell, error) {
	bed.stepped++
	return nil, errors.New("fuel stepped during failed sequence preparation")
}
func (bed *preparationBed) Close() { bed.closed++ }

func TestSequencePreparationPropagatesBoundaryFailuresAndClosesOwnedBeds(t *testing.T) {
	for _, fault := range []string{"parse", "stage", "scene read", "scene changed", "executable", "fingerprint", "folder"} {
		t.Run(fault, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "", "")
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			plan := sequencePlan{scene: given.set, output: filepath.Join(t.TempDir(), "frames"), frames: 1, fps: 1}
			executable := filepath.Join(t.TempDir(), "renderer")
			sequenceWrite(t, executable, []byte("an actual executable fixture"))
			_, missingErr := os.Open(filepath.Join(t.TempDir(), "absent"))
			if !errors.Is(missingErr, os.ErrNotExist) {
				t.Fatal(missingErr)
			}
			_, parseErr := parse([]string{"--not-a-hillside-option"}, io.Discard)
			if parseErr == nil {
				t.Fatal("invalid parser fixture succeeded")
			}
			wantCause := missingErr
			if fault == "parse" {
				wantCause = parseErr
			}
			bed := &preparationBed{}
			given.live = &liveScene{beds: []retainedBed{{model: bed}}}
			stages, executables := 0, 0
			prepare := sequencePreparation{
				parse: func(args []string, _ io.Writer) (asked, error) {
					if !reflect.DeepEqual(args, []string{"-set", plan.scene}) {
						t.Fatalf("parse arguments: %v", args)
					}
					if fault == "parse" {
						return asked{}, parseErr
					}
					return given, nil
				},
				stage: func(asked) (*thing.World, tally, asked, error) {
					stages++
					if fault == "stage" {
						return nil, tally{}, asked{}, missingErr
					}
					if fault == "scene read" {
						if err := os.Remove(plan.scene); err != nil {
							t.Fatal(err)
						}
					}
					if fault == "scene changed" {
						sequenceWrite(t, plan.scene, append(raw, []byte("\n# changed after staging\n")...))
					}
					if fault == "fingerprint" {
						path, err := nearest(filepath.Dir(plan.scene), "sources.yaml")
						if err != nil {
							t.Fatal(err)
						}
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
					}
					return &thing.World{}, tally{}, given, nil
				},
				executable: func() (string, error) {
					executables++
					if fault == "executable" {
						return "", missingErr
					}
					return executable, nil
				},
			}
			if fault == "folder" {
				sequenceWrite(t, plan.output, []byte("keep this file"))
			}
			err = drawPreparedSequence(plan, raw, false, io.Discard, prepare)
			if err == nil {
				t.Fatal("failed preparation rendered")
			}
			if fault == "parse" || fault == "stage" || fault == "executable" {
				if !errors.Is(err, wantCause) {
					t.Fatalf("boundary lost its cause: %v", err)
				}
			} else {
				want := map[string]string{"scene read": plan.scene, "scene changed": "scene changed while preparing", "fingerprint": "sources.yaml", "folder": plan.output}[fault]
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("wrong refusal for %s: %v", fault, err)
				}
			}
			wantClosed := 1
			if fault == "parse" || fault == "stage" {
				wantClosed = 0
			}
			if bed.closed != wantClosed || bed.stepped != 0 {
				t.Fatalf("ownership: closed %d, stepped %d", bed.closed, bed.stepped)
			}
			if fault == "parse" && stages != 0 {
				t.Fatal("staged after parse failure")
			}
			if (fault == "parse" || fault == "stage" || fault == "scene read" || fault == "scene changed") && executables != 0 {
				t.Fatal("requested executable after earlier failure")
			}
		})
	}
}

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
