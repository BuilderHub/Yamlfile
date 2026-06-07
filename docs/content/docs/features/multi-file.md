---
title: "Multi-file Builds & Cross-Target Copy"
weight: 25
aliases:
  - /features/multi-file/
---

> **Note**: Grammar + graph support for `builds:` exists for forward compatibility and so that external tools can see the declared structure. **Full runtime support is not yet implemented.** Only same-file sibling targets work for `from:` and `copy.from:` today. The claims and examples below describe the intended design; see the "Current status" callout.

Multi-file orchestration will let a single top-level `Yamlfile` coordinate builds defined in other `Yamlfile`s (often in subdirectories) and then `copy` artifacts across them.

## Intended `builds:` section (grammar only today)

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

The cross-file form (`from: "torch:base"`) is parsed and represented in the graph but not yet wired for loading.

`from` may be omitted / set to empty or `"context"` to copy from the caller's original build context (supported).

## Current status

See the prominent note at the top of the [Syntax Reference]({{< relref "/docs/syntax-reference#multi-file--orchestration-builds--grammar-only-not-yet-implemented" >}}).

In short: keep related targets in one Yamlfile for now. The dependency graph, cycle detection, and reachable set already handle multi-target single-file cases well today.

When multi-file lands, the same `from:` / `copy.from:` syntax will just work across files.

