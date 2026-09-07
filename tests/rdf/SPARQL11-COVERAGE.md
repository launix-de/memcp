# SPARQL 1.1 conformance inventory

Copyright (C) 2026 Carl-Philip Hänsch

This directory treats the stable SPARQL 1.1 W3C Recommendations and the RDF
1.1/Turtle term model required by them as the conformance target. SPARQL 1.2 is
still a draft and is deliberately not mixed into this baseline.

The inventory starts new cases as `noncritical: true`. A passing noncritical
case describes capability already present; a failing noncritical case is an
implementation candidate. Once a feature is selected and implemented, its
cases become critical in that feature PR. The algebra, update, and query-form /
dataset suites are critical; the remaining inventory still follows the
noncritical selection workflow.

## Executable coverage

| Recommendation area | Executable suites |
| --- | --- |
| Query syntax, RDF terms, operators, built-ins | `rdf-spec11-terms-expressions.yaml`, `rdf-filter.yaml`, `rdf-bind-functions.yaml` |
| Basic and group graph patterns | `rdf-spec11-algebra-complete.yaml`, `rdf-spec-graph-algebra.yaml` |
| OPTIONAL, UNION, MINUS, EXISTS | `rdf-spec11-algebra-complete.yaml`, `rdf-optional.yaml`, `rdf-union-exists.yaml` |
| Property paths | `rdf-spec11-property-paths-complete.yaml`, `rdf-advanced-sparql.yaml` |
| Assignment and inline data | `rdf-spec11-algebra-complete.yaml`, `rdf-queryplan-coverage.yaml` |
| Aggregates, grouping, HAVING | `rdf-spec11-algebra-complete.yaml`, `rdf-aggregates.yaml` |
| Subqueries | `rdf-spec11-algebra-complete.yaml`, `rdf-advanced-sparql.yaml` |
| RDF datasets and GRAPH | `rdf-spec11-query-forms-datasets.yaml`, `rdf-named-graphs.yaml` |
| Solution modifiers | `rdf-spec11-query-forms-datasets.yaml`, `rdf-spec-bgp-modifiers.yaml` |
| SELECT, CONSTRUCT, ASK, DESCRIBE | `rdf-spec11-query-forms-datasets.yaml`, `rdf-spec-forms-turtle.yaml` |
| Federated SERVICE patterns | `rdf-spec11-query-forms-datasets.yaml` |
| Update and graph management | `rdf-spec11-update-complete.yaml`, `rdf-update.yaml`, `rdf-named-graphs.yaml` |
| JSON, XML, CSV, TSV result formats | `rdf-spec11-protocol-results-entailment.yaml`, `rdf-results-json.yaml` |
| Direct POST media types and protocol errors | `rdf-spec11-protocol-results-entailment.yaml` |
| Simple and RDFS entailment behavior | `rdf-spec11-protocol-results-entailment.yaml` |
| Positive Turtle syntax and RDF term construction | `rdf-spec11-turtle-complete.yaml`, `rdf-turtle-advanced.yaml` |
| Normative query/update rejection rules | `rdf-spec11-negative-syntax.yaml` |

## Harness gaps

The existing YAML runner sends query text to the MemCP RDF endpoint using
POST. It cannot yet express arbitrary HTTP methods or inspect an endpoint that
is not the configured RDF query path. Consequently these protocol surfaces
cannot be tested honestly by this test-only PR:

- SPARQL Protocol GET with the `query` parameter;
- form-encoded POST and protocol dataset parameters;
- Graph Store HTTP Protocol GET, PUT, POST, DELETE, and HEAD;
- discovery through a SPARQL Service Description endpoint;
- protocol-level response status distinctions beyond the runner's generic
  success/error contract.

Those cases require a separate runner-capability PR. They are recorded here so
that "full standard" does not silently imply coverage that the current harness
cannot execute.

## Scope notes

- Entailment regimes are optional capabilities. The cases state observable
  expectations for any regime MemCP later chooses to advertise.
- Remote `SERVICE` success needs a deterministic in-CI peer endpoint. The
  current cases cover parsing, failure, and `SILENT`; a peer fixture belongs in
  the protocol harness extension.
- Negative Turtle parser tests likewise need an expected-error contract on the
  `ttl_data` loader. This PR covers all positive Turtle term and syntax
  families and records the missing negative-loader contract here.
- JSON functions are MemCP extensions and remain covered separately in
  `rdf-json.yaml`; they are not counted as SPARQL conformance.
