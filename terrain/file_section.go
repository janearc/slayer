package terrain

import (
	"bufio"
	"fmt"
	"io"
)

// section validates source offsets before their signed file API conversion.
func (raster *Raster) section(offset, count uint64) (*io.SectionReader, error) {
	if offset > uint64(raster.fileBytes) ||
		count > uint64(raster.fileBytes)-offset {
		return nil, fmt.Errorf("terrain: TIFF payload is outside the source file")
	}
	return io.NewSectionReader(raster.file, int64(offset), int64(count)), nil
}

// readMetadata accounts for the whole directory's raw payload budget.
func (raster *Raster) readMetadata(offset uint64, size int) ([]byte, error) {
	if int64(size) > raster.budget.MetadataBytes-raster.metadataBytes {
		return nil, fmt.Errorf("terrain: TIFF metadata exceeds its byte budget")
	}
	input, err := raster.section(offset, uint64(size))
	if err != nil {
		return nil, err
	}
	raw := make([]byte, size)
	if _, err := io.ReadFull(input, raw); err != nil {
		return nil, err
	}
	raster.metadataBytes += int64(size)
	return raw, nil
}

// newSampleReader uses Go's bounded default buffer for small float reads.
func newSampleReader(input io.Reader) *bufio.Reader {
	return bufio.NewReader(input)
}
