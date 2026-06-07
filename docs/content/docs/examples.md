---
title: "Examples"
weight: 50
aliases:
  - /examples/
---



## Minimal

See `examples/minimal.Yamlfile`:

```yaml
apiVersion: v1alpha1

targets:
  hello:
    from: alpine:3.19
    steps:
      - run:
          command: echo "hello from Yamlfile" > /msg.txt
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

A realistic pattern with:

- Independent targets (`prep`, `test`) that have no dep on each other.
- A `build` target that uses `run.script`, `run.env`, and an `env:` step.
- A `final` target that copies outputs from multiple previous targets.
- A secret declared for the build step (supply at `docker buildx` time with `--secret`).

See the complete, self-contained file + supporting script:

- `examples/multi-target.Yamlfile`
- `examples/scripts/prepare.sh`

The example is fully runnable today (uses only currently implemented features). Because the leaf target is named `default`, you do not need `--target`:

```bash
docker buildx build -f examples/multi-target.Yamlfile \
  --build-arg BUILDKIT_SYNTAX=ghcr.io/builderhub/yamlfile:latest \
  --output type=local,dest=/tmp/out \
  .
cat /tmp/out/final.txt
ls /tmp/out/out/   # the copied artifacts from prep, build, and test targets live here
```

(If you name your final target something else, e.g. `final`, pass `--target final`.)

To exercise the (optional) secret path:

```bash
# create a dummy secret file (in real life this would be a token)
printf 'dummy-token' > /tmp/dummy-token
docker buildx build -f examples/multi-target.Yamlfile \
  --secret id=token,src=/tmp/dummy-token \
  --build-arg BUILDKIT_SYNTAX=ghcr.io/builderhub/yamlfile:latest \
  --output type=local,dest=/tmp/out \
  .
```

See the [Syntax Reference]({{< relref "/docs/syntax-reference" >}}) for the grammar and [Features]({{< relref "/docs/features" >}}) for deep dives into `run.script` and secrets.

## Build args, variable expansion, and workdir

A small pattern using the features added in this iteration:

```yaml
apiVersion: v1alpha1
targets:
  build:
    from: golang:1.25
    steps:
      - arg:
          vars:
            VERSION: "dev"
      - env:
          vars:
            CGO_ENABLED: "0"
            BIN: /out/myapp-${VERSION}
      - workdir:
          path: /src
      - run:
          command: go build -o ${BIN} .
          workdir: /src/cmd/myapp   # per-run override (does not persist)
```

Build with an override:

```bash
docker buildx build ... --build-arg VERSION=1.2.3 ...
```

More examples will be added in future releases (multi-file orchestration, explicit platform handling, and intra-build parallel execution are on the roadmap).

