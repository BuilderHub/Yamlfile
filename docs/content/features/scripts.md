---
title: "run.script — Baked-in Script Execution"
weight: 10
---

# run.script — Baked-in Script Execution

One of yamlfile's most useful "baked-in" features is `run.script`.

```yaml
steps:
  - run:
      script: ./scripts/install-deps.sh
      env:
        DEBIAN_FRONTEND: noninteractive
```

## How it works (without you writing a COPY)

1. You list the script path in your Yamlfile.
2. The frontend (at build-plan time) reads the file content from your build context using a restricted `llb.Local` + `FollowPaths`.
3. It creates a tiny ephemeral `llb.Scratch()` + `llb.Mkfile(...)` with 0755 perms.
4. The script is mounted **read-only** into the `RUN` filesystem at a generated path (e.g. `/.yamlfile-script-...`).
5. The frontend executes the script directly (or via `/bin/sh`).

Result: the script runs exactly as if it had been copied in, **but it never appears in any layer of the final image** unless you later explicitly `copy` it from the build stage.

This is the moral equivalent of a heredoc `RUN <<'EOF' ...` or a temporary `COPY --from=...` that gets cleaned up, but expressed cleanly in YAML.

## Comparison to Dockerfile patterns

Traditional:

```dockerfile
COPY scripts/install-deps.sh /tmp/
RUN chmod +x /tmp/install-deps.sh && /tmp/install-deps.sh && rm /tmp/install-deps.sh
```

yamlfile:

```yaml
- run:
    script: scripts/install-deps.sh
```

Cleaner, and the frontend guarantees the temporary mount semantics.

See also: [Secrets](/features/secrets) (often used together with scripts that need credentials).
