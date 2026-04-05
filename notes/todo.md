# Todo

## Properties consolidation

Properties predates OrderedMap and duplicates the same pattern with
different field names (Name/Schema vs Key/Val). Could unify, but it
touches a lot of call sites.

## Components.Schemas ordering

Still map[string]*Schema — renders in random order at the bottom of
the page. Less visible than paths/responses but still nondeterministic.

## OpenAPI validation

No validation step yet. Could validate the generated document against
the OpenAPI 3.0 spec — either offline via a linter or as a test assertion.

## Boiler Docs

Mention "touch openapi.yaml" there?

## Consumers and Contributors

Update everything this thing is baked!!
