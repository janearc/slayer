package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/render/frame"
)

// directoryMode is defined as owner rwx, group rx and other rx.
const directoryMode = 0o755

// artifactMode is defined as owner rw, group r and other r.
const artifactMode = 0o644

// settingsName is the file a specimen's plan is saved to, beside its
// picture, and settingsShown the same plan under a name that ends in
// txt: a browser downloads a yaml and will not show it.
const (
	settingsName  = "settings.yaml"
	settingsShown = "settings-yaml.txt"
)

type card struct {
	Entry   entry  `json:"entry"`
	Detail  string `json:"detail"`
	Folder  string `json:"folder"`
	Status  string `json:"status"`
	Failure string `json:"failure,omitempty"`
	Cutaway bool   `json:"cutaway,omitempty"`
	// Settings is the hash of the plan the card was drawn under, and
	// Producer the binary that drew it: both belong to the card, so a
	// book resumed by another binary keeps each card's own record.
	Settings string    `json:"settings-sha256,omitempty"`
	Producer *producer `json:"producer,omitempty"`
}

type book struct {
	Cards    []card    `json:"cards"`
	Producer *producer `json:"producer"`
}

// prepare resolves every selector and detail before creating output.
func prepare(call pzm.Call) (*book, *plan, error) {
	menu, err := readMenu()
	if err != nil {
		return nil, nil, err
	}
	entries, err := menu.selectEntries(call.Args())
	if err != nil {
		return nil, nil, err
	}
	plan, err := loadPlan(call)
	if err != nil {
		return nil, nil, err
	}
	var cards []card
	for _, entry := range entries {
		before := len(cards)
		for _, detail := range plan.Details {
			if slices.Contains(entry.Details, detail) {
				view := card{Entry: entry, Detail: detail,
					Folder: entry.Kind + "-" + detail,
					Status: pendingStatus}
				cards = append(cards, view)
			}
		}
		if len(cards) == before {
			return nil, nil, fmt.Errorf("%s supports none of %v",
				entry.Kind, plan.Details)
		}
	}
	return &book{Cards: cards}, plan, nil
}

// render writes one completed specimen at a time, keeping a truthful
// index. A specimen that fails is marked failed and the run goes on; the
// run says every failure at its end and exits with them. A destination
// that holds a book is resumed.
func (app application) render(call pzm.Call) error {
	if err := app.context.Err(); err != nil {
		return err
	}
	book, plan, err := prepare(call)
	if err != nil {
		return err
	}
	destination := filepath.Join("out", "catalogue")
	if call.Given("out") {
		destination = call.Value("out")
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "%d views -> %s\n",
			len(book.Cards), destination)
		return err
	}
	selected, err := book.open(call.Out(), destination, plan)
	if err != nil {
		return err
	}
	failures, err := app.drawAll(call, destination, book, plan, selected)
	if err != nil {
		return err
	}
	index := filepath.Join(destination, "index.html")
	if _, err = fmt.Fprintln(call.Out(), index); err != nil {
		return err
	}
	return failed(failures, len(selected))
}

// drawAll draws each selected card the book does not already hold, and
// returns a line for each that failed. A card that cannot be recorded
// or said stops the run, as does cancellation; a card that cannot be
// drawn does not.
func (app application) drawAll(call pzm.Call, destination string,
	book *book, plan *plan, selected map[string]bool,
) ([]string, error) {
	var failures []string
	settings := plan.hash()
	for index := range book.Cards {
		view := &book.Cards[index]
		if !selected[view.Folder] || view.drawnUnder(settings) {
			continue
		}
		if err := app.context.Err(); err != nil {
			return failures, err
		}
		if err := app.drawOne(destination, view, plan, book.Producer); err != nil {
			failures = append(failures, view.Folder+": "+err.Error())
		}
		if err := book.write(destination); err != nil {
			return failures, err
		}
		if view.Status != renderedStatus {
			continue
		}
		if _, err := fmt.Fprintln(call.Out(), view.Folder); err != nil {
			return failures, err
		}
	}
	return failures, nil
}

