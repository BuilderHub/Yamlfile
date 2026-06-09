---
title: "run.script — Baked-in Script Execution"
weight: 10
aliases:
  - /features/scripts/
---



One of Yamlfile's most useful "baked-in" features is `run.script`.

```yaml
steps:
  - run:
      script: ./scripts/install-deps.sh
      env:
        DEBIAN_FRONTEND: noninteractive
```

## How it works (without you writing a COPY)

1. You list the script path in your Yamlfile.
2. Yamlfile reads the file content from your build context (only the script you declared).
3. A temporary scratch layer is created with the script content and executable permissions.
4. The script is mounted **read-only** into the `RUN` filesystem at a generated path (e.g. `/.yamlfile-script-...`).
5. The script is executed directly (or via `/bin/sh`).

Result: the script runs exactly as if it had been copied in, **but it never appears in any layer of the final image** unless you later explicitly `copy` it from the build stage.

This is the moral equivalent of a heredoc `RUN <<'EOF' ...` or a temporary `COPY --from=...` that gets cleaned up, but expressed cleanly in YAML.

## Comparison to Dockerfile patterns

Traditional:

```dockerfile
COPY scripts/install-deps.sh /tmp/
RUN chmod +x /tmp/install-deps.sh && /tmp/install-deps.sh && rm /tmp/install-deps.sh
```

Yamlfile:

```yaml
- run:
    script: scripts/install-deps.sh
```

Much cleaner — Yamlfile takes care of the temporary mount for you.

See also: [Secrets]({{< relref "/docs/features/secrets" >}}) (often used together with scripts that need credentials).
