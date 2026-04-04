package apispec

import (
	"os"
	"path/filepath"
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

	var doc Document
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
	if len(doc.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(doc.Paths))
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
