---
title: "Syntax Reference"
weight: 20
bookToC: true
aliases:
  - /syntax-reference/
---

This page is the authoritative reference for the structure understood by the `Yamlfile` BuildKit frontend.

> **Status**: The format is designed to be forward-compatible (unknown fields are retained via extensions). Most of the documented features work when you build with Yamlfile:
>
> - [x] `defaults.platform` and per-target `platform:` are honored (target.platform > defaults.platform > client `--platform` / dockerui). See the Platform section below.
> - [ ] `builds:` + `component:target` cross-file refs — you can write the syntax, and it will give you a clear error if you try to use it. Actually loading and using targets from other Yamlfiles is not yet supported.
> - [x] Dependency graphs (including cycle detection) are fully supported. Independent targets are not artificially serialized. See [Parallelism & Graphs]({{< relref "/docs/features/parallelism" >}}) for details.
>
> See [Development]({{< relref "/docs/development" >}}) for scope. Multi-file runtime remains the main pending item.

## Top Level

```yaml
apiVersion: v1alpha1

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
- Targets are built in the order required by their dependencies. Only targets needed for the one you requested (via `--target` or a default) will be built.

## Step (discriminated union)

A step must specify exactly one of the following (and only one):

- `run`
- `copy`
- `env`
- `arg`
- `workdir`
- `label`
- `entrypoint`

> **Variable expansion**: Values inside `env.vars`, `arg.vars`, `label.vars`, `workdir.path` (standalone or `run.workdir`), and `run.env` support `$VAR` and `${VAR}` references. These are expanded using BuildKit's shell lexer against CLI `--build-arg` values, `arg:` declarations (with defaults), and any `env:` / `run.env` values set earlier in the same target. Sibling `from:` targets inherit their final `ENV` values for expansion. `from:`, `copy.from`, `script:`, and the bodies of `command`/`inline` are left literal (shell handles `$` inside commands at runtime).

A machine-readable [JSON Schema](https://builderhub.github.io/Yamlfile/schema/v1alpha1.json) is published alongside the docs site at `schema/v1alpha1.json`. Point `yaml-language-server` or your editor at it for completion, validation, and hover documentation. The schema is the source of truth for the supported Yamlfile surface.

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
    script: ./scripts/build.sh   # loaded by Yamlfile and mounted read-only; no COPY needed in the image

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

**Quoting tip (common gotcha)**: If your `command:` value contains `key: value` (or looks like a YAML map), quote it or use `inline:` / `script:`. Unquoted `go build -o /out/app .` is fine, but `command: echo foo: bar > /x` can be misparsed by YAML as a map. Prefer double quotes or the `|` block form for safety.

- `script` paths are resolved relative to the build context. Yamlfile loads the content (using a restricted source) and makes it available inside the `RUN` via a temporary scratch mount. The script does **not** end up as a layer in the final image unless you explicitly copy it.
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

Sets the persistent working directory for the remainder of the target (affects subsequent steps and the exported image config). This is the moral equivalent of a Dockerfile `WORKDIR` instruction.

```yaml
- workdir:
    path: /app
```

Subsequent `run` steps (and `copy` destinations that are relative) will be relative to this directory. A later `workdir:` overrides it. Per-run overrides are available via `run.workdir` (see above).

### label

Sets OCI image config labels (the equivalent of Dockerfile `LABEL`). Values support `${VAR}` / `$VAR` expansion.

```yaml
- label:
    vars:
      org.opencontainers.image.title: Yamlfile
      moby.buildkit.frontend.network.none: "true"
```

### entrypoint

Sets the image config entrypoint (Dockerfile `ENTRYPOINT`). It accepts the same invocation fields as a `run` step:

- `command`
- `inline`
- `script`

The step emits image metadata rather than executing a build step.

```yaml
- entrypoint:
    command: "/bin/yamlfile-frontend"
```

Semantics differ from `run` for `command`:

- **`entrypoint.command`**: exec-form — the string is shlex-split into argv (maps to `ENTRYPOINT ["/bin/foo"]`). No `/bin/sh -c` wrapper.
- **`entrypoint.inline`**: shell-form — prepends the image shell (e.g. `/bin/sh -c`), like Dockerfile shell `ENTRYPOINT`.
- **`entrypoint.script`**: the script content is loaded from the build context at build time and baked into the final image as an executable file (at a generated hidden path like `/.yamlfile-entrypoint-...`). The entrypoint is set to run it directly. Unlike `run.script` (which is a temporary build-time mount that does not persist in image layers), an entrypoint script *will* be present in the final image. This is a convenience over writing an explicit `copy` step for the script.

If you need the script to receive CMD arguments in the usual way, your script should typically end with `exec "$@"`.

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

Both the file form (`target:`) and env-var form (`env:`) map to BuildKit's `--mount=type=secret` mechanism. Secrets are **never** present in final image layers or history when used correctly.

### SecretMount options

Each entry in a `run.secrets` list supports:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string (required) | Secret identifier; must match `--secret id=...` at build time |
| `target` | string | File path inside the run (default: `/run/secrets/<id>`) |
| `env` | string | Inject as an environment variable instead of a file (value masked in logs) |
| `optional` | bool | If true, the run proceeds when the secret is not supplied |
| `mode` | int | File permission bits (e.g. `0600`) |
| `uid` | int | Owner UID for the mounted secret file |
| `gid` | int | Owner GID for the mounted secret file |

Use either `target:` (file form) or `env:` (env form), not both on the same entry.

See [Features / Secrets]({{< relref "/docs/features/secrets" >}}) for supply examples (`--secret id=...,env=...` or `src=...`) and the exact option semantics.


## Multi-file / Orchestration (`builds:`) — not yet supported at runtime

You can declare other Yamlfiles using the `builds:` section (and refer to them using `component:target` syntax). However, this is not yet supported at runtime:

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

**Current status**: You can write `builds:` sections and use `component:target` references today. If you actually use a cross-file reference for a target that gets built, you will get a clear error. Full support for loading other Yamlfiles and using their targets is not yet implemented.

For now, keep everything in one Yamlfile using multiple named targets + sibling references. Full multi-file support (loading, caching, cross-copy) is planned.

See the status box at the top of this page for current limitations.


## Platform & Defaults

```yaml
defaults:
  platform: linux/amd64

targets:
  foo:
    from: ...
    platform: linux/arm64   # overrides default for this target
```

`platform:` values are of the form `os/arch[/variant]` (e.g. `linux/amd64`, `linux/arm64/v8`).

Precedence (highest first):
- per-target `platform:`
- document `defaults.platform`
- the platform(s) requested by the BuildKit client (e.g. `docker buildx build --platform ...` or the builder default)

When a target declares (or inherits via defaults) a platform:
- Base images (`from:` that are not siblings or scratch) are resolved for that platform (`llb.Image(..., llb.Platform(...))`).
- The exported image config for the target records that platform.
- Sibling `from:` targets use the state produced by the depended-on target (which may have used a different platform); the child's own platform governs its subsequent layers and final config.

Variable expansion and most other steps are platform-agnostic. Building a single target for multiple platforms at once is not yet supported.

## Extensibility

All top-level objects and step types accept an `Extensions` map (via YAML `<<` or unknown keys) so future fields can be added without breaking existing documents.

Unknown step kinds or required fields that are missing will produce clear errors at parse / conversion time.
