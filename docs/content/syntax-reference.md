---
title: "Syntax Reference (v1alpha1)"
weight: 20
---

# Yamlfile v1alpha1 Syntax Reference

This page is the authoritative reference for the structure understood by the `yamlfile` BuildKit frontend.

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

A step has exactly one of `run`, `copy`, or `env`.

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

    secrets:
      - id: mytoken
        env: GITHUB_TOKEN
      - id: netrc
        target: /root/.netrc
        mode: 0600
        optional: true
```

- `script` paths are resolved relative to the build context. The frontend loads the content (via a restricted local source) and makes it available inside the `RUN` via a temporary scratch mount. The script does **not** end up as a layer in the final image unless you explicitly copy it.
- Secrets are passed using BuildKit's native `--mount=type=secret` mechanism. They are never present in image layers or history.

### copy

```yaml
- copy:
    from: "previous-target"    # or "component:target", or empty / "context" for the main build context
    src: ["./bin/", "LICENSE"]
    dest: "/app/"
```

`from` can be a sibling target (or a target from another file via the `builds:` mechanism).

### env

Convenience form (equivalent to putting `env:` inside a `run` step, but clearer when you just want to set image config).

```yaml
- env:
    PATH: "/app/bin:${PATH}"
    FOO: bar
```

## Secrets (SecretMount)

```yaml
secrets:
  - id: foo
    target: /run/secrets/foo     # file form (default location if omitted: /run/secrets/<id>)
    # or
    env: FOO_SECRET              # env-var form (injected only for the duration of that run)

    optional: false
    mode: 0400
    uid: 0
    gid: 0
```

Matches the semantics of `RUN --mount=type=secret,id=...,env=...` (and the file form) in modern Dockerfiles.

## Multi-file / Orchestration (`builds:`)

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

The frontend loads the referenced Yamlfile(s) from the build context, resolves the named target inside it, and makes the resulting state available for `from:` / `copy.from:`.

## Platform & Defaults

```yaml
defaults:
  platform: linux/amd64

targets:
  foo:
    from: ...
    platform: linux/arm64   # overrides default for this target
```

## Extensibility

All top-level objects and step types accept an `Extensions` map (via YAML `<<` or unknown keys) so future fields can be added without breaking existing documents.

Unknown step kinds or required fields that are missing will produce clear errors at parse / conversion time in the current v1alpha1 implementation.
