package apispec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestGenerate(t *testing.T) {

	outPath := filepath.Join(t.TempDir(), "openapi.yaml")

	err := Generate("testdata/apispec.yaml", outPath)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	// document metadata
	if doc.OpenAPI != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %s", doc.OpenAPI)
	}
	if doc.Info.Title != "Widget Service API" {
		t.Errorf("expected title 'Widget Service API', got %s", doc.Info.Title)
	}

	// paths from fragment
	if len(doc.Paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(doc.Paths))
	}
	if doc.Paths.Get("/widgets") == nil {
		t.Error("missing /widgets path")
	}
	if doc.Paths.Get("/widgets/{id}") == nil {
		t.Error("missing /widgets/{id} path")
	}

	// tags not collected from fragments
	if len(doc.Tags) != 0 {
		t.Errorf("expected no tags, got %d", len(doc.Tags))
	}

	// Widget schema from types
	if doc.Components == nil {
		t.Fatal("missing components")
	}
	if doc.Components.Schemas["Widget"] == nil {
		t.Fatal("missing Widget schema")
	}
	if doc.Components.Schemas["Widget"].Type != "object" {
		t.Errorf("expected Widget type object, got %s", doc.Components.Schemas["Widget"].Type)
	}

	// Part schema from transitive dep
	if doc.Components.Schemas["Part"] == nil {
		t.Error("missing Part schema (transitive dep)")
	}

	// ServerConfig from config
	if doc.Components.Schemas["ServerConfig"] == nil {
		t.Fatal("missing ServerConfig schema")
	}
	cfg := doc.Components.Schemas["ServerConfig"]
	if cfg.Properties.Get("version") != nil {
		t.Error("version should be ignored in config schema")
	}
	if cfg.Properties.Get("port") == nil {
		t.Error("missing port in config schema")
	}

	t.Logf("output:\n%s", string(data))
}

func TestGenerateComposition(t *testing.T) {

	outPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := Generate("testdata/apispec.yaml", outPath); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	t.Logf("output:\n%s", data)

	var generated document
	if err := yaml.Unmarshal(data, &generated); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	want := &schema{
		Type: "object",
		Properties: properties{{Name: "events", Schema: &schema{
			Type: "array",
			Items: &schema{AllOf: []*schema{
				ref("EventEnvelope"),
				{
					Type: "object",
					Properties: properties{{Name: "event", Schema: &schema{
						OneOf: []*schema{
							ref("Status"),
							ref("Audit"),
							ref("InventoryAudit"),
							ref("Finding"),
							ref("Observation"),
						},
					}}},
				},
			}},
		}}},
	}

	// Generate writes YAML; JSON is also supported by the document model.
	jsonData, err := json.Marshal(generated)
	if err != nil {
		t.Fatalf("marshaling JSON: %v", err)
	}
	for _, tc := range []struct {
		name      string
		data      []byte
		unmarshal func([]byte, any) error
		marshal   func(any) ([]byte, error)
	}{
		{"YAML", data, yaml.Unmarshal, yaml.Marshal},
		{"JSON", jsonData, json.Unmarshal, json.Marshal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var doc document
			if err := tc.unmarshal(tc.data, &doc); err != nil {
				t.Fatalf("unmarshaling document: %v", err)
			}
			path := doc.Paths.Get("/events")
			if path == nil || path.Get == nil {
				t.Fatal("missing GET /events")
			}
			response := path.Get.Responses.Get("200")
			if response == nil || response.Content["application/json"] == nil {
				t.Fatal("missing JSON response")
			}
			if got := response.Content["application/json"].Schema; !reflect.DeepEqual(got, want) {
				t.Errorf("response composition changed: got %#v, want %#v", got, want)
			}

			if doc.Components == nil {
				t.Fatal("missing components")
			}
			for name, field := range map[string]string{
				"EventEnvelope":  "event",
				"Status":         "status",
				"Audit":          "action",
				"InventoryAudit": "changes",
				"Finding":        "finding",
				"Observation":    "observation",
			} {
				component := doc.Components.Schemas[name]
				if component == nil || component.Type != "object" || component.Properties.Get(field) == nil {
					t.Errorf("missing generated object schema or property: %s.%s", name, field)
				}
			}
			// Composition is not inferred from the envelope's Go any field.
			envelope := doc.Components.Schemas["EventEnvelope"]
			if envelope == nil || !reflect.DeepEqual(envelope.Properties.Get("event"), &schema{Type: "object"}) {
				t.Error("EventEnvelope.event should retain its generated object schema")
			}

			// Neither generated components nor the plain response gain empty fields.
			for name, value := range map[string]any{
				"components": doc.Components,
				"widgets":    doc.Paths.Get("/widgets"),
			} {
				plain, err := tc.marshal(value)
				if err != nil {
					t.Fatalf("marshaling %s: %v", name, err)
				}
				if strings.Contains(string(plain), "allOf") || strings.Contains(string(plain), "oneOf") {
					t.Errorf("unexpected composition fields in %s:\n%s", name, plain)
				}
			}
		})
	}
}

