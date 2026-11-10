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

type card struct {
	Entry   entry  `json:"entry"`
	Detail  string `json:"detail"`
	Folder  string `json:"folder"`
	Status  string `json:"status"`
	Failure string `json:"failure,omitempty"`
	Cutaway bool   `json:"cutaway,omitempty"`
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
					Status: "pending"}
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

// render writes one completed specimen at a time, keeping a truthful index.
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
	if err := book.start(destination); err != nil {
		return err
	}
	for index := range book.Cards {
		if err := app.context.Err(); err != nil {
			return err
		}
		err := plan.draw(destination, &book.Cards[index], book.Producer)
		if err != nil {
			book.Cards[index].Status = "failed"
			book.Cards[index].Failure = err.Error()
		}
		writeError := book.write(destination)
		if err != nil || writeError != nil {
			return errors.Join(err, writeError)
		}
		_, err = fmt.Fprintln(call.Out(), book.Cards[index].Folder)
		if err != nil {
			return err
		}
	}
	index := filepath.Join(destination, "index.html")
	_, err = fmt.Fprintln(call.Out(), index)
	return err
}

// start creates a new book with the identity of its actual producer.
func (book *book) start(destination string) error {
	var err error
	book.Producer, err = compiledProducer()
	if err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, directoryMode); err != nil {
		return err
	}
	if err := os.Mkdir(destination, directoryMode); err != nil {
		return err
	}
	if err := writeIcons(destination); err != nil {
		return err
	}
	return book.write(destination)
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
	if err := os.WriteFile(filepath.Join(folder, "settings.yaml"),
		plan.raw, artifactMode); err != nil {
		return err
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
