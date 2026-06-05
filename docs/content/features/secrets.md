---
title: "Secrets (file and env forms)"
weight: 20
---



yamlfile supports BuildKit's native secret mounts in a declarative way.

```yaml
- run:
    script: ./scripts/push-to-registry.sh
    secrets:
      - id: registry_token
        env: REGISTRY_TOKEN          # injected only for this run

      - id: netrc
        target: /root/.netrc         # file form
        mode: 0600
        optional: true
```

## File form vs. Env form

- `target:` (or omitting it) → secret appears as a file (default location `/run/secrets/<id>`).
- `env:` → secret is exported as an environment variable inside the `RUN` (the value is masked in logs by BuildKit).

Both are implemented using `llb.AddSecretWithDest` + the appropriate `SecretAsEnvName` / `SecretFileOpt` options.

## Supplying secrets at build time

```bash
docker buildx build ... \
  --secret id=registry_token,env=REGISTRY_TOKEN \
  --secret id=netrc,src=$HOME/.netrc-for-build
```

Or using the `on-release.yaml` / `on-pr.yaml` patterns with the reusable buildah workflow if you are publishing the image from CI.

See the [Syntax Reference]({{< relref "/syntax-reference#secrets-secretmount" >}}) for the full `SecretMount` options (`optional`, `mode`, `uid`, `gid`).

Secrets are **never** present in the final image layers or history when used correctly.
