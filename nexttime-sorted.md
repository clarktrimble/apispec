
  ---
  The Sift

  Do Now (stabilization)

  1. Remove runtime reflection engine — SchemaFrom, ConfigSchema, GenerateSchemas, Deps, Describer, tagReader, apiTags, configTags,
  wellKnown, schemaFromStruct, typeSchema, deref, parseJsonTag in root schema.go. Plus Merge, MergeConfig, SpecFunc, unmarshalFragment,
  mergePaths, mergeTags and the fragment type in merge.go. The tests that exercise them (schema_test.go, merge_test.go,
  document_test.go) go too. This is the biggest cleanup and unblocks everything else — dead code is confusing for anyone who finds this
  module.
  2. Trim document.go — NewDocument, Marshal (JSON), StatusResponse, ErrorResponse, ObjResponse, JsonContent are runtime helpers nobody
  will call anymore. Ref is still used by static/schema.go. Keep the types (Document, Schema, etc.), MarshalYaml, Ref, and
  OpenAPIVersion. Drop the rest.
  3. DONE. Duplicate path detection in Generate — one-liner to match what the runtime did. Worth it for correctness.
  4. DONE (stopped collecting from fragments). Tags dedup in Generate — same, small and correct.
  5. PUNT (working fine). Makefile cleanup — drop the commented-out cross-compile lines, keep linux-amd64 build for the CLI tool. It's a library+tool, not an
   app.
  6. Top-level README — replace with a pointer to static/README.md.

  Do Soon (quality)

  7. fixture/ to testdata/ — it's an importable package that shouldn't be. Move types into static/testdata/fixture/ or make it internal.
   The go/packages loader can still find it.
  8. Root testdata/ — delete once runtime tests are gone.
  9. Properties vs OrderedMap — leave it. Name/Schema reads better than Key/Val, and both are used. The duplication is small and stable.

  Punt (not worth it now)

  10. Error schema in boiler — that's a boiler change, track it there.
  11. Config name defaulting — the name field workaround is fine, document it.
  12. Embedded structs, interface fields, cyclic types, enum support, map key warnings — all real but speculative until you hit them in
  a real service. Add tests when they come up.
  13. $ref with description (3.0 vs 3.1) — punt until you target 3.1 or a validator complains.
  14. --dry-run / --check on CLI — nice for CI but not blocking.
  15. paths.yaml name hardcoding — works fine, solve when someone needs it.
  16. JSON output flag — MarshalYaml is the path, JSON marshal can stay dead.
  17. Build cache invalidation — Makefile gen target already always runs; that's good enough.
  18. omitempty on non-pointer structs, config nested required — subtle edge cases, not blocking real usage.
  19. ${RELEASE} placeholder validation — boiler's problem at runtime, not apispec's.
  20. No tags section in top-level config — add a tags: field to Config when someone needs it.
  21. configSchemaFrom fallback — fine and untestable in practice.
  22. Schema ordering (yaml/v3) — Components.Schemas is map[string]*Schema, so yaml/v3 sorts keys alphabetically. Verified by the fact
  that output is deterministic already.
  23. go/packages error reporting — you already surface pkg.Errors[0], that's enough.
  24. Doc comments on config types — nice but config types use desc tags consistently.
  25. CLI -h / help — flag.ExitOnError already prints usage on bad flags. Minimal but fine.
  26. Multiple mains testing — do when you have a second cmd in fwd.
