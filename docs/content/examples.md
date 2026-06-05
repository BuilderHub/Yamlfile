---
title: "Examples"
weight: 50
---

# Examples

## Minimal

See `examples/minimal.Yamlfile` in the repository:

```yaml
apiVersion: v1alpha1

targets:
  hello:
    from: alpine:3.19
    steps:
      - run:
          command: echo "hello from yamlfile v1alpha1" > /msg.txt
      - run:
          inline: |
            echo "inline shell logic works" >> /msg.txt
```

Build it:

```bash
docker buildx build -f examples/minimal.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=ghcr.io/builderhub/yamlfile:latest \
  --output type=local,dest=/tmp/out \
  .
cat /tmp/out/msg.txt
```

## Multi-target + script + secret (typical pattern)

(Expanded example showing independent download stages, a builder that uses a script + S3 secrets as env, and a final stage that copies artifacts from multiple previous targets.)

See the full source tree and the [Syntax Reference](/syntax-reference) for the building blocks.

More examples will be added here as the project evolves.
