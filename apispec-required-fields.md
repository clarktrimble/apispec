# apispec Required Fields Note

`apispec` decides OpenAPI `required` fields from Go JSON tags, not from Jed's `Validate` methods.

For API schema types, any exported struct field without `omitempty` and not a pointer is marked required. That means generated schemas can be stricter than runtime validation or common request bodies.

## Current Jed Example

For `jed.Service`, apispec currently marks these fields as required:

- `name`
- `image`
- `ports`
- `labels`
- `volumes`
- `network`
- `resources`
- `about`

Only some of those are truly validated as required by `Service.Validate()`:

- `name`
- `image`
- `network`

The others are required only because their JSON tags do not include `omitempty`:

```go
Ports     map[string]string `json:"ports"`
Labels    map[string]string `json:"labels"`
Volumes   map[string]string `json:"volumes"`
Resources Resources         `json:"resources"`
About     About             `json:"about"`
```

## Why It Matters

OpenAPI clients may believe fields like `ports`, `labels`, `volumes`, `resources`, and `about` must always be present, even if Jed can reasonably handle empty or zero values.

Existing tests currently include empty maps for service PUT bodies, so they remain compatible with the generated schema. But clients that omit these fields may appear invalid according to OpenAPI tooling.

## Possible Fixes

Options if this becomes a problem:

1. Add `omitempty` to fields that are optional in the API contract.
2. Teach `apispec` to read an explicit tag for required/optional API fields.
3. Introduce separate API DTOs whose JSON tags exactly match the HTTP contract.

For now, this is just a schema/API-contract caveat to keep in mind.
