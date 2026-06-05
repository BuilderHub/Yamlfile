# yamlfile

**BuildKit frontend** (v1alpha1) for intuitive, declarative, parallel, multi-file, secret-aware builds that are easier to reason about than large Dockerfiles.

Inspired by the complexity of [coreweave/ml-containers](https://github.com/coreweave/ml-containers) (hundreds of lines of parallel downloaders, secret env mounts, shared scripts via contexts, cross-stage wheel copies, etc.).

## Status

First iteration (v1alpha1). Basic targets + run/copy work; full script/secrets/parallel/multi-file + docs coming in this implementation.

## Usage

```bash
# From the BuilderHub *repo root* (the Dockerfile needs sibling buildkit-hive for the replace):
docker buildx build \
  -f yamlfile/examples/minimal.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=ghcr.io/builderhub/yamlfile:dev \
  --output type=local,dest=/tmp/out \
  .
```

(When inside `yamlfile/` use `make docker-build` or adjust context to `..`.)

Or with a remote registry image once published:

```dockerfile
# syntax=ghcr.io/builderhub/yamlfile:v1alpha1
apiVersion: v1alpha1
targets:
  app:
    from: golang:1.25
    ...
```

See `examples/` and the `Makefile` for dev commands. Design notes are in the source (graph + convert) and this README.

## Key Features (Goals)

- Explicit `targets` (named, not linear stages) with `from:` (image or sibling) → clear DAG.
- `steps:` list: `run`, `copy`, `env`, etc.
- `run.script: path/to/script.sh` — frontend loads + securely mounts (no COPY layer needed).
- `run.inline: |` or `command:` for baked-in shell logic.
- Secure secrets: per-run `secrets: [{id, target: /path or env: VAR}]` (file or env form; matches `--mount=type=secret` + `,env=` from ml-containers).
- Parallel execution of independent targets (errgroup in frontend + natural LLB DAG).
- Single top-level Yamlfile can reference multiple component files (`file:` or `builds:`) that each build their target(s); cross-`copy` artifacts between them.
- `apiVersion: v1alpha1` + extension maps for safe evolution (no breaks on additive features).
- Reuses dockerui + llb + gateway so it supports build args, platforms, named contexts, cache, etc. natively.

## Why not (just) Dockerfile?

See the pain points in `torch/Dockerfile` and `torch-extras/Dockerfile` in ml-containers: sequential downloader stages meant to be parallel, repeated secret mount boilerplate, heredoc + context copies for every script, hard-to-visualize dep graph for 10+ stages.

yamlfile makes the graph, the parallelism, the script injection, and the secret mounting first-class and declarative in YAML.

## Development

```bash
nix develop   # or from root
make test
docker buildx build -f cmd/yamlfile-frontend/Dockerfile -t localhost:5000/yamlfile:dev --load .
docker buildx build -f examples/minimal.Yamlfile --build-arg BUILDKIT_SYNTAX=localhost:5000/yamlfile:dev --output type=local,dest=/tmp/out .
cat /tmp/out/msg.txt
```

See the source (especially `pkg/convert/`) and the verification steps in the implementing PR/session for design rationale.

## License

MIT (BuilderHub)
