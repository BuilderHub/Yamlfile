---
title: "Getting Started"
weight: 10
---

# Getting Started with yamlfile

## Prerequisites

- Docker with BuildKit enabled (Docker 23+ or `DOCKER_BUILDKIT=1`).
- (Recommended for development) `nix develop` from the BuilderHub monorepo root.

## Using a Released Image

The official images are published to `ghcr.io/builderhub/yamlfile`.

In your `Yamlfile` (or any file you pass with `-f`):

```dockerfile
# syntax=ghcr.io/builderhub/yamlfile:latest
apiVersion: v1alpha1

targets:
  myapp:
    from: golang:1.25
    steps:
      - run:
          command: go build -o /out/myapp .
```

Then build with:

```bash
docker buildx build -f MyYamlfile.yaml \
  --output type=image,name=myapp,push=false \
  .
```

## Local Development / Custom Build

```bash
# From the BuilderHub monorepo root (so the Dockerfile can see ../buildkit-hive)
docker buildx build \
  -f yamlfile/cmd/yamlfile-frontend/Dockerfile \
  -t localhost:5000/yamlfile:dev \
  --load \
  .

# Use your local image
docker buildx build -f yamlfile/examples/minimal.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=localhost:5000/yamlfile:dev \
  --output type=local,dest=/tmp/out \
  yamlfile
```

See the [Makefile](/Makefile) targets (`make docker-build`, `make docker-build-multiarch`) for the canonical commands used in CI/release.

## Supplying Secrets

yamlfile passes secrets through to BuildKit's native secret mechanism. Example:

```yaml
targets:
  build:
    from: golang:1.25
    steps:
      - run:
          script: ./scripts/build-with-creds.sh
          secrets:
            - id: github_token
              env: GITHUB_TOKEN
            - id: netrc
              target: /root/.netrc
```

Invoke with:

```bash
docker buildx build ... \
  --secret id=github_token,env=GITHUB_TOKEN \
  --secret id=netrc,src=$HOME/.netrc
```

See the [Secrets](/features/secrets) page for details on file vs. env forms and options (`optional`, `mode`, `uid`, `gid`).

## Next Steps

- Read the [Syntax Reference](/syntax-reference) for the full v1alpha1 grammar.
- Look at [Examples](/examples).
- See how the frontend implements script injection and secret mounts in the source (`pkg/convert/`).
