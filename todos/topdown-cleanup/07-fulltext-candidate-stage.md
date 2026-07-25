# AP7: Fulltext Candidate-ID Stage

## Problem

The text-filtered Omnestum document query must not scan every document and then
evaluate file/content/document text predicates late. MemCP should build and
combine candidate document IDs earlier than traditional DBMS planners can for
this query shape.

## Required Planner Behaviour

Introduce generic candidate stages:

- metadata/document text candidate source
- file/content candidate source
- ACL-visible candidate source
- candidate key, usually document ID
- set intersection / union / dedupe by actual key equality
- projection and expensive joins after candidate reduction

Do not expose FOP or Omnestum names in MemCP engine APIs.

## Omnestum Focus

Search terms can match:

- inventory number
- city / location
- document metadata
- file name
- file content

The planner should produce candidate document IDs from each source, combine
them, then apply ACL/order/projection.

## Tests

Anonymized YAML structure:

- `docs(id, year, location, title, uploaded_at)`
- `files(id, doc_id, name, content)`
- `permissions(user_id, doc_id, can_view)`

Query shape:

- derived dataview wrapper
- ACL EXISTS
- text filter over docs and files
- ORDER BY / LIMIT

Correctness cases:

- duplicate file hits dedupe to one document
- document-only hit
- file-only hit
- ACL denies hit
- no hits

## Done

- Text-filtered Omnestum query does not use a broad document scan as primary
  driver.
- EXPLAIN IR shows candidate stages and document-ID set operations.
