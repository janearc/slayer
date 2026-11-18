package set

import (
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oakland"
	yaml "gopkg.in/yaml.v3"
)

// Import is a checked asset, explicit native bindings and a set placement.
type Import struct {
	name, file, source string
	license            string
	asset              *oakland.Asset
	bindings           map[string]string
	at                 [3]float64
	heading            float64
	scene              *int
	height             float64
	root               string
	node               *yaml.Node
}

// Name identifies this placement in the set.
func (made Import) Name() string { return made.name }

// Asset is the immutable geometry and exact byte closure read for this set.
func (made Import) Asset() *oakland.Asset { return made.asset }

// File is the absolute entry file, whose bytes are already in Asset.
func (made Import) File() string { return made.file }

// Source records the author's account of the asset and its choices.
func (made Import) Source() string { return made.source }

// License is the author's explicit rights declaration for this asset.
func (made Import) License() string { return made.license }

// Bindings maps source slots to set material names, in a fresh map.
func (made Import) Bindings() map[string]string { return maps.Clone(made.bindings) }

// Placement is the asset's metre space placed in a world of unit metres.
func (made Import) Placement(unit float64) maths.Subspace {
	return maths.Subspace{Origin: fromMap(made.at), Scale: 1 / unit,
		Orientation: maths.About(maths.YAxis, -made.heading*maths.Degree)}
}

// Imports is every imported placement in authored order, in a fresh slice.
func (made World) Imports() []Import { return slices.Clone(made.imports) }

// imports reads explicit placements; geometry is opened after all materials exist.
func (reading reader) imports(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Import{name: name.Value, node: block, bindings: map[string]string{}}
		var scene float64
		err := reading.keys(block, map[string]step{
			"file":    reading.text(&made.file),
			"scene":   reading.whole(&scene, 0, math.MaxInt32),
			"height":  reading.quantity(&made.height, "m", math.SmallestNonzeroFloat64, 1e6),
			"at":      reading.three(&made.at, -farthest, farthest),
			"heading": reading.quantity(&made.heading, "deg", 0, 360),
			"source":  reading.source(&made.source),
			"license": reading.text(&made.license),
			"materials": func(_, bindings *yaml.Node) error {
				return reading.each(bindings, func(slot, value *yaml.Node) error {
					var target string
					if err := reading.text(&target)(slot, value); err != nil {
						return err
					}
					made.bindings[slot.Value] = target
					return nil
				})
			},
		}, nil)
		if err != nil {
			return err
		}
		if err := reading.all(block, "an import", "file", "at", "materials", "source", "license"); err != nil {
			return err
		}
		if strings.TrimSpace(made.license) == "" {
			return reading.refuse(block, "import license must name the source's rights")
		}
		if says(block, "scene") {
			picked := int(scene)
			made.scene = &picked
		}
		if filepath.IsAbs(made.file) {
			return reading.refuse(block, "import file is relative to its set, not absolute")
		}
		reading.set.world.imports = append(reading.set.world.imports, made)
		return nil
	})
}

// imported opens assets once per set and validates all native material bindings.
func (reading reader) imported() error {
	world := reading.set.world
	if world == nil || len(world.imports) == 0 {
		return nil
	}
	root, err := filepath.Abs(filepath.Dir(reading.file))
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, material := range world.materials {
		known[material.name] = true
	}
	cache := map[string]*oakland.Asset{}
	for index := range world.imports {
		part := &world.imports[index]
		relative := part.file
		part.root, part.file = root, filepath.Join(root, part.file)
		selected := -1
		if part.scene != nil {
			selected = *part.scene
		}
		key := fmt.Sprintf("%s\x00%d\x00%g", part.file, selected, part.height)
		part.asset = cache[key]
		if part.asset == nil {
			policy := oakland.Policy{Root: root, Scene: part.scene, TargetHeight: part.height, Limits: oakland.DefaultLimits()}
			if part.height > 0 {
				policy.HeightUnit = "m"
			}
			part.asset, err = oakland.Read(relative, policy)
			if err != nil {
				return reading.refuse(part.node, "import %s: %v", part.name, err)
			}
			cache[key] = part.asset
		}
		if err := reading.importBindings(part, known); err != nil {
			return err
		}
	}
	return reading.consistentImports()
}

// importBindings refuses missing slots, stray bindings and unknown native materials.
func (reading reader) importBindings(part *Import, known map[string]bool) error {
	slots := map[string]bool{}
	for _, slot := range part.asset.MaterialSlots() {
		slots[slot] = true
	}
	for slot := range slots {
		target, has := part.bindings[slot]
		if !has {
			return reading.refuse(part.node, "import %s needs a native material for slot %q", part.name, slot)
		}
		if !known[target] {
			return reading.refuse(part.node, "import %s names unknown native material %q", part.name, target)
		}
	}
	for slot := range part.bindings {
		if !slots[slot] {
			return reading.refuse(part.node, "import %s binds unused slot %q", part.name, slot)
		}
	}
	return nil
}
