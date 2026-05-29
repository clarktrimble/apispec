  The need is real and standards-level. OpenAPI 3 has a top-level tags: array specifically for this:

  tags:
    - name: tags
      description: Device tag management operations
    - name: webhooks
      description: Fusion Center webhook event processing
    - name: operations
      description: Service health and operational controls

  It defines tag metadata (descriptions, external docs) and, critically, imposes render order on renderers like Swagger UI. Without it,
  tools fall back to first-occurrence order in paths: — which is the exact fragile behavior we're hitting. The old hand-written
  docs/openapi.yaml had this block; I flagged it as "decorative" earlier and we agreed to live without, but now we can see it was
  carrying load.

  And yes, it's apispec's concern. apispec already composes the top-level output structure — info, paths, components.schemas — from
  multiple inputs. tags is the same kind of top-level field. The alternatives are all worse:
  - Sub-package-for-ordering hack: hijacks Go package boundaries for layout reasons, bleeds an apispec merge-order concern into the
  codebase structure.
  - Post-processing step: adds a second tool to the pipeline for something apispec should just handle.
  - Ignoring it: forces users to arrange their paths.yaml files around invisible first-occurrence rules.

  Once you're in the business of composing info.description from multiple sources and stitching schemas from types across packages, also
   stitching tags: is well within scope.

  Design sketch

  Source of truth: each contributing paths.yaml can carry a tags: block. The webhook package declares its own "webhooks" tag metadata,
  boiler declares "operations", tag/paths.yaml declares "tags". apispec merges them in spec order, deduping by name (first definition
  wins, or last — TBD).

  # tag/paths.yaml
  tags:
    - name: tags
      description: Device tag management operations
  paths:
    /tags/{tid}: ...

  # webhook package's paths.yaml
  tags:
    - name: webhooks
      description: Fusion Center webhook event processing
  paths:
    /fc-events: ...

  # boiler's paths.yaml
  tags:
    - name: operations
      description: Service health and operational controls
  paths:
    /config: ...

  Merge order = spec order. Whatever specs: list in apispec.yaml says determines the tag array order. Today's order (tag, webhook, bfc,
  boiler) would produce tags → webhooks → operations automatically — no hacks needed.

  Escape hatch (optional): if a cmd wants to override order or metadata for a specific instance, allow tags: at the top of apispec.yaml
  to take precedence / extend.

  Side benefits:
  - Tag metadata lives next to the paths it describes (webhook package owns "webhooks" tag description — no duplication across cmds).
  - Solves the /dryrun ordering problem without creating phantom sub-packages.
  - Brings back the descriptive content from the old docs/openapi.yaml ("Fusion Center webhook event processing" etc.) that was
  otherwise lost.
