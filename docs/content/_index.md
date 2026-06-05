---
title: "yamlfile"
weight: 1
---

# yamlfile

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

See the [Getting Started](/getting-started) page and the [Syntax Reference](/syntax-reference) for details.

## Key Features

- Explicit `targets` (named stages) with `from:` (image or sibling target) → clear dependency graph.
- `run.script: path` — the frontend securely loads the script from your build context and mounts it at runtime (no `COPY` layer left in the image).
- `run.inline` / `command` for embedded shell logic.
- Secure secrets: per-run `secrets: [{id, target: /path (file) or env: VAR (env var)}]`.
- Natural parallelism when independent targets have no data dependencies.
- Single top-level Yamlfile can orchestrate multiple component files (`builds:` / `file:` refs) with cross-`copy`.
- `apiVersion: v1alpha1` + inline extensions for forward-compatible evolution.

## Why yamlfile instead of (just) a Dockerfile?

Traditional multi-stage Dockerfiles become hard to reason about once you have 10–50 stages, lots of independent downloaders, shared scripts, and secret mounts. yamlfile makes the graph, the parallelism, the script injection, and the secret mounting first-class and declarative.

Full motivation and comparison: [vs. Dockerfile](/vs-dockerfile).

## Documentation Sections

- [Getting Started](/getting-started)
- [Syntax Reference](/syntax-reference)
- [Features](/features) (scripts, secrets, copy & multi-target, parallelism)
- [Examples](/examples)
- [Development](/development) (for contributors)

## License

MIT (BuilderHub)
