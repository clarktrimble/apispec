# apispec/static — Build-Time OpenAPI Spec Generator

Generates OpenAPI 3.0.3 specs from Go source at build time using
`go/packages` and `go/types`. No reflection, no runtime cost. Doc comments
become schema descriptions.

## Quick Start

Install:
```
go install github.com/clarktrimble/apispec/cmd/apispec@latest
```

Create `apispec.yaml` in your cmd directory:
```yaml
title: My Service API
description:
  - Service does things.
  - It does them well.

config:
  name: ServiceConfig
  package: myproject/cmd/myservice
  type: config

specs:
  - package: myproject/internal/widgets
    types: [Widget]
  - package: github.com/example/webhooks
    types: [Event]
  - package: github.com/clarktrimble/delish/boiler
```

Touch openapi.yaml in cmd directory too!

Generate from the repo root:
```
apispec gen -c cmd/myservice/apispec.yaml -o cmd/myservice/openapi.yaml
```

Or from the cmd directory using defaults:
```
cd cmd/myservice
apispec gen
```

Or wire it into `go generate` in your cmd's `main.go`:
```go
//go:generate apispec gen
```
Then `go generate ./cmd/myservice` (or `go generate ./...` from the root).

Embed and serve via boiler:
```go
//go:embed openapi.yaml
var spec []byte

// in main():
spec := boiler.SubSpec(spec, version, release, cfg.Url)
rtr := boiler.NewRouter(ctx, cfg, "My Service API", spec, lgr)
```

## Config File

| Field | Description |
|---|---|
| `title` | API title in the OpenAPI info block |
| `description` | List of paragraphs, joined with double newlines |
| `config` | Config schema entry (optional) |
| `config.name` | Schema key in components (defaults to `type` if omitted) |
| `config.package` | Go package containing the config type |
| `config.type` | Go type name to generate config schema from |
| `specs` | List of packages contributing paths and/or types |
| `specs[].package` | Go import path |
| `specs[].types` | List of Go type names for schema generation |

A package with no `types` contributes only paths. A package with no
`paths.yaml` contributes only types.

## How It Works

1. Loads all referenced packages via `go/packages`, which uses the
   project's `go.mod` for module resolution — tagged versions, pseudo-versions,
   and `replace` directives all work as expected
2. For each spec entry, finds `paths.yaml` in the package directory and
   parses it as an OpenAPI paths fragment (with optional `tags:` section)
3. Generates schemas from listed types using `go/types` — struct fields,
   json tags, nested type resolution
4. Chases transitive deps: if Widget has a Part field, Part gets its own
   schema with a `$ref` link
5. Generates config schema separately (different tag conventions, always
   inlined)
6. Adds a shared Error schema
7. Assembles the full document and writes YAML

## Schema Generation

Two modes handle different struct tag conventions:

### API Types (specs.types)

Reads `json`, `desc`, and `example` struct tags. Named struct fields become
`$ref` entries with transitive dependency resolution.

```go
// Widget represents a mechanical component.
type Widget struct {
    Name  string `json:"name" desc:"widget name" example:"sprocket"`
    Count int    `json:"count"`
    Part  *Part  `json:"part,omitempty"`
}
```

Produces:
- `name`: string, description "widget name", example "sprocket", required
- `count`: integer, required
- `part`: `$ref: '#/components/schemas/Part'`, optional

### Config Types (config)

Reads `json`, `desc`, `default`, `required`, and `ignored` struct tags
(envconfig conventions). Always inlines nested structs.

```go
type config struct {
    Version string        `json:"version" ignored:"true"`
    Host    string        `json:"host" desc:"hostname to bind"`
    Port    int           `json:"port" desc:"listen port" required:"true"`
    Timeout time.Duration `json:"timeout" desc:"request timeout" default:"10s"`
}
```

- `version`: skipped (ignored)
- `port`: required (from tag)
- `timeout`: example "10s" (from default tag)

### Doc Comments

Type-level doc comments become the schema description:
```go
// Widget represents a mechanical component.
type Widget struct { ... }
```
Produces `description: Widget represents a mechanical component.`

Field-level doc comments are used as a fallback when no `desc` tag is present:
```go
type Part struct {
    // Human-readable label for the part.
    Label string `json:"label"`
}
```

The `desc` tag always wins when both are present.

## Placeholders

The generated spec includes two placeholders for runtime substitution:

- `${RELEASE}` — in `info.version`, replaced by boiler's `SubSpec` with
  the git tag or branch info
- `${PUBLISHED_URL}` — in `servers[].url`, replaced with the configured
  service URL

## Path Fragments

Each contributing package can include a `paths.yaml` file in its directory.
This is standard OpenAPI paths YAML with an optional `tags:` section:

```yaml
tags:
  - name: widgets
    description: Widget operations

paths:
  /widgets:
    get:
      summary: List widgets
      operationId: listWidgets
      tags:
        - widgets
      responses:
        '200':
          description: Widgets retrieved
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Widget'
```

Fragments are merged in the order listed in the config file. Duplicate
paths are not currently detected at build time (they are in the runtime
approach).
