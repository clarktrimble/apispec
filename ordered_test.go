package apispec

import (
	"encoding/json"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// OrderedMap tests

func TestOrderedMapJSONRoundTrip(t *testing.T) {

	m := OrderedMap[int]{
		{Key: "z", Val: 3},
		{Key: "a", Val: 1},
		{Key: "m", Val: 2},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// keys must appear in insertion order, not sorted
	expected := `{"z":3,"a":1,"m":2}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var got OrderedMap[int]
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got) != len(m) {
		t.Fatalf("expected %d entries, got %d", len(m), len(got))
	}
	for i, kv := range m {
		if got[i].Key != kv.Key || got[i].Val != kv.Val {
			t.Errorf("entry %d: expected {%s %d}, got {%s %d}",
				i, kv.Key, kv.Val, got[i].Key, got[i].Val)
		}
	}
}

func TestOrderedMapYAMLRoundTrip(t *testing.T) {

	m := OrderedMap[int]{
		{Key: "z", Val: 3},
		{Key: "a", Val: 1},
		{Key: "m", Val: 2},
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// verify key order in raw YAML output
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	expectedKeys := []string{"z:", "a:", "m:"}
	for i, want := range expectedKeys {
		if i >= len(lines) || !strings.HasPrefix(lines[i], want) {
			t.Errorf("line %d: expected prefix %q, got %q", i, want, lines[i])
		}
	}

	var got OrderedMap[int]
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got) != len(m) {
		t.Fatalf("expected %d entries, got %d", len(m), len(got))
	}
	for i, kv := range m {
		if got[i].Key != kv.Key || got[i].Val != kv.Val {
			t.Errorf("entry %d: expected {%s %d}, got {%s %d}",
				i, kv.Key, kv.Val, got[i].Key, got[i].Val)
		}
	}
}

func TestOrderedMapGetAndHas(t *testing.T) {

	m := OrderedMap[int]{
		{Key: "a", Val: 1},
		{Key: "b", Val: 2},
	}

	if v := m.Get("a"); v != 1 {
		t.Errorf("Get(a): expected 1, got %d", v)
	}
	if v := m.Get("missing"); v != 0 {
		t.Errorf("Get(missing): expected 0, got %d", v)
	}
	if !m.Has("a") {
		t.Error("Has(a): expected true")
	}
	if m.Has("missing") {
		t.Error("Has(missing): expected false")
	}
}

func TestOrderedMapEmpty(t *testing.T) {

	var m OrderedMap[int]

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("expected {}, got %s", string(data))
	}
}

// Properties tests

func TestPropertiesJSONRoundTrip(t *testing.T) {

	ps := Properties{
		{Name: "z_field", Schema: &Schema{Type: "string"}},
		{Name: "a_field", Schema: &Schema{Type: "integer"}},
	}

	data, err := json.Marshal(ps)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// names must appear in insertion order
	expected := `{"z_field":{"type":"string"},"a_field":{"type":"integer"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var got Properties
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got) != len(ps) {
		t.Fatalf("expected %d properties, got %d", len(ps), len(got))
	}
	for i, p := range ps {
		if got[i].Name != p.Name || got[i].Schema.Type != p.Schema.Type {
			t.Errorf("property %d: expected {%s %s}, got {%s %s}",
				i, p.Name, p.Schema.Type, got[i].Name, got[i].Schema.Type)
		}
	}
}

func TestPropertiesYAMLRoundTrip(t *testing.T) {

	ps := Properties{
		{Name: "z_field", Schema: &Schema{Type: "string"}},
		{Name: "a_field", Schema: &Schema{Type: "integer"}},
	}

	data, err := yaml.Marshal(ps)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// verify name order in raw YAML
	output := string(data)
	zIdx := strings.Index(output, "z_field")
	aIdx := strings.Index(output, "a_field")
	if zIdx < 0 || aIdx < 0 {
		t.Fatalf("missing field names in output:\n%s", output)
	}
	if zIdx > aIdx {
		t.Errorf("z_field should appear before a_field in output:\n%s", output)
	}

	var got Properties
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got) != len(ps) {
		t.Fatalf("expected %d properties, got %d", len(ps), len(got))
	}
	for i, p := range ps {
		if got[i].Name != p.Name || got[i].Schema.Type != p.Schema.Type {
			t.Errorf("property %d: expected {%s %s}, got {%s %s}",
				i, p.Name, p.Schema.Type, got[i].Name, got[i].Schema.Type)
		}
	}
}

func TestPropertiesGet(t *testing.T) {

	ps := Properties{
		{Name: "id", Schema: &Schema{Type: "string"}},
		{Name: "count", Schema: &Schema{Type: "integer"}},
	}

	if s := ps.Get("id"); s == nil || s.Type != "string" {
		t.Errorf("Get(id): expected string schema, got %v", s)
	}
	if s := ps.Get("missing"); s != nil {
		t.Errorf("Get(missing): expected nil, got %v", s)
	}
}

func TestPropertiesEmpty(t *testing.T) {

	var ps Properties

	data, err := json.Marshal(ps)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("expected {}, got %s", string(data))
	}
}