func TestGenerateRequestBody(t *testing.T) {

	outPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := Generate("testdata/apispec.yaml", outPath); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	t.Logf("output:\n%s", data)

	var generated document
	if err := yaml.Unmarshal(data, &generated); err != nil {
		t.Fatalf("parsing output: %v", err)
	}
	jsonData, err := json.Marshal(generated)
	if err != nil {
		t.Fatalf("marshaling JSON: %v", err)
	}

	for _, tc := range []struct {
		name      string
		data      []byte
		unmarshal func([]byte, any) error
	}{
		{"YAML", data, yaml.Unmarshal},
		{"JSON", jsonData, json.Unmarshal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Inspect parsed output independently of the model, including omitted keys.
			var output map[string]any
			if err := tc.unmarshal(tc.data, &output); err != nil {
				t.Fatalf("parsing document: %v", err)
			}
			objectAt := func(keys ...string) map[string]any {
				t.Helper()
				object := output
				for _, key := range keys {
					next, ok := object[key].(map[string]any)
					if !ok {
						t.Fatalf("expected object at %v, got %T for %s", keys, object[key], key)
					}
					object = next
				}
				return object
			}

			body := objectAt("paths", "/events", "post", "requestBody")
			const description = "One event object per line.\nBlank lines are ignored.\n"
			if body["description"] != description {
				t.Errorf("description: got %q, want %q", body["description"], description)
			}
			if body["required"] != true {
				t.Errorf("required: got %v, want true", body["required"])
			}
			content := objectAt("paths", "/events", "post", "requestBody", "content")
			if len(content) != 2 {
				t.Errorf("expected 2 content types, got %d", len(content))
			}

			ndjson := objectAt("paths", "/events", "post", "requestBody", "content", "application/x-ndjson")
			const example = "{\"event_type\":\"observations\",\"payload\":{\"id\":\"one\"}}\n" +
				"{\"event_type\":\"observations\",\"payload\":{\"id\":\"two\"}}\n"
			if got, ok := ndjson["example"].(string); !ok || got != example {
				t.Errorf("NDJSON example: got %#v, want string %q", ndjson["example"], example)
			}
			if !reflect.DeepEqual(ndjson["schema"], map[string]any{"type": "string"}) {
				t.Errorf("NDJSON schema changed: %#v", ndjson["schema"])
			}

			structured := objectAt("paths", "/events", "post", "requestBody", "content", "application/json")
			wantExample := map[string]any{
				"event_type": "observations",
				"payload":    map[string]any{"id": "one"},
			}
			if !reflect.DeepEqual(structured["example"], wantExample) {
				t.Errorf("structured example: got %#v, want %#v", structured["example"], wantExample)
			}
			if !reflect.DeepEqual(structured["schema"], map[string]any{"type": "object"}) {
				t.Errorf("JSON schema changed: %#v", structured["schema"])
			}

			plain := objectAt("paths", "/widgets", "post", "requestBody")
			if _, exists := plain["description"]; exists {
				t.Error("absent request-body description should be omitted")
			}
			media := objectAt("paths", "/widgets", "post", "requestBody", "content", "application/json")
			if _, exists := media["example"]; exists {
				t.Error("absent media-type example should be omitted")
			}
		})
	}
}

func TestGenerateNameCollision(t *testing.T) {

	outPath := filepath.Join(t.TempDir(), "openapi.yaml")

	err := Generate("testdata/name_collision.yaml", outPath)
	if err == nil {
		t.Fatal("expected error for name collision")
	}
	if !strings.Contains(err.Error(), "schema name collision") {
		t.Errorf("expected 'schema name collision' in error, got: %v", err)
	}
}

func TestGenerateDuplicatePaths(t *testing.T) {

	outPath := filepath.Join(t.TempDir(), "openapi.yaml")

	err := Generate("testdata/duplicate_paths.yaml", outPath)
	if err == nil {
		t.Fatal("expected error for duplicate paths")
	}
	if !strings.Contains(err.Error(), "duplicate path") {
		t.Errorf("expected 'duplicate path' in error, got: %v", err)
	}
}
