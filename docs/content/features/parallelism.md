---
title: "Parallelism & Dependency Graphs"
weight: 30
---

# Parallelism & Dependency Graphs

When you define multiple top-level `targets` that do not depend on each other (no `from:` or `copy.from:` chain between them), the frontend can (and does) prepare them without artificial serialization.

The implementation in `pkg/convert/graph.go` builds a dependency map, computes the reachable set for the requested target, and provides `parallelRoots` / reachable ordering helpers.

The actual execution parallelism comes from two places:
1. The frontend constructing independent sub-DAGs (potentially concurrently).
2. BuildKit's own solver running independent operations in parallel.

In the v1alpha1 implementation the build within a single requested target is still largely serial (for simplicity and determinism), but independent top-level targets are naturally parallelizable by the graph.

Future iterations may add explicit `errgroup` construction for independent roots inside one build request.
