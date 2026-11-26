//go:build darwin

package gpu

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/janearc/slayer/render/frame"
)

// checkedFineSize guards sampling products before scene/ray-buffer encoding.
// The maximum record stride is defined by two CPU texture records per sample.
func checkedFineSize(width, height, side int) (int, int, error) {
	if width < 0 || height < 0 || side < 1 || side > math.MaxInt32 ||
		width > math.MaxInt32/side || height > math.MaxInt32/side {
		return 0, 0, fmt.Errorf("gpu: sample dimensions exceed the kernel index range")
	}
	fineWidth, fineHeight := width*side, height*side
	strideBytes := (per + perRecord*2) * 4
	if fineHeight > 0 && fineWidth > math.MaxInt/strideBytes/fineHeight {
		return 0, 0, fmt.Errorf("gpu: sample payload size overflows the host range")
	}
	return fineWidth, fineHeight, nil
}

// bandFinisher retains pixels, not every ray's raw shading record.
// Bands arrive in order; a pixel may receive sample rows from several bands.
type bandFinisher struct {
	finish      finishing
	picture     *frame.Frame
	side, next  int
	exact       int
	native      NativeCounts
	bytes, peak int64
	spent       time.Duration
}

// consume finishes a globally addressed band before its storage is reused.
func (band *bandFinisher) consume(first, down int, samples []float32) error {
	started := time.Now()
	defer func() { band.spent += time.Since(started) }()
	if err := band.valid(first, down, samples); err != nil {
		return err
	}
	stride := per + perRecord*band.finish.records
	for sample := 0; sample < len(samples); sample += stride {
		if uint32(samples[sample+11])&(1<<20) != 0 {
			return fmt.Errorf("gpu: lunar mean-interface source path left its finite water footprint or became invalid")
		}
	}
	workReceipt(first, down, band.sampleWidth(),
		per+perRecord*band.finish.records, samples)
	counts := nativeCounts(samples, per+perRecord*band.finish.records)
	band.native.Queries += counts.Queries
	band.native.Candidates += counts.Candidates
	band.native.Refined += counts.Refined
	band.native.Fallbacks += counts.Fallbacks
	band.bytes += int64(len(samples)) * 4
	band.peak = max(band.peak, int64(cap(samples))*4)
	band.rows(first, down, samples)
	band.next += down
	// a watched drawing says its pixel rows finished, and stops here,
	// between bands, once its watch's context is done
	finished := band.next / band.side
	if band.exact > 0 {
		finished = band.next
	}
	watched.rows(finished, band.picture.H)
	return watched.stopped()
}

// valid refuses skipped/overlapping bands before they can corrupt accumulation.
func (band *bandFinisher) valid(first, down int, samples []float32) error {
	if band.exact > 0 {
		if first != band.next || down <= 0 || first < 0 || first > band.picture.H-down {
			return fmt.Errorf("gpu: invalid or out-of-order sample band")
		}
		wanted := band.sampleWidth() * down * (per + perRecord*band.finish.records)
		if len(samples) != wanted {
			return fmt.Errorf("gpu: sample band has %d numbers, needs %d", len(samples), wanted)
		}
		return nil
	}
	if first != band.next || down <= 0 || first < 0 ||
		first > band.picture.H*band.side-down {
		return fmt.Errorf("gpu: invalid or out-of-order sample band")
	}
	wanted := band.picture.W * band.side * down * (per + perRecord*band.finish.records)
	if len(samples) != wanted {
		return fmt.Errorf("gpu: sample band has %d numbers, needs %d", len(samples), wanted)
	}
	return nil
}

// rows shares independent output rows across the allowed Go CPU workers.
func (band *bandFinisher) rows(first, down int, samples []float32) {
	start, stop := first/band.side, (first+down-1)/band.side+1
	if band.exact > 0 {
		start, stop = first, first+down
	}
	jobs := make(chan int, stop-start)
	for row := start; row < stop; row++ {
		jobs <- row
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), stop-start) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for row := range jobs {
				band.row(first, down, row, samples)
			}
		}()
	}
	workers.Wait()
}

// row preserves the old down-then-across summation order for every pixel.
func (band *bandFinisher) row(first, down, row int, samples []float32) {
	if band.exact > 0 {
		band.exactRow(first, row, samples)
		return
	}
	begin, end := max(first, row*band.side), min(first+down, (row+1)*band.side)
	each := per + perRecord*band.finish.records
	fineWidth := band.picture.W * band.side
	share := 1 / float32(band.side*band.side)
	for column := range band.picture.W {
		sum := band.picture.At(column, row)
		for fineRow := begin; fineRow < end; fineRow++ {
			for across := range band.side {
				index := ((fineRow-first)*fineWidth + column*band.side + across) * each
				sample := band.finish.sample(samples[index : index+each])
				sum.R += sample.R * share
				sum.G += sample.G * share
				sum.B += sample.B * share
				sum.Cover += sample.Cover * share
				sum.Depth = min(sum.Depth, sample.Depth)
			}
		}
		band.picture.Set(column, row, sum)
	}
}

// sampleWidth is the number of dispatched samples across a band.
func (band *bandFinisher) sampleWidth() int {
	if band.exact > 0 {
		return band.picture.W * band.exact
	}
	return band.picture.W * band.side
}

// exactRow combines exactly the requested consecutive samples per pixel.
func (band *bandFinisher) exactRow(first, row int, samples []float32) {
	each := per + perRecord*band.finish.records
	share := 1 / float32(band.exact)
	for column := range band.picture.W {
		sum := band.picture.At(column, row)
		for index := 0; index < band.exact; index++ {
			at := ((row-first)*band.sampleWidth() + column*band.exact + index) * each
			sample := band.finish.sample(samples[at : at+each])
			sum.R += sample.R * share
			sum.G += sample.G * share
			sum.B += sample.B * share
			sum.Cover += sample.Cover * share
			sum.Depth = min(sum.Depth, sample.Depth)
		}
		band.picture.Set(column, row, sum)
	}
}
