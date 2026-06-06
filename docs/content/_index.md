---
title: "Yamlfile"
weight: 1
---

**BuildKit frontend** (v1alpha1) for intuitive, declarative, parallel, multi-file, secret-aware builds that are easier to reason about than large Dockerfiles.

Inspired by the complexity of [coreweave/ml-containers](https://github.com/coreweave/ml-containers) (hundreds of lines of parallel downloaders, secret env mounts, shared scripts via contexts, cross-stage wheel copies, etc.).

## Quick Start

```bash
docker buildx build \
  -f examples/minimal.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=ghcr.io/builderhub/yamlfile:latest \
  --output type=local,dest=/tmp/out \
  .
```

See the [Getting Started]({{< relref "/getting-started" >}}) page and the [Syntax Reference]({{< relref "/syntax-reference" >}}) for details.

## Key Features

- Explicit `targets` (named stages) with `from:` (image or sibling target) → clear dependency graph.
- `run.script: path` — the frontend securely loads the script from your build context and mounts it at runtime (no `COPY` layer left in the image).
- `run.inline` / `command` for embedded shell logic.
- Secure secrets: per-run `secrets: [{id, target: /path (file) or env: VAR (env var)}]`.
- Natural parallelism when independent targets have no data dependencies (graph prep + helpers exist; execution within one request is serial in v1alpha1 for determinism).
- `apiVersion: v1alpha1` + inline extensions for forward-compatible evolution.
- (Planned) Multi-file orchestration via `builds:` + `component:target` cross-copy (grammar + graph support present; runtime loading not yet wired).

## Why Yamlfile instead of (just) a Dockerfile?

Traditional multi-stage Dockerfiles become hard to reason about once you have 10–50 stages, lots of independent downloaders, shared scripts, and secret mounts. Yamlfile makes the graph, the parallelism, the script injection, and the secret mounting first-class and declarative.

Full motivation and comparison: [vs. Dockerfile]({{< relref "/vs-dockerfile" >}}).

## Documentation Sections

- [Getting Started]({{< relref "/getting-started" >}})
- [Syntax Reference]({{< relref "/syntax-reference" >}})
- [Features]({{< relref "/features" >}}) (scripts, secrets, copy, parallelism; multi-file planned)
- [Examples]({{< relref "/examples" >}})
- [Development]({{< relref "/development" >}}) (for contributors)

## License

MIT (BuilderHub)
