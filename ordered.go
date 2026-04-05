package apispec

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
)

// KV is a key-value pair for ordered maps.
type KV[V any] struct {
	Key string
	Val V
}

// OrderedMap is an ordered collection of key-value pairs
// that marshals as a JSON/YAML object preserving insertion order.
type OrderedMap[V any] []KV[V]

// Get returns the value for a given key, or the zero value.
func (m OrderedMap[V]) Get(key string) V {
	for _, kv := range m {
		if kv.Key == key {
			return kv.Val
		}
	}
	var zero V
	return zero
}

// Has reports whether key exists.
func (m OrderedMap[V]) Has(key string) bool {
	for _, kv := range m {
		if kv.Key == key {
			return true
		}
	}
	return false
}

// MarshalJSON renders as an ordered JSON object.
func (m OrderedMap[V]) MarshalJSON() ([]byte, error) {

	buf := []byte{'{'}
	for i, kv := range m {
		if i > 0 {
			buf = append(buf, ',')
		}
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(kv.Val)
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
func (m *OrderedMap[V]) UnmarshalJSON(data []byte) error {

	dec := json.NewDecoder(bytes.NewReader(data))

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil
	}

	*m = nil
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := tok.(string)
		if !ok {
			return errors.Errorf("expected string key, got %T", tok)
		}

		var val V
		if err := dec.Decode(&val); err != nil {
			return err
		}
		*m = append(*m, KV[V]{Key: key, Val: val})
	}
	return nil
}

// MarshalYAML renders as an ordered YAML mapping.
func (m OrderedMap[V]) MarshalYAML() (any, error) {

	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, kv := range m {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: kv.Key}

		valNode := &yaml.Node{}
		data, err := yaml.Marshal(kv.Val)
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
func (m *OrderedMap[V]) UnmarshalYAML(node *yaml.Node) error {

	if node.Kind != yaml.MappingNode {
		return nil
	}

	*m = nil
	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i].Value

		var val V
		if err := node.Content[i+1].Decode(&val); err != nil {
			return err
		}
		*m = append(*m, KV[V]{Key: key, Val: val})
	}
	return nil
}

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
