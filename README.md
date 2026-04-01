# apispec — Composable OpenAPI Spec Builder

100% vibe coded and it shows!

Todo: move testdata to test/data and fuss over quality and organization
Todo: look at unit coverage and expand as needed
Todo: add an example usage here
Todo: add a rationale section here
Todo: seriously consider not doing this sort of thing at runtime
Todo: fresh stoplight or maybe an alternative
Todo: cleanup the rest of this doc

## Stoplight Specifics

### Tags

Yaml might have something like:
```
tags:
  - name: enrichment
    description: Threat enrichment cache management
paths:
  /enrichment/cache:
    get:
      summary: Get enrichment cache
      description: Return the current threat enrichment cache (category and rule lookups)
      operationId: getEnrichmentCache
      tags:
        - enrichment
      responses:
        ...
```
But Stoplight ignores the tag desc and the tag in the path is enough,
so we drop the tags section altogether.

### Info

Stoplight ignores contact and license so we've dropped them.
```
  contact:
    name: Bastille Integration Team
  license:
    name: Proprietary
```

## Approach

OpenAPI specs for our services are assembled from fragments contributed by
dependencies. Each dependency (webhook, boiler, stats) owns its own routes
and ships a spec fragment. A central merge step combines them into a complete
OpenAPI 3.0.3 document.

The key insight: routes are defined in dependencies injected via `main.go`,
so the spec should follow the same pattern. Each dependency contributes what
it knows, and the assembler merges it.

### Schema Generation

Schemas are generated from Go types via reflection rather than hand-written.
Two modes handle different tag conventions:

- **`SchemaFrom(v)`** — for API types (request/response). Reads `json`, `desc`,
  and `example` struct tags. Named struct fields become `$ref` entries with
  transitive dependency resolution.
- **`ConfigSchema(v)`** — for config structs. Reads `json`, `desc`, `default`,
  `required`, and `ignored` tags already present for envconfig. Always inlines
  nested structs.

