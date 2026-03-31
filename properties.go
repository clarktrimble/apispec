package apispec

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
)

// Todo: Consider consolidating Properties with OrderedMap[*Schema].
// Properties predates OrderedMap and duplicates the same pattern with
// different field names (Name/Schema vs Key/Val). Name/Schema reads better
// at call sites, so the tradeoff is less code vs less readability.

// Property is a named schema entry.
type Property struct {
	Name   string
	Schema *Schema
}

// Properties is an ordered collection of named schemas.
type Properties []Property

// Get returns the schema for a given name, or nil.
func (ps Properties) Get(name string) *Schema {
	for _, p := range ps {
		if p.Name == name {
			return p.Schema
		}
	}
	return nil
}

// MarshalJSON renders as an ordered JSON object.
func (ps Properties) MarshalJSON() ([]byte, error) {

	buf := []byte{'{'}
	for i, p := range ps {
		if i > 0 {
			buf = append(buf, ',')
		}
		key, err := json.Marshal(p.Name)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(p.Schema)
		if err != nil {
			return nil, err
		}
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = append(buf, val...)
	}
	buf = append(buf, '}')
	return buf, nil
}

// UnmarshalJSON reads a JSON object preserving key order.
func (ps *Properties) UnmarshalJSON(data []byte) error {

	dec := json.NewDecoder(bytes.NewReader(data))

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil
	}

	*ps = nil
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name, ok := tok.(string)
		if !ok {
			return errors.Errorf("expected string key, got %T", tok)
		}

		var schema Schema
		if err := dec.Decode(&schema); err != nil {
			return err
		}
		*ps = append(*ps, Property{Name: name, Schema: &schema})
	}
	return nil
}

// MarshalYAML renders as an ordered YAML mapping.
func (ps Properties) MarshalYAML() (any, error) {

	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	for _, p := range ps {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: p.Name}

		valNode := &yaml.Node{}
		data, err := yaml.Marshal(p.Schema)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, valNode); err != nil {
			return nil, err
		}
		if valNode.Kind == yaml.DocumentNode && len(valNode.Content) > 0 {
			valNode = valNode.Content[0]
		}

		node.Content = append(node.Content, keyNode, valNode)
	}
	return node, nil
}

// UnmarshalYAML reads a YAML mapping preserving key order.
func (ps *Properties) UnmarshalYAML(node *yaml.Node) error {

	if node.Kind != yaml.MappingNode {
		return nil
	}

	*ps = nil
	for i := 0; i < len(node.Content)-1; i += 2 {
		name := node.Content[i].Value

		var schema Schema
		if err := node.Content[i+1].Decode(&schema); err != nil {
			return err
		}
		*ps = append(*ps, Property{Name: name, Schema: &schema})
	}
	return nil
}
