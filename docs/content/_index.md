---
title: "Yamlfile"
bookToC: true
---

**BuildKit frontend** for intuitive, declarative, parallel, multi-file, secret-aware builds that are easier to reason about than large Dockerfiles.

Inspired by the complexity of [coreweave/ml-containers](https://github.com/coreweave/ml-containers) (hundreds of lines of parallel downloaders, secret env mounts, shared scripts via contexts, cross-stage wheel copies, etc.).

## Quick Start

```bash
docker buildx build \
  -f examples/minimal.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=ghcr.io/builderhub/yamlfile:latest \
  --output type=local,dest=/tmp/out \
  .
```

See the [Getting Started]({{< relref "/docs/getting-started" >}}) page and the [Syntax Reference]({{< relref "/docs/syntax-reference" >}}) for details.

## Key Features

- Explicit `targets` (named stages) with `from:` (image or sibling target) → clear dependency graph.
- `run.script: path` — Yamlfile securely loads the script from your build context and mounts it at runtime (no `COPY` layer left in the image).
- `run.inline` / `command` for embedded shell logic.
- Secure secrets: per-run `secrets: [{id, target: /path (file) or env: VAR (env var)}]`.
- Natural parallelism for independent targets. Targets with no dependency relationship between them are not forced into sequential order.
- `apiVersion` + inline extensions so you can use additional fields without breaking your files.
- Multi-file orchestration via `builds:` + `component:target` cross-copy (you can write the syntax today, but cross-file builds are not yet supported at runtime).

## Why Yamlfile instead of (just) a Dockerfile?

Traditional multi-stage Dockerfiles become hard to reason about once you have 10–50 stages, lots of independent downloaders, shared scripts, and secret mounts. Yamlfile makes the graph, the parallelism, the script injection, and the secret mounting first-class and declarative.

Full motivation and comparison: [vs. Dockerfile]({{< relref "/docs/vs-dockerfile" >}}).

## Documentation Sections

- [Getting Started]({{< relref "/docs/getting-started" >}})
- [Syntax Reference]({{< relref "/docs/syntax-reference" >}})
- [Features]({{< relref "/docs/features" >}})
  - Scripts (`run.script`)
  - Secrets (per-run file + env forms)
  - Copy (context + sibling targets)
  - Parallelism & graphs
  - Platforms (`defaults.platform` + per-target)
  - Multi-file (syntax supported, but runtime loading not yet available)
- [Examples]({{< relref "/docs/examples" >}})
- [Development]({{< relref "/docs/development" >}}) (for contributors)

## License

MIT (BuilderHub)
