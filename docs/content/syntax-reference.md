---
title: "Syntax Reference (v1alpha1)"
weight: 20
---

This page is the authoritative reference for the structure understood by the `yamlfile` BuildKit frontend.

> **v1alpha1 MVP status**: The grammar, parser, and graph are forward-compatible (unknown fields are retained via extensions). However, not everything declared in the grammar is fully wired in the current frontend:
> - `builds:` + `component:target` cross-file orchestration — parsed and graphed but loading/resolution is not yet implemented (only same-file sibling `from:` / `copy.from:` work today).
> - `defaults.platform` and per-target `platform:` — accepted by the parser but ignored (the frontend follows the platform(s) requested via the BuildKit client / `--platform`).
> - Full independent parallel execution inside one build request — the graph helpers (`parallelRoots`, reachable ordering) exist and are tested; the actual ToLLB path is intentionally serial for determinism ("For MVP we build serially in reachable order").
>
> See `pkg/convert/convert.go` (comments around ToLLB and dispatch*) and `pkg/convert/graph.go` for the current scope. Multi-file and platform support are high priority for the next iteration.

## Top Level

```yaml
apiVersion: v1alpha1
kind: Yamlfile                 # optional

defaults:
  platform: linux/amd64

targets:
  # map of name -> TargetSpec

builds:
  # optional multi-file orchestration (see Multi-file section)

secrets:
  # optional top-level secret descriptions (for docs / lint)
```

## TargetSpec

```yaml
my-target:
  from: "golang:1.25"          # image ref, sibling target name, "component:target", or "scratch"
  platform: linux/arm64        # optional per-target platform override
  steps:
    - ...                      # list of Step
```

- `from` can refer to a previously-defined target in the same file (sibling) or a named build from the `builds:` section (for multi-file).
- Targets are built in dependency order (the frontend computes a reachable graph from the requested `--target` or a default).

## Step (discriminated union)

A step has exactly one of `run`, `copy`, `env`, `arg`, or `workdir`.

> **Variable expansion**: Values inside `env.vars`, `arg.vars`, `workdir.path` (standalone or `run.workdir`), and `run.env` support `$VAR` and `${VAR}` references. These are expanded using BuildKit's shell lexer against CLI `--build-arg` values, `arg:` declarations (with defaults), and any `env:` / `run.env` values set earlier in the same target. Sibling `from:` targets inherit their final `ENV` values for expansion. `from:`, `copy.from`, `script:`, and the bodies of `command`/`inline` are left literal (shell handles `$` inside commands at runtime).

