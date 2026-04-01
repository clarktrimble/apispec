
  Clean up the old runtime code:
  - The reflection engine in schema.go (SchemaFrom, ConfigSchema, tagReader, etc.) — does it still have consumers, or can it go?
  - Merge, MergeConfig, SpecFunc — same question. Once fwd switches to static generation + embedded spec, these are dead code
  - Describer interface — replaced by doc comments
  - The decision: gut the runtime stuff entirely, or keep it for a deprecation period?

  Consolidate the document types:
  - document.go, ordered.go, properties.go — these stay, they're used by both approaches
  - The Properties vs OrderedMap duplication — still worth considering, or leave it since "Name/Schema reads better"?
  - The forcetypeassert linter fixes in ordered.go and properties.go are still pending

  Project hygiene:
  - Makefile — works now but still has app-oriented cruft (cross-compile, GOOS/GOARCH). Trim for a library+tool module?
  - Top-level README — currently full of session notes. Replace with something that points to static/README.md as the primary approach
  - fixture/ package — is it OK shipping as a real importable package, or should it move somewhere less visible?
  - The testdata/ in root vs static/testdata/ — the root one is for the old runtime tests

  Error schema:
  - Still hardcoded in both Merge and Generate. The todo says move to boiler/respond as a real type. That's a boiler change but worth
  tracking.

  Anda:
  - Config name defaulting — the name field on TypeRef falls back to the Go type name, but for unexported types like config that
  produces a lowercase schema key. Should it be required instead of optional?
  - Duplicate path detection — the todo we added in generate.go. The runtime version errors, the static version silently appends.
  - Tags dedup — runtime mergeTags deduplicates, static Generate just appends. Same gap.
  - JSON output — Generate only writes YAML. Might want a -format json flag on the CLI, since Marshal (JSON) already exists on the
  document types.
  - go generate in Makefile — the gen target runs go generate ./.... Once a cmd has //go:generate apispec gen, this ties into make check
   naturally. Worth noting.

  Another thing:
  - Testing against fwd's actual output — compare the static-generated spec against what the runtime Merge produces for the same
  service. A snapshot diff would validate the migration.
  - Multiple mains — we designed for this but only tested with fwd-phosphorous. Try another cmd in fwd to exercise the "one config per
  main" pattern.
  - Doc comments on config types — currently config schema doesn't use docFinder at all. Could be a nice fallback for fields that lack
  desc tags, same as API types.
  - The version subcommand — apispec version works but there's no -h / help output. Minimal but noticeable.

  Still coming!:
  - Embedded structs — Go embedded fields (anonymous structs) promote their fields. Haven't tested whether go/types handles this
  correctly through our schema generation, or if we need to walk promoted fields explicitly.
  - Interface fields — any / interface{} fields map to type: object but we haven't tested that path in static mode.
  - Enum support — string constants with a common type (e.g. type Status string; const Active Status = "active") could populate enum in
  schemas. go/types can see constants — this is something reflection couldn't do.
  - Map key types — map[string]string works, but what about map[SomeType]string? OpenAPI only supports string keys. Might want a
  warning.
  - Cyclic types — a struct that references itself (e.g. tree nodes). The $ref approach handles this naturally, but resolveAll could
  loop forever if there's a cycle in deps. Worth a test.
  - go/packages error reporting — if a listed package doesn't compile, we get a terse error from packages.Load. Could surface that more
  helpfully.
  - Schema ordering in output — we confirmed encoding/json sorts map keys but never actually verified yaml/v3 does. Still unverified
  from earlier in this session.
  - The paths.yaml name is hardcoded — every package must use exactly paths.yaml. If a package wants a different name (or has multiple
  fragments, e.g. one per route group), there's no way to specify that. Could be a field on Spec — paths: custom_paths.yaml — defaulting
   to paths.yaml

Still more:
  - Build cache invalidation — go generate doesn't know about paths.yaml or apispec.yaml as inputs. If you change a path fragment but no
   Go files, go generate won't re-run. Might need a Makefile target that's smarter, or just always regenerate.
  - Schema $ref with description — the fwd output had $ref alongside description on Widget.part. That's technically invalid in OpenAPI
  3.0 (sibling properties next to $ref are ignored). 3.1 allows it. Worth deciding: drop the description on $ref fields, or accept the
  3.1-ism.
  - omitempty on non-pointer structs — if a field is Widget json:"widget,omitempty" (not a pointer but has omitempty), we mark it
  not-required. But OpenAPI "required" and "nullable" are different concerns. Might be subtly wrong in edge cases.
  - Config schema nested required — config mode reads required:"true" tags, but only on direct fields. If a nested config struct has
  required fields, those come through. But the nesting struct itself is never marked required (it's always a pointer). Is that right?
  - No --dry-run on the CLI — would be useful for CI to validate the config without writing a file. Or --check that fails if the output
  would differ from what's already on disk (like gofmt -l).

Finally:
 - The info.version placeholder shows as ${RELEASE} in the YAML — if someone validates the spec before boiler substitution, it'll have
  literal ${RELEASE} as the version string, which is technically valid but looks broken. Same for ${PUBLISHED_URL} failing URL
  validation.
  - No tags section in the top-level config — if you want global tags that aren't contributed by any package's paths.yaml, there's no
  way to declare them. The runtime approach had them on the base document.
  - configSchemaFrom silently falls back to API-mode typeToSchema — if the config type isn't a struct (e.g. a type alias), it drops into
   the API path with nil deps. Probably fine but untested.
  - The fixture package ships in the module — anyone who go gets apispec pulls in static/fixture with its test types. It's harmless but
  not clean. Could be internal or testdata if we solve the go/packages loading issue.