// drawOne draws one card afresh: whatever an earlier try left in its
// folder is removed first, and the card says on its face what it holds.
func (app application) drawOne(destination string, view *card, plan *plan,
	maker *producer,
) error {
	view.Status, view.Failure = pendingStatus, ""
	view.Settings, view.Producer = plan.hash(), maker
	err := os.RemoveAll(filepath.Join(destination, view.Folder))
	if err == nil {
		err = app.drawer.draw(*plan, destination, view, maker)
	}
	if err != nil {
		view.Status, view.Failure = failedStatus, err.Error()
		view.Settings, view.Producer = "", nil
		return err
	}
	view.Status = renderedStatus
	return nil
}

// failed says every specimen that failed, one line each, as the error
// the run ends with; a run with none ends well.
func failed(failures []string, selected int) error {
	if len(failures) == noEntries {
		return nil
	}
	return fmt.Errorf("%d of %d specimens failed:\n%s", len(failures),
		selected, strings.Join(failures, "\n"))
}

// draw reaches native Metal and refuses empty or unrefined mesh pictures.
func (plan plan) draw(destination string, card *card,
	producer *producer,
) error {
	start := time.Now()
	object, err := plan.construct(card.Entry, card.Detail)
	if err != nil {
		return err
	}
	card.Cutaway = object.cutaway
	prepared := time.Since(start)
	picture, profile, err := gpu.SampledAccelerated(object.world,
		*object.eye, object.lens, specimenEpoch,
		plan.Width, plan.Height, plan.Rays)
	if err != nil {
		return err
	}
	if profile.Native.Refined == noEntries || profile.Device == "" ||
		profile.TraceGPU <= specimenEpoch {
		return fmt.Errorf("specimen had no native refinement")
	}
	if err := plan.visible(picture); err != nil {
		return err
	}
	folder := filepath.Join(destination, card.Folder)
	if err := os.Mkdir(folder, directoryMode); err != nil {
		return err
	}
	imagePath := filepath.Join(folder, "specimen.png")
	if err := file.PNG(imagePath,
		plan.Tone.Develop(picture), plan.PixelAspect); err != nil {
		return err
	}
	// the settings twice: as the yaml a plan is read back from, and as
	// the same bytes under a name a browser will show, which is the one
	// the book's page links to
	for _, name := range []string{settingsName, settingsShown} {
		if err := os.WriteFile(filepath.Join(folder, name), plan.raw,
			artifactMode); err != nil {
			return err
		}
	}
	card.Status = "native GPU render"
	imageHash, err := hashFile(imagePath)
	if err != nil {
		return err
	}
	record := receipt{Card: *card, Profile: profile,
		Bounds: object.bounds, StoredFaces: object.faces,
		Seed:     plan.Seed,
		Producer: producer, DurationUnit: "nanoseconds",
		PreparationWall: prepared,
		ImageSHA256:     hex.EncodeToString(imageHash),
		SettingsSHA256:  fmt.Sprintf("%x", sha256.Sum256(plan.raw))}
	return writeJSON(filepath.Join(folder, "render.json"), record)
}

// visible measures surface coverage independently of background brightness.
func (plan plan) visible(picture *frame.Frame) error {
	covered := noEntries
	for _, pixel := range picture.Pix {
		light := float64(pixel.R) + float64(pixel.G) +
			float64(pixel.B)
		coverage := float64(pixel.Cover)
		if math.IsNaN(light+coverage) ||
			math.IsInf(light+coverage, bothInfinitySigns) ||
			coverage < zeroCoverage || coverage > fullCoverage {
			return errors.New("invalid specimen light or coverage")
		}
		if coverage > zeroCoverage {
			depth := float64(pixel.Depth)
			if !(depth > zeroLength) ||
				math.IsInf(depth, bothInfinitySigns) {
				return errors.New("invalid covered depth")
			}
			covered++
		}
	}
	share := float64(covered) / float64(len(picture.Pix))
	if !(share >= plan.MinCoverage) {
		return fmt.Errorf("specimen coverage %g is below %g",
			share, plan.MinCoverage)
	}
	return nil
}

// zeroCoverage and fullCoverage are defined probability endpoints.
const zeroCoverage, fullCoverage = 0, 1

// writeJSON writes output records; configuration is read through Puzzlemouth.
func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), artifactMode)
}
