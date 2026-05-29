  Proposal: description: items can be strings or package references

  Schema

  description:
    - "plain paragraph string"                      # literal, as today
    - package: tag                                  # pulls package doc comment
    - package: tag/tagger/objectstore               # works for any spec-reachable pkg
    - package: tag
      type: Widget                                  # pulls a type's doc comment, if ever useful

  Rules:
  - Each item renders as one paragraph in info.description, separated by blank lines (same as now).
  - A string → literal.
  - A map with package: and no type: → extract that package's go/doc.Package.Doc field.
  - A map with package: + type: → extract the named type's doc comment. (Optional; falls out for free if you use go/doc.)
  - Referenced packages do not need to be in specs: — the resolver loads them on demand. Keeps documentation references decoupled from
  path/schema contribution.
  - Unknown fields in the map → parse error (don't silently drop).

  What this looks like for tag-calypso

  title: Bastille Tag Integration API
  description:
    - package: tag
    - >-
      This instance (tag-calypso) sources tags from ClearPass endpoints,
      matching devices by MAC address.

  config:
    name: ServiceConfig
    package: tag/cmd/tag-calypso
    type: config

  specs:
    - package: tag
    - package: github.com/bastille-integration/bfc/webhook
    - package: github.com/bastille-integration/bfc
      types: [Device]
    - package: github.com/clarktrimble/delish/boiler

  And tag.go's existing package doc becomes the canonical shared prose. A small enrichment could add a "Features" paragraph if desired,
  but I'd lean toward letting the prose be developer-audience-appropriate and dropping the marketing-ish bullets.

  tag-phosphorous gets the same - package: tag line + its own differentiator paragraph.

  Implementation sketch in apispec

  Wherever specs[].package is currently resolved to find paths.yaml and load types, a similar loader likely already exists that can give
   you a *doc.Package. The Doc field is the rendered package doc text. For a type: sub-reference, iterate pkg.Types and grab the
  matching one's .Doc.

  The YAML decoder change is small: description becomes []DescriptionItem where DescriptionItem is a custom UnmarshalYAML that accepts
  either a scalar string or a mapping with package/type keys.

  README update

  Add a row to the fields table noting the object form, and a small example in the Usage section.
