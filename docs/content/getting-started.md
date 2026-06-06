---
title: "Getting Started"
weight: 10
---



## Prerequisites

- Docker with BuildKit enabled (Docker 23+ or `DOCKER_BUILDKIT=1`).
- (Recommended for development) `nix develop` (from the yamlfile directory).

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

## Local Development / Custom Build

From the yamlfile directory:

```bash
docker buildx build \
  -f cmd/yamlfile-frontend/Dockerfile \
  -t localhost:5000/yamlfile:dev \
  --load \
  .

# Use your local image (context can be any dir with your Yamlfile; here "." for the example)
docker buildx build -f examples/minimal.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=localhost:5000/yamlfile:dev \
  --output type=local,dest=/tmp/out \
  .
```

See the `Makefile` targets (`make docker-build`, `make docker-build-multiarch`) for the canonical commands used in CI/release. (Run `make` from the yamlfile directory, or `make -C /path/to/yamlfile ...`.) The source is at the root of the repository.

To dogfood the Yamlfile-based frontend image build (requires a published yamlfile image as bootstrap), use `make docker-build-from-yamlfile`.

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

See the [Secrets]({{< relref "/features/secrets" >}}) page for details on file vs. env forms and options (`optional`, `mode`, `uid`, `gid`).

## Next Steps

- Read the [Syntax Reference]({{< relref "/syntax-reference" >}}) for the full v1alpha1 grammar.
- Look at [Examples]({{< relref "/examples" >}}).
- See how the frontend implements script injection and secret mounts in the source (`pkg/convert/`).
