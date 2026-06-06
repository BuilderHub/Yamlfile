#!/bin/sh
set -eux

PKG=github.com/builderhub/yamlfile/cmd/yamlfile-frontend
REVISION=$(git rev-parse HEAD)$(if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)
LDFLAGS="-X main.Version=${VERSION:-v1alpha1-dev} -X main.Revision=${REVISION} -X main.Package=${PKG}"

CGO_ENABLED=0 go build \
	-o /yamlfile-frontend \
	-ldflags "-d ${LDFLAGS}" \
	-tags "netgo static_build osusergo" \
	./cmd/yamlfile-frontend

file /yamlfile-frontend
