---
title: "Parallelism & Dependency Graphs"
weight: 30
aliases:
  - /features/parallelism/
---

When you define multiple top-level `targets` that have no dependency on each other (no `from:` or `copy.from:` relationship), Yamlfile prepares them without forcing unnecessary ordering.

### What this means for your builds

- Targets that don't depend on one another won't block each other.
- You don't have to do anything special. Just write your targets with clear dependencies using `from:` and `copy.from:`. Yamlfile figures out what can safely run in parallel.
- BuildKit's execution engine also runs independent operations concurrently and takes advantage of caching.

If you request a specific target (with `--target`), only the targets it actually needs will be built. Independent "sibling" targets that aren't required are left alone.

This gives you natural parallelism for things like preparing multiple base stages or running unrelated setup steps, without the linear ordering problems common in large Dockerfiles.

See the status box in the [Syntax Reference]({{< relref "/docs/syntax-reference" >}}) for current limitations. Implementation details are in the [Development]({{< relref "/docs/development" >}}) page.
