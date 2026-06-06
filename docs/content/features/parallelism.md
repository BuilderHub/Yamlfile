---
title: "Parallelism & Dependency Graphs"
weight: 30
---

When you define multiple top-level `targets` that do not depend on each other (no `from:` or `copy.from:` chain between them), the frontend can (and does) prepare them without artificial serialization.

The frontend builds a dependency map, computes the reachable set for the requested target, and provides parallel-root and reachable-ordering helpers.

The actual execution parallelism comes from two places:
1. The frontend constructing independent sub-DAGs (potentially concurrently).
2. BuildKit's own solver running independent operations in parallel.

Today the build within a single requested target is still largely serial (for simplicity and determinism), but independent top-level targets are naturally parallelizable by the graph.

Future iterations may add explicit concurrent execution of independent roots inside one build request.

See the "MVP status" note in the [Syntax Reference]({{< relref "/syntax-reference" >}}) for the current serial execution reality inside a single requested target (the graph prep is honest; the ToLLB path is intentionally serial today).
