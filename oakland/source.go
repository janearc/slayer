package oakland

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sourceReader struct {
	root       string
	maxSources int64
	readBytes  int64
	sources    map[string]Source
	contents   map[string][]byte
}

type gltfInput struct {
	json []byte
	bin  []byte
}

// newSourceReader pins a resolved local root and bounded source budget.
func newSourceReader(root string, limits Limits) (*sourceReader, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("oakland: source root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("oakland: source root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("oakland: source root is not a directory")
	}
	return &sourceReader{root: resolved, maxSources: limits.SourceBytes,
		sources: make(map[string]Source), contents: make(map[string][]byte)}, nil
}

// validateLimits permits positive budgets up to the adapter's hard ceilings.
func validateLimits(limits Limits) error {
	defaults := DefaultLimits()
	if limits.SourceBytes <= 0 || limits.SourceBytes > defaults.SourceBytes ||
		limits.BufferBytes <= 0 || limits.BufferBytes > defaults.BufferBytes ||
		limits.Vertices <= 0 || limits.Vertices > defaults.Vertices ||
		limits.Indices <= 0 || limits.Indices > defaults.Indices ||
		limits.Nodes <= 0 || limits.Nodes > defaults.Nodes ||
		limits.Instances <= 0 || limits.Instances > defaults.Instances ||
		limits.Depth <= 0 || limits.Depth > defaults.Depth {
		return fmt.Errorf("oakland: resource limits must be positive and no greater than the documented defaults")
	}
	return nil
}

// localPath resolves a relative resource while keeping symlinks inside the root.
func (reader *sourceReader) localPath(relative string) (string, string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
		return "", "", fmt.Errorf("oakland: resource path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("oakland: resource path escapes source root")
	}
	path := filepath.Join(reader.root, clean)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", fmt.Errorf("oakland: resource %q: %w", relative, err)
	}
	rel, err := filepath.Rel(reader.root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("oakland: resource %q escapes source root", relative)
	}
	return resolved, filepath.ToSlash(rel), nil
}

// read retains the exact bounded bytes that are both hashed and decoded.
func (reader *sourceReader) read(relative string) (string, []byte, error) {
	resolved, canonical, err := reader.localPath(relative)
	if err != nil {
		return "", nil, err
	}
	if contents, ok := reader.contents[canonical]; ok {
		return canonical, contents, nil
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("oakland: resource %q is not a regular file", canonical)
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", nil, fmt.Errorf("oakland: read %q: %w", canonical, err)
	}
	defer file.Close()
	remaining := reader.maxSources - reader.readBytes
	if remaining < 0 {
		return "", nil, fmt.Errorf("oakland: source closure exceeds %d bytes", reader.maxSources)
	}
	limit := remaining
	if limit < math.MaxInt64 {
		limit++
	}
	contents, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return "", nil, fmt.Errorf("oakland: read %q: %w", canonical, err)
	}
	if int64(len(contents)) > remaining {
		return "", nil, fmt.Errorf("oakland: source closure exceeds %d bytes", reader.maxSources)
	}
	reader.readBytes += int64(len(contents))
	reader.contents[canonical] = contents
	reader.sources[canonical] = Source{path: canonical, size: int64(len(contents)), sha256: sha256.Sum256(contents)}
	return canonical, contents, nil
}

// sourceRecords returns the closure in canonical path order.
func (reader *sourceReader) sourceRecords() []Source {
	paths := make([]string, 0, len(reader.sources))
	for path := range reader.sources {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	sources := make([]Source, 0, len(paths))
	for _, path := range paths {
		sources = append(sources, reader.sources[path])
	}
	return sources
}

// readInput opens one supported container and checks its JSON framing.
func readInput(path string, policy Policy) (*sourceReader, gltfInput, string, error) {
	limits := policy.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	reader, err := newSourceReader(policy.Root, limits)
	if err != nil {
		return nil, gltfInput{}, "", err
	}
	canonical, raw, err := reader.read(path)
	if err != nil {
		return nil, gltfInput{}, "", err
	}
	if strings.EqualFold(filepath.Ext(path), ".glb") {
		input, err := splitGLB(raw)
		if err == nil && !json.Valid(input.json) {
			err = fmt.Errorf("oakland: GLB JSON chunk is not JSON")
		}
		return reader, input, canonical, err
	}
	if strings.EqualFold(filepath.Ext(path), ".gltf") {
		if !json.Valid(raw) {
			return nil, gltfInput{}, "", fmt.Errorf("oakland: .gltf source is not JSON")
		}
		return reader, gltfInput{json: raw}, canonical, nil
	}
	return nil, gltfInput{}, "", fmt.Errorf("oakland: source must be .gltf or .glb")
}

// splitGLB validates version, declared lengths, alignment and chunk order.
func splitGLB(raw []byte) (gltfInput, error) {
	if len(raw) < 12 || string(raw[:4]) != "glTF" || binary.LittleEndian.Uint32(raw[4:8]) != 2 {
		return gltfInput{}, fmt.Errorf("oakland: invalid GLB header")
	}
	if uint64(binary.LittleEndian.Uint32(raw[8:12])) != uint64(len(raw)) {
		return gltfInput{}, fmt.Errorf("oakland: GLB declared length does not match file")
	}
	offset := 12
	var input gltfInput
	chunk := 0
	for offset < len(raw) {
		if len(raw)-offset < 8 {
			return gltfInput{}, fmt.Errorf("oakland: truncated GLB chunk header")
		}
		size := uint64(binary.LittleEndian.Uint32(raw[offset : offset+4]))
		kind := string(raw[offset+4 : offset+8])
		offset += 8
		if size > uint64(len(raw)-offset) || size%4 != 0 {
			return gltfInput{}, fmt.Errorf("oakland: invalid GLB chunk length")
		}
		data := raw[offset : offset+int(size)]
		switch {
		case chunk == 0 && kind == "JSON":
			input.json = bytes.TrimRight(data, " \t\r\n")
		case chunk == 1 && kind == "BIN\x00":
			input.bin = data
		default:
			return gltfInput{}, fmt.Errorf("oakland: unexpected or out-of-order GLB chunk %q", kind)
		}
		offset += int(size)
		chunk++
	}
	if len(input.json) == 0 || chunk < 1 || chunk > 2 {
		return gltfInput{}, fmt.Errorf("oakland: GLB requires one JSON chunk and at most one BIN chunk")
	}
	return input, nil
}

// externalURI admits only a local relative buffer path, without URI extras.
func externalURI(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("oakland: buffer URI must be a local relative path")
	}
	path, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil || path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return "", fmt.Errorf("oakland: buffer URI must be a local relative path")
	}
	return filepath.ToSlash(path), nil
}

// validNumber excludes infinities and NaN at the geometry boundary.
func validNumber(number float64) bool { return !math.IsNaN(number) && !math.IsInf(number, 0) }
