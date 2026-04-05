# apispec

Generates OpenAPI 3.0.3 documents from Go source at build time.

A YAML config file points at packages — apispec reads their types for schema
generation and their `paths.yaml` files for route definitions, then assembles
a complete spec. Struct tags and doc comments drive the output; no hand-written
schemas needed.

The generated spec includes two placeholders for runtime substitution:
- `${RELEASE}` — to be replaced with the git tag or branch info
- `${PUBLISHED_URL}` — to be replaced with deployed URL for requests from API doc

## Install

```
go install github.com/clarktrimble/apispec/cmd/apispec@latest
```

## Usage

Create `apispec.yaml` in your cmd directory:

```yaml
title: Widget Service API
description:
  - A service for managing widgets.
  - It manages them well.

config:
  package: myproject/cmd/myservice
  type: ServerConfig

specs:
  - package: myproject/internal/widgets
    types: [Widget]
  - package: github.com/example/webhooks
    types: [Event]
```

The config section generates schema from the service's configuration type.
The specs section loads `paths.yaml` and generates supporting schema from any
types listed.

Generate an OpenAPI spec:

```
apispec gen [-c apispec.yaml] [-o openapi.yaml]
```

Or via `go generate` in main.go:

```go
//go:generate apispec gen
```

## apispec.yaml Fields

| Field | Description |
|---|---|
| `title` | API title |
| `description` | List of paragraphs for the API description |
| `config` | Config schema entry (optional) |
| `config.name` | Config schema name, match $ref for path returning config |
| `config.package` | Go package containing the config type |
| `config.type` | Go type from which to generate config schema |
| `specs` | Contributors to path and schema |
| `specs[].package` | Go import path |
| `specs[].types` | List of Go types from which to generate schema, match path $refs |

A package with no `types` contributes only paths.
A package with no `paths.yaml` in its root contributes only types.

## Path Fragments

Each contributing package can include a `paths.yaml`.

```yaml
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

Fragments are merged in config file order.
Referenced types should be included using the package that defines them.

## Schema Generation

Two modes handle different struct tag conventions.

### API Types

Reads `json`, `desc`, and `example` struct tags:

```go
// Widget represents a mechanical component.
type Widget struct {
    Name  string            `json:"name" desc:"widget name" example:"sprocket"`
    Count int               `json:"count"`
    Part  *Part             `json:"part,omitempty"`
    Tags  map[string]string `json:"tags,omitempty"`
}
```

Produces:

```yaml
Widget:
  type: object
  description: Widget represents a mechanical component.
  properties:
    name:
      type: string
      description: widget name
      example: sprocket
    count:
      type: integer
    part:
      $ref: '#/components/schemas/Part'
    tags:
      type: object
      additionalProperties:
        type: string
  required:
    - name
    - count
```

- Fields without `omitempty` and not pointers are marked required
- `desc` tag sets the field description; doc comments are the fallback
- Type-level doc comments become the schema description
- Named struct fields (like `Part`) get their own schema via `$ref`,
  and their dependencies are chased transitively
- Map fields generate `additionalProperties` from the value type

### Config Type

Reads existing envconfig tags — `json`, `desc`, `default`, `required`,
and `ignored`. Nested structs are always inlined (no `$ref`).

```go
type ServerConfig struct {
    Version string        `json:"version" ignored:"true"`
    Host    string        `json:"host" desc:"hostname or ip to bind"`
    Port    int           `json:"port" desc:"port to listen on" required:"true"`
    Timeout time.Duration `json:"timeout" desc:"request timeout" default:"10s"`
}
```

Produces:

```yaml
ServerConfig:
  type: object
  properties:
    host:
      type: string
      description: hostname or ip to bind
    port:
      type: integer
      description: port to listen on
    timeout:
      type: string
      description: request timeout
      example: 10s
  required:
    - port
```

- `ignored:"true"` fields are skipped entirely
- `required:"true"` marks the field required
- `default` tag value becomes the example

### Well-Known Types

`time.Time` maps to `string` with `format: date-time`.
`time.Duration` maps to `string`.
`json.RawMessage` maps to `object` with description "raw JSON".

### Error Type

A shared `Error` schema is built-in (for now?):

```yaml
Error:
  type: object
  properties:
    error:
      type: string
      description: Error message
```

