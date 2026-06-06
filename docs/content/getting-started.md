---
title: "Getting Started"
weight: 10
---



## Prerequisites

- Docker with BuildKit enabled (Docker 23+ or `DOCKER_BUILDKIT=1`).

## Using a Released Image

The official images are published to `ghcr.io/builderhub/yamlfile`.

In your `Yamlfile` (or any file you pass with `-f`):

```yaml
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
docker buildx build -f MyYamlfile \
  --output type=image,name=myapp,push=false \
  .
```

If your Yamlfile defines multiple top-level targets and you don't want the default (first reachable), pass `--target`:

```bash
docker buildx build -f MyYamlfile \
  --target myapp \
  --output type=image,name=myapp,push=false \
  .
```

To use a custom frontend image instead of the published one, pass `--build-arg BUILDKIT_SYNTAX=<your-image>`.

## Build from source

To build the frontend from source or run project CI locally, see [Development]({{< relref "/development" >}}).

## Supplying Secrets

Yamlfile passes secrets through to BuildKit's native secret mechanism. Example:

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

See the [Secrets]({{< relref "/features/secrets" >}}) page for details on file vs. env forms and options (`optional`, `mode`, `uid`, `gid`).

## Next Steps

- Read the [Syntax Reference]({{< relref "/syntax-reference" >}}) for the full v1alpha1 grammar.
- Look at [Examples]({{< relref "/examples" >}}).
- See [Development]({{< relref "/development" >}}) for implementation details and contributor setup.
