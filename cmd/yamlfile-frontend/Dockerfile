# syntax=docker/dockerfile-upstream:master

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.23
ARG XX_VERSION=1.9.0

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
RUN apk add git bash
COPY --from=xx / /
WORKDIR /src
ENV GOFLAGS=-mod=vendor

FROM base AS version
ARG CHANNEL=mainline
RUN --mount=target=. <<EOT
  set -e
  PKG=github.com/builderhub/yamlfile/cmd/yamlfile-frontend
  REVISION=$(git rev-parse HEAD)$(if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)
  echo "-X main.Version=${VERSION:-v1alpha1-dev} -X main.Revision=${REVISION} -X main.Package=${PKG}" | tee /tmp/.ldflags
EOT

FROM base AS build
RUN apk add --no-cache file
ARG TARGETPLATFORM
# yamlfile uses its own go.mod (clear any inherited -mod=vendor etc.)
ENV GOFLAGS=
RUN --mount=target=. --mount=type=cache,target=/root/.cache \
  --mount=target=/go/pkg/mod,type=cache \
  --mount=source=/tmp/.ldflags,target=/tmp/.ldflags,from=version <<EOT
  set -e
  ldflags=$(cat /tmp/.ldflags)
  set -x
  cd /src
  CGO_ENABLED=0 xx-go build -o /yamlfile-frontend -ldflags "-d $ldflags" -tags "netgo static_build osusergo" ./cmd/yamlfile-frontend
  xx-verify --static /yamlfile-frontend
EOT

FROM scratch AS release
LABEL moby.buildkit.frontend.network.none="true"
LABEL moby.buildkit.frontend.caps="moby.buildkit.frontend.inputs,moby.buildkit.frontend.subrequests,moby.buildkit.frontend.contexts,moby.buildkit.frontend.gitquerystring"
LABEL org.opencontainers.image.title="yamlfile"
LABEL org.opencontainers.image.description="BuildKit frontend for Yamlfile (v1alpha1) - easier multi-target, parallel, secret-aware builds"
LABEL org.opencontainers.image.vendor="BuilderHub"
COPY --from=build /yamlfile-frontend /bin/yamlfile-frontend
ENTRYPOINT ["/bin/yamlfile-frontend"]

FROM release
