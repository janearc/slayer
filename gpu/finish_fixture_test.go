//go:build darwin

package gpu

import (
	"os"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// finishColour is a constant authored texture and sky for host-only checks.
type finishColour [3]float64

func (colour finishColour) At(maths.Vec) maths.Light {
	return maths.RGB(colour[0], colour[1], colour[2])
}

func (colour finishColour) Light(point maths.Vec) maths.Light {
	return colour.At(point)
}

type finishMaterial struct {
	Paint, Shine      finishColour
	Diffuse, Emissive [3]float32
}

type finishExpected struct {
	Base, Scales, Offsets, Depth [3]float64
	Sky                          [3]float64
	Cover, Relative, Absolute    float64
}

type finishBook struct {
	Width, Height, Side int
	Batches             [][2]int
	SampleLanes         int `yaml:"sample-lanes"`
	TextureLanes        int `yaml:"texture-lanes"`
	Sky                 finishColour
	SkyWeights          [3]float32 `yaml:"sky-weights"`
	Hit                 float32    `yaml:"hit-diagnostics"`
	Fallback            float32    `yaml:"fallback-diagnostics"`
	Miss                float32    `yaml:"miss-diagnostics"`
	MissDepth           float32    `yaml:"miss-depth"`
	Textures            []finishMaterial
	Native              NativeCounts
	Bytes               int64 `yaml:"sample-bytes"`
	Peak                int64 `yaml:"peak-sample-bytes"`
	Expected            finishExpected
}

func readFinishBook(t *testing.T) finishBook {
	t.Helper()
	const path = "testdata/finish-bands.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var book finishBook
	if err := yamlfile.Read(path, raw, &book); err != nil {
		t.Fatal(err)
	}
	if book.SampleLanes != per || book.TextureLanes != perRecord {
		t.Fatal("the authored kernel ABI changed", book)
	}
	return book
}

func finishFixture(t *testing.T) (*bandFinisher, finishBook) {
	t.Helper()
	book := readFinishBook(t)
	paints := make([]painting, len(book.Textures))
	for i, material := range book.Textures {
		paints[i] = painting{paint: material.Paint, shine: material.Shine}
	}
	finish := finishing{world: &thing.World{Sky: book.Sky}, paints: paints,
		records: len(paints), wide: 1}
	return &bandFinisher{finish: finish,
		picture: frame.New(book.Width, book.Height), side: book.Side}, book
}

func (book finishBook) stride() int {
	return book.SampleLanes + len(book.Textures)*book.TextureLanes
}

func (book finishBook) missAt(column, row int) bool {
	return column/book.Side == book.Width-1 &&
		row/book.Side == book.Height-1
}

// writeFinishSample is an independent authored sample-layout producer.
// Its base light and depth vary in both global source coordinates.
func writeFinishSample(raw []float32, book finishBook, column, row int) {
	base := float32(column + 8*row)
	cover := float32(column%3) / 2
	depth, diagnostic := float32(1000+10*row+column), book.Hit
	if column%3 == 0 {
		diagnostic = book.Fallback
	}
	if book.missAt(column, row) {
		base, cover, depth, diagnostic = 0, 0, book.MissDepth, book.Miss
	}
	copy(raw, []float32{base, 2 * base, base / 2, cover,
		book.SkyWeights[0], book.SkyWeights[1], book.SkyWeights[2],
		depth, 0, 1, 0, diagnostic})
	writeFinishRecords(raw[book.SampleLanes:], book, column, row)
}

func writeFinishRecords(raw []float32, book finishBook, column, row int) {
	for i, material := range book.Textures {
		record := raw[i*book.TextureLanes : (i+1)*book.TextureLanes]
		if book.missAt(column, row) {
			record[0] = -1
			continue
		}
		copy(record, []float32{float32(i), float32(column), float32(row),
			0, 1, material.Diffuse[0], material.Diffuse[1],
			material.Diffuse[2], material.Emissive[0],
			material.Emissive[1], material.Emissive[2]})
	}
}

func fillFinishBand(buffer []float32, book finishBook,
	first, rows int,
) []float32 {
	width, stride := book.Width*book.Side, book.stride()
	raw := buffer[:width*rows*stride]
	for down := range rows {
		for column := range width {
			at := (down*width + column) * stride
			writeFinishSample(raw[at:at+stride], book, column, first+down)
		}
	}
	return raw
}
