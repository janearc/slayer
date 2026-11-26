package draft

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// StockChoice reads a job's stock if no conflicting print curve is authored.
func (document *Document) StockChoice() (string, bool) {
	if document.kind != "job" {
		return "", false
	}
	tree, err := document.Tree()
	if err != nil {
		return "", false
	}
	root := tree.Content[0]
	if member(root, "print") != nil {
		return "", false
	}
	stock := member(root, "stock")
	if stock == nil {
		return "", true
	}
	name := member(stock, "name")
	if name == nil {
		return "", false
	}
	return name.Value, true
}

// SetStock changes a job's development stock and retains its seed and rating.
func (document *Document) SetStock(name string) error {
	current, ok := document.StockChoice()
	if !ok {
		return fmt.Errorf("stock cannot be changed while a print curve is authored")
	}
	if current == name {
		return nil
	}
	tree, err := document.Tree()
	if err != nil {
		return err
	}
	root := tree.Content[0]
	stock := member(root, "stock")
	if stock == nil {
		stock = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "stock"}, stock)
	}
	value := member(stock, "name")
	if value == nil {
		value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str"}
		stock.Content = append(stock.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"}, value)
	}
	value.Value = name
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(tree); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return fmt.Errorf("stock edit would make invalid job YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}
