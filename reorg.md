
  ---
  Current state

  11 source files in root package apispec:

  ┌─────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │        File         │                                                What's in it                                                │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ document.go         │ 15 OpenAPI types (Document, Info, Schema, Operation, etc.), Ref(), OpenAPIVersion                          │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ ordered.go          │ OrderedMap[V], KV[V] — generic ordered map with JSON/YAML marshaling                                       │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ properties.go       │ Properties, Property — specialized ordered map for schema properties                                       │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ config.go           │ Config, TypeRef, Spec — YAML config for the generator                                                      │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ generate.go         │ Generate() — the main pipeline, plus loadConfig, loadPackages                                              │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ schema.go           │ schemaEntry, deps, resolveAll, schemaFrom, typeToSchema, namedSchema, structSchema, configSchemaFrom,      │
  │                     │ configStructSchema, configTypeSchema, basicSchema, parseJSONTag, isPointer                                 │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ fragment.go         │ fragment, loadFragment                                                                                     │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ docs.go             │ docFinder, findField, clean                                                                                │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ generate_test.go    │ Integration tests for Generate                                                                             │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ schema_test.go      │ Unit tests for schema functions                                                                            │
  ├─────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ cmd/apispec/main.go │ CLI                                                                                                        │
  └─────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  What bothers me

  document.go is doing too much. It has 15 types, a constant, and a function. The OpenAPI document types, Schema, and Ref are all
  crammed together. Schema is arguably more at home near the schema generation code that produces it — but it's also a document type.
  The real issue is that this file is the "everything else" bucket.

  schema.go is doing too much the other way. It has the generation engine (13 functions) plus internal types (deps, schemaEntry). The
  API-mode and config-mode schema generation are interleaved. Two parallel type-switch trees (typeToSchema and configTypeSchema) that
  share basicSchema.

  File names don't tell you the right story. document.go sounds like it's about the Document type, but it's really "all the OpenAPI
  types." schema.go sounds like it's about the Schema type, but it's really the generation engine. config.go is about the generator's
  config, not the OpenAPI config schema feature.

  properties.go has a stale todo. We decided to keep the duplication. Remove the todo.

  fragment type in fragment.go shadows the concept. It's a tiny struct only used by loadFragment. Could just inline it.

  The json tags on all the document types. We only output YAML now. The JSON marshal/unmarshal on OrderedMap and Properties is dead code
   — nothing calls Marshal (JSON) anymore. But they're harmless and might be useful if someone reads the spec back as JSON. Leave them.

  fixture/ and fixture2/ are importable packages. Anyone who go gets apispec pulls them in. They're test-only.

  Proposed reorganization

  Files:

  ┌──────────────────┬───────────────────────────────────────────────────────────────────────────┬──────────────────────────────────┐
  │     New name     │                                 Contents                                  │            Rationale             │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │ openapi.go       │ All OpenAPI types (Document through Components), OpenAPIVersion, Ref()    │ Named for what it models, not    │
  │                  │                                                                           │ one type                         │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │ ordered.go       │ OrderedMap, KV, Properties, Property                                      │ Both are ordered collections     │
  │                  │                                                                           │ with marshal logic — consolidate │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │ generate.go      │ Generate, loadConfig, loadPackages, loadFragment, fragment                │ The pipeline and its I/O helpers │
  │                  │                                                                           │  together                        │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │ config.go        │ Config, TypeRef, Spec                                                     │ Fine as-is                       │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │                  │ API-mode schema generation: schemaEntry, deps, resolveAll, schemaFrom,    │                                  │
  │ schema.go        │ typeToSchema, namedSchema, structSchema, basicSchema, isPointer,          │ Just the API schema engine       │
  │                  │ parseJSONTag                                                              │                                  │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │ config_schema.go │ Config-mode schema generation: configSchemaFrom, configStructSchema,      │ Separate the two modes           │
  │                  │ configTypeSchema                                                          │                                  │
  ├──────────────────┼───────────────────────────────────────────────────────────────────────────┼──────────────────────────────────┤
  │ docs.go          │ docFinder, findField, clean                                               │ Fine as-is                       │
  └──────────────────┴───────────────────────────────────────────────────────────────────────────┴──────────────────────────────────┘

  Test fixtures: Move fixture/ and fixture2/ into testdata/fixture/ and testdata/fixture2/. The go/packages loader can still find them
  by import path — we just need to check that. This keeps them out of the importable API.

  Naming:
  - Config is ambiguous (generator config vs OpenAPI config schema). Consider GenConfig or leave it since it's unexported-ish (only the
  CLI and tests use it).
  - schemaEntry and deps are fine — they're unexported internal types.
