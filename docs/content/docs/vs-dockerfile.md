---
title: "Why Yamlfile instead of a plain Dockerfile?"
weight: 60
aliases:
  - /vs-dockerfile/
---



Traditional multi-stage Dockerfiles are extremely powerful but become difficult to reason about at scale:

- Linear file order + implicit dependencies via `FROM foo` / `COPY --from=foo`.
- Lots of boilerplate for "download in parallel for cache" patterns (see the giant `torch/Dockerfile` and `torch-extras/Dockerfile` in coreweave/ml-containers).
- Scripts must be `COPY`ed (or put in heredocs) even when you only want them for the build.
- Secret mounts are verbose and easy to get slightly wrong (leaking into layers or logs).
- Hard to express "these N independent things can be built in parallel and then combined".

Yamlfile makes the **graph explicit**:

- `targets` are named first-class citizens.
- `from:` and `copy.from:` (sibling targets today; the `component:target` form for multi-file references is accepted but not yet supported at runtime) declare dependencies.
- Only the targets you actually need are built.
- `run.script` lets you run a script from your build context without leaving a copy of it in the final image.
- Secrets have a clean declarative form that maps directly to `type=secret` (file or `env=`).

You still get the full power of BuildKit (caching, multi-platform, provenance, etc.).

See the [Syntax Reference]({{< relref "/docs/syntax-reference" >}}) for how targets, dependencies, and steps are declared.
