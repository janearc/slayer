//go:build draw

package session

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/scene/thing"
)

// preparationBed observes ownership at the staging boundary. Real solver
// arithmetic and device cleanup have their separate CPU/Metal checks.
type preparationBed struct{ closed, stepped int }

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
				request: func(scene string) (asked, error) {
					if scene != plan.scene {
						t.Fatalf("request scene: %v", scene)
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

func (bed *preparationBed) At(float64) ([]fuelbed.Cell, error) {
	bed.stepped++
	return nil, errors.New("fuel stepped during failed sequence preparation")
}

func (bed *preparationBed) Close() { bed.closed++ }