Types can implement `Description() string` to provide a top-level schema
description (since Go doc comments aren't available via reflection).

### Contributor Contract

Dependencies export a `SpecFunc`:

```go
func ApiSpec() ([]byte, map[string]any)
```

- `[]byte` — embedded YAML containing paths (with a top-level `paths:` key)
- `map[string]any` — type map: schema name to Go zero value for generation

No apispec import required in the dependency. Example from `bfc/webhook`:

```go
//go:embed paths.yaml
var pathsYaml []byte

func ApiSpec() ([]byte, map[string]any) {
    return pathsYaml, map[string]any{"Event": bfc.Event{}}
}
```

### Assembly

`apispec.Merge` takes a base document and any number of `SpecFunc`
contributors. It unmarshals each YAML fragment, generates schemas from the
type maps (including transitive `$ref` deps), and adds a shared `Error`
schema.

```go
doc, err := apispec.Merge(base, webhook.ApiSpec)
spec, err := apispec.Marshal(doc)
// pass spec to boiler.NewRouter
```

## Current State

Working prototype in `fwd/apispec/`. Integrated into `fwd-phosphorous/main.go`
and rendering in the browser via boiler.

### What works

- Schema generation from Go structs with ordered field output
- Two-mode tag reading: api types vs config structs
- `$ref` extraction for named struct types with recursive dep resolution
- `Description()` interface for type-level schema descriptions
- YAML path fragments unmarshal into typed Go structs
- `Merge` assembles a complete OpenAPI doc from a base + contributors
- `Marshal` / `MarshalYaml` for output
- Custom `Properties` type preserving field order through JSON and YAML
- Webhook spec fragment consumed from real `bfc/webhook` package

### Files

- `schema.go` — Schema type, `SchemaFrom`, `ConfigSchema`, reflection engine
- `document.go` — OpenAPI document types, helpers (`Ref`, `JsonContent`, etc.)
- `properties.go` — ordered Properties type with custom JSON/YAML marshaling
- `merge.go` — `Merge`, `Marshal`, `MarshalYaml`
- `ordered.go` — generic `OrderedMap[V]` with JSON/YAML marshal preserving order
- `*_test.go` — tests against realistic types (Event, Observation, nested configs)

## What Remains

### Promote to boiler/delish

The apispec package should move to `delish` (alongside boiler) so all
services can use it. Boiler then owns the merge and serves the spec. The
`Merge` call and base document construction move into boiler's `NewRouter`
or a companion function.

Or, consider an apispec module please.

### Boiler and stats contribute their own specs

- **boiler**: `/config`, `/monitor`, `/log`, `/log/{level}` as embedded YAML
  paths. Config schema generated from the `cfg any` boiler already receives.
- **stats**: `/stats` as embedded YAML path.

Both follow the same `SpecFunc` pattern as webhook.

### Error schema ownership

Currently hard-coded in `Merge`. Should live in boiler (or delish/respond)
since that's where the error response convention is defined. Consider whether
a Go struct for error responses would be valuable.

### Enrich struct tags in bfc

`bfc.Event` now has `desc`, `example` tags and a `Description()` method.
The same treatment for `Observation`, `Emitter`, `Area`, `DeviceInfo`,
`Network`, and other types used in specs would make generated schemas
match the hand-written ones from the tag project.

### Path ordering — done, see 2nd session

`Paths` is currently `map[string]*PathItem` — path order in the output is
not guaranteed. Same ordered-output treatment as `Properties` may be needed
if path order matters in the rendered spec.

### Remaining fwd routes — done, see 2nd session

The enrichment routes (`GET /enrichment/cache`, `POST /enrichment/cache`)
from `fwd.Start` need a spec fragment — either YAML in fwd or contributed
by the fwd service itself.

### Apply to tag project

Once apispec lives in delish, the tag project can adopt the same pattern,
replacing its hand-written `docs/openapi.yaml` with assembled fragments.
The tag-specific routes become a small YAML fragment; webhook, boiler, and
config schemas come for free.

### Top-level document fields

The base document in `main.go` needs attention for:

- **`info.description`** — multi-line service description, not yet set
- **`servers`** — currently hardcoded URL; tag project uses `${PUBLISHED_URL}`
  placeholder via `boiler.SubSpec`. Need a substitution or config-driven
  approach.

### Tags merging — done, see 2nd session

Contributors could declare their `tags:` (grouping metadata) alongside
paths. `Merge` would collect and dedup them, rather than requiring the
base document to know all tag groups upfront.

### Response ordering — done, see 2nd session

`Responses` is `map[string]*Response` — status codes render in arbitrary
order (e.g. "500" before "200"). Same ordered-type treatment as Properties
may be needed.

### Testing / validation

No OpenAPI validation step yet. Could validate the merged document against
the OpenAPI 3.0 spec — either offline via a linter or as a test assertion.

**Snapshot test** — marshal the merged doc to a golden `.json` file checked
into the repo. The test marshals, compares against the golden file, and
fails on any diff. This catches unintended changes: field ordering shifts,
missing `omitempty`, schema deps appearing or disappearing — things that
are technically correct but surprising in rendered docs. Update the golden
file intentionally (`go test -update`) when the output changes on purpose.

Snapshot idea brings up: should allthis be done at build time??

## Notes From 2nd Session

  Correctness
  - Duplicate paths error on merge
  - Schema name collisions error (different types, same name)
  - Shared maps cloned in Merge to avoid caller mutation

  Enrichment spec
  - fwd.ApiSpec() — embedded YAML paths + EnrichmentCache type
  - EnrichmentCache exported (was mappings)
  - Wired into main.go as second contributor

  Document helpers
  - ObjResponse — mirrors respond.WriteObjects envelope pattern
  - additionalProperties on maps — map[string]string renders properly

  Ordering
  - Generic OrderedMap[V] with JSON/YAML marshal preserving insertion order
  - Paths and Responses now ordered (was random map iteration)

  Tags
  - Contributors declare tags in their YAML fragments
  - Merge collects and dedupes (first wins, call order preserved)
  - Base document no longer owns contributor tags

### Learned along the way

  `ObjResponse` helper — mirrors `respond.WriteObjects` envelope pattern
  (`{"key": value}`). Use alongside `StatusResponse` (for `rp.Ok`) and
  `ErrorResponse` (for `rp.NotOk`). Built but not exercised yet — will
  matter when boiler routes get their spec.

  `additionalProperties` — `map[K]V` now generates value type schema
  (e.g. `map[string]string` → `additionalProperties: {type: string}`).
  `Schema.AdditionalProperties` changed from `*bool` to `*Schema`.

  Stoplight Elements quirks — tag descriptions are not rendered in the
  sidebar or section headers. Only the first tag per operation is used
  (stoplightio/elements#1776). No tag group support (#2580).

### Consider for next time

  Promote to module — feels close. The SpecFunc contract is stable, the merge semantics are solid, ordering works. The main thing
  blocking downstream adoption (boiler, stats, tag) is apispec living inside fwd. This is probably the highest-leverage next step.

  Properties consolidation — Properties predates OrderedMap and duplicates the same pattern with different field names (Name/Schema vs
  Key/Val). Could unify, but it touches a lot of call sites. Might be natural to do during the module extraction.

  Components.Schemas ordering — still map[string]*Schema. Less visible than paths/responses but still renders in a random order at the
  bottom of the page.

  Boiler spec fragment — once apispec is a module, boiler can own /config, /monitor, /log, /log/{level} as a real SpecFunc. That's the
  biggest payoff of the module move.