A machine-readable **JSON Schema** is automatically generated from the Go types on every `make docs` (and in CI) and published alongside the site:
`schema/v1alpha1.json` (relative to the docs root; e.g. https://builderhub.github.io/Yamlfile/schema/v1alpha1.json).
Point `yaml-language-server` or your editor at it for completion, validation, and hover documentation. The schema is the source of truth and always matches the code you are running.

### run

```yaml
- run:
    command: "go build -o /out/app ."
    # or
    inline: |
      set -e
      go mod download
      go build ...
    # or (the "baked-in" feature)
    script: ./scripts/build.sh   # loaded by the frontend and mounted read-only; no COPY needed in the image

    env:
      CGO_ENABLED: "0"
      GOOS: linux

    workdir: /app/src          # per-run working directory (transient for this exec only)

    secrets:
      - id: mytoken
        env: GITHUB_TOKEN
      - id: netrc
        target: /root/.netrc
        mode: 0600
        optional: true
```

**Quoting tip (common gotcha)**: If your `command:` value contains `key: value` (or looks like a YAML map), quote it or use `inline:` / `script:`. Unquoted `go build -o /out/app .` is fine, but `command: echo foo: bar > /x` can be misparsed by YAML as a map. The examples in this repo always use double quotes or the `|` block form for safety.
```

- `script` paths are resolved relative to the build context. The frontend loads the content (via a restricted local source) and makes it available inside the `RUN` via a temporary scratch mount. The script does **not** end up as a layer in the final image unless you explicitly copy it.
- Secrets are passed using BuildKit's native `--mount=type=secret` mechanism. They are never present in image layers or history.
- `workdir` sets the working directory only for this `RUN` (equivalent to `RUN --workdir=...` or `cd` inside the command). It does not affect subsequent steps or the final image `WORKDIR`. Use the top-level `workdir:` step (below) if you want a persistent change.

### copy

```yaml
- copy:
    from: "previous-target"    # or "component:target", or empty / "context" for the main build context
    src: ["./bin/", "LICENSE"]
    dest: "/app/"
```

`from` can be a sibling target (or a target from another file via the `builds:` mechanism).

### env

Convenience form (equivalent to putting `env:` inside a `run` step, but clearer when you just want to set image config). Values support `${VAR}` / `$VAR` expansion (see the Variable expansion note above).

```yaml
- env:
    vars:
      PATH: "/app/bin:${PATH}"
      FOO: bar
```

### arg

Declares a build-time variable (analogous to `ARG` in a Dockerfile). The value is a default; it can be overridden with `--build-arg NAME=...` at build time. `arg:` values participate in expansion for later steps and are visible (as environment variables) inside subsequent `run` steps' shells, but they do **not** appear in the final image `ENV` unless you also emit them via an `env:` step.

```yaml
- arg:
    vars:
      GO_VERSION: "1.25"   # default; override with --build-arg GO_VERSION=1.24
      VARIANT:             # no default; must be supplied via --build-arg or expands to ""
```

You can reference a build arg (CLI or declared) inside later `env:`, `arg:`, or `workdir:` values:

```yaml
- arg:
    vars:
      APP: myapp
- env:
    vars:
      BIN: /out/${APP}
```

### workdir

Sets the persistent working directory for the remainder of the target (affects the `llb.State` for later steps and the exported image config). This is the moral equivalent of a Dockerfile `WORKDIR` instruction.

```yaml
- workdir:
    path: /app
```

Subsequent `run` steps (and `copy` destinations that are relative) will be relative to this directory. A later `workdir:` overrides it. Per-run overrides are available via `run.workdir` (see above).

## Secrets

There are two places secrets appear:

1. **Top-level `secrets:`** (optional, for documentation / linting only). This is a map of id → description. It does **not** affect build behavior.

```yaml
secrets:
  github_token: "Token used to fetch private Go modules (supply at build time with --secret id=github_token,env=GITHUB_TOKEN)"
  netrc: "netrc for legacy registry auth (file form)"
```

2. **Per-`run` secrets** (the actual mechanism). This is a list. It maps directly to BuildKit's `--mount=type=secret`.

```yaml
- run:
    script: ./scripts/push.sh
    secrets:
      - id: github_token
        env: GITHUB_TOKEN          # injected only for the duration of this run (masked in logs)
      - id: netrc
        target: /root/.netrc       # file form (defaults to /run/secrets/<id> if target omitted)
        mode: 0600
        optional: true
```

Both the file form (`target:`) and env-var form (`env:`) are implemented using `llb.AddSecretWithDest` + the appropriate `SecretAsEnvName` / `SecretFileOpt` options. Secrets are **never** present in final image layers or history when used correctly.

See [Features / Secrets]({{< relref "/features/secrets" >}}) for supply examples (`--secret id=...,env=...` or `src=...`) and the exact option semantics.


## Multi-file / Orchestration (`builds:`) — grammar only in v1alpha1

The grammar and dependency graph prep support declaring other Yamlfiles:

```yaml
builds:
  torch:
    file: torch/Yamlfile
    target: base          # optional; defaults to first or a "default" target inside that file

targets:
  final:
    from: alpine
    steps:
      - copy:
          from: "torch:base"   # "component:target" syntax
          src: ["/opt/torch"]
          dest: "/torch"
```

**Current status (v1alpha1 MVP)**: `builds:` entries and the `component:target` form are parsed and appear in the graph for forward compatibility and tooling, but the frontend does **not** yet load external Yamlfiles or wire cross-file state. Only same-file sibling targets (via `from:` or `copy.from:` using a bare target name) are resolved today.

For now, keep everything in one Yamlfile using multiple named targets + sibling references. Full multi-file support (loading, caching, cross-copy) is planned for the next release.

See the note at the top of this page and `pkg/convert/graph.go` (the `builds` map is stored but not traversed for external loading in ToLLB).


## Platform & Defaults — parsed only in v1alpha1

```yaml
defaults:
  platform: linux/amd64

targets:
  foo:
    from: ...
    platform: linux/arm64   # overrides default for this target
```

**Current status**: The fields exist in the types and survive parsing (for forward-compat and so that linters/docs tools can see the intent). However, the v1alpha1 frontend does not yet read `defaults.platform` or per-target `platform:` — it always uses the platform(s) requested by the BuildKit client (e.g. `docker buildx build --platform linux/amd64,linux/arm64 ...` or the default of the builder).

See the MVP note at the top of the page.

## Extensibility

All top-level objects and step types accept an `Extensions` map (via YAML `<<` or unknown keys) so future fields can be added without breaking existing documents.

Unknown step kinds or required fields that are missing will produce clear errors at parse / conversion time in the current v1alpha1 implementation.
