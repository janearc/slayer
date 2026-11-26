package draft

import (
	"fmt"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"gopkg.in/yaml.v3"
)

// Open recognizes a Slayer job or set while preserving its exact source text.
func Open(name string, source []byte) (*Document, error) {
	var tree yaml.Node
	identity, kind, schemaFile := "https://github.com/janearc/slayer/schemas/job-v1.json", "job", "job.schema.json"
	if err := yamlfile.Read(name, source, &tree); err == nil && len(tree.Content) == 1 && tree.Content[0].Kind == yaml.MappingNode {
		root := tree.Content[0]
		if value := member(root, "$schema"); value != nil {
			identity = value.Value
			switch identity {
			case "https://github.com/janearc/slayer/schemas/job-v1.json":
			case "https://github.com/janearc/slayer/schemas/ecosystem-v1.json":
				kind, schemaFile = "set", "ecosystem.schema.json"
			case "https://github.com/janearc/slayer/schemas/ecosystem-v2.json":
				kind, schemaFile = "set", "ecosystem-v2.schema.json"
			default:
				return nil, fmt.Errorf("unsupported Slayer schema %q", identity)
			}
		} else if scene := member(root, "scene"); scene != nil && scene.Kind == yaml.MappingNode {
			kind = "legacy-set"
		} else if member(root, "details") != nil || member(root, "sample-limit") != nil || member(root, "face-limit") != nil {
			kind, schemaFile = "catalogue", "catalogue.schema.json"
			identity = "https://github.com/janearc/slayer/cmd/catalogue/plan.schema.json"
		}
	}
	raw, err := assets.Read(schemaFile)
	if err != nil {
		return nil, err
	}
	return New(Options{Name: name, Source: source, Schema: raw, Kind: kind, SchemaName: identity})
}

// Kind says whether controls belong to a job or a scene set.
func (document *Document) Kind() string { return document.kind }

// Tree returns a fresh checked syntax tree; callers never mutate stored source.
func (document *Document) Tree() (*yaml.Node, error) { return document.read(document.source) }

// read keeps legacy syntax checks at Puzzlemouth's YAML boundary.
func (document *Document) read(source []byte) (*yaml.Node, error) {
	if document.kind != "legacy-set" {
		return document.schema.Read(document.name, source)
	}
	var plain map[string]any
	if err := yamlfile.Read(document.name, source, &plain); err != nil {
		return nil, err
	}
	var tree yaml.Node
	if err := yamlfile.Read(document.name, source, &tree); err != nil {
		return nil, err
	}
	if len(tree.Content) != 1 || tree.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("a set must be a mapping")
	}
	return &tree, nil
}

// Cameras returns camera names explicitly written in the scene document.
func (document *Document) Cameras() []string {
	tree, err := document.Tree()
	if err != nil {
		return nil
	}
	scene := member(tree.Content[0], "scene")
	if scene == nil {
		return nil
	}
	cameras := member(scene, "cameras")
	if cameras == nil || cameras.Kind != yaml.MappingNode {
		return nil
	}
	var names []string
	for index := 0; index+1 < len(cameras.Content); index += 2 {
		names = append(names, cameras.Content[index].Value)
	}
	return names
}

// ReferencedSet returns a job's explicitly named scene file, if present.
func (document *Document) ReferencedSet() string {
	if document.kind != "job" {
		return ""
	}
	tree, err := document.Tree()
	if err != nil {
		return ""
	}
	path := member(member(tree.Content[0], "scene"), "set")
	if path == nil || path.Kind != yaml.ScalarNode || path.Tag != "!!str" {
		return ""
	}
	return path.Value
}
