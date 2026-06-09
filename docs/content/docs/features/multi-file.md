---
title: "Multi-file Builds & Cross-Target Copy"
weight: 25
aliases:
  - /features/multi-file/
---

> **Note**: You can write `builds:` sections and use `component:target` references (e.g. `from: "torch:base"`) in your Yamlfile today. These will be accepted, but cross-file builds are not yet supported at runtime. Only targets defined in the *same* Yamlfile can be used with `from:` and `copy.from:`. The examples below show the intended future behavior.

Multi-file orchestration will let a single top-level `Yamlfile` coordinate builds defined in other `Yamlfile`s (often in subdirectories) and then `copy` artifacts across them.

## Intended `builds:` section (not yet supported at runtime)

```yaml
builds:
  torch:
    file: torch/Yamlfile
    target: base          # optional; defaults to first reachable or a target named "default"

targets:
  final:
    from: alpine
    steps:
      - copy:
          from: "torch:base"   # "component:target" syntax
          src: ["/opt/torch"]
          dest: "/torch"
```

- `file:` will be resolved relative to the main build context.
- The referenced file will be loaded and its named target (or default) built to a state usable for `from:` / `copy.from:`.
- Use the `component:target` form to name the origin.

## `copy` from other targets (sibling form works today)

```yaml
- copy:
    from: "previous-target"    # sibling in same file (supported today)
    src: ["./bin/", "LICENSE"]
    dest: "/app/"
```

The cross-file form (`from: "torch:base"`) is accepted when you write your file. If you try to use one for an actual build, you will get a clear error explaining that cross-file builds are not supported yet.

`from` may be omitted, left empty, or set to `"context"` to copy from the original build context (this works today).

## Current status

See the status box at the top of the [Syntax Reference]({{< relref "/docs/syntax-reference" >}}).

In short: keep related targets in one Yamlfile for now. Targets within the same file work reliably with `from:` and `copy.from:`.

When full multi-file support lands, the same `from:` / `copy.from:` syntax will work across files.

