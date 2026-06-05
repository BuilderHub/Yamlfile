---
title: "Why yamlfile instead of a plain Dockerfile?"
weight: 60
---



Traditional multi-stage Dockerfiles are extremely powerful but become difficult to reason about at scale:

- Linear file order + implicit dependencies via `FROM foo` / `COPY --from=foo`.
- Lots of boilerplate for "download in parallel for cache" patterns (see the giant `torch/Dockerfile` and `torch-extras/Dockerfile` in coreweave/ml-containers).
- Scripts must be `COPY`ed (or put in heredocs) even when you only want them for the build.
- Secret mounts are verbose and easy to get slightly wrong (leaking into layers or logs).
- Hard to express "these N independent things can be built in parallel and then combined".

yamlfile makes the **graph explicit**:

- `targets` are named first-class citizens.
- `from:` and `copy.from:` (sibling targets today; the `component:target` form for multi-file is in the grammar and graph for future use) declare dependencies.
- The frontend computes reachable targets for a given `--target`, detects cycles, and only builds what is needed.
- `run.script` is a first-class concept (the frontend does the temporary mount for you).
- Secrets have a clean declarative form that maps directly to `type=secret` (file or `env=`).

You still get the full power of BuildKit (caching, multi-platform, provenance, etc.) because yamlfile is "just" another frontend that emits LLB.

See the [Syntax Reference]({{< relref "/syntax-reference" >}}) and the source in `pkg/convert/graph.go` for how the dependency graph is built.
