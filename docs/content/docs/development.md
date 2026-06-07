---
title: "Development"
weight: 70
aliases:
  - /development/
---

## Prerequisites

- Docker with BuildKit (23+ recommended).
- Nix (for the hermetic dev shell that includes Go, hugo, linters, docker-buildx, etc.).

## Setup

```bash
git clone --recurse-submodules https://github.com/BuilderHub/Yamlfile.git
cd Yamlfile
nix develop
```

If you already cloned without submodules: `git submodule update --init --recursive`.

## Common commands

```bash
make help          # list targets
make ci            # tests + lint + vet + revive
make test
make generate-schema  # (re)generate docs/static/schema/v1alpha1.json from pkg/spec/v1alpha1 (also run by docs)
make docs
make docker-build  # current-arch image tagged localhost... or REGISTRY=... TAG=...
```

See `Makefile` for the full set (including multi-arch push flows used in release).

## CI and release

Image builds and releases are driven by GitHub Actions workflows under `.github/workflows/`:

- `on-pr.yaml` — CI checks on pull requests.
- `on-release.yaml` — release builds using the reusable buildah workflow pattern for publishing images.

When supplying secrets in CI builds (e.g. registry tokens), use the same `--secret id=...,env=...` or `--secret id=...,src=...` flags as local `docker buildx` invocations.

## Documentation

```bash
make docs        # build to docs/public (used by the GitHub Pages deploy)
make docs-serve  # live reload at http://localhost:1313
make docs-mod    # update Hugo Book theme module (docs/go.mod)
```

The site uses the [Hugo Book](https://github.com/alex-shpak/hugo-book) theme (v0.14.0), installed as a git submodule at `docs/themes/hugo-book` and tracked in `docs/go.mod` via Hugo Modules.

- Edit pages under `docs/content/docs/` (the Book sidebar is built from this section).
- The home page is `docs/content/_index.md` (uses the standard Book layout with sidebar and optional right-rail ToC).
- Front matter `title:` + `weight:` controls sidebar order. Book-specific params include `bookToC`, `bookHidden`, and `bookCollapseSection`.
- Internal links: use the `relref` shortcode with paths under `/docs/...` (e.g. `[text]({{</* relref "/docs/getting-started" */>}}))` so they resolve correctly under the GitHub Pages sub-path (`/Yamlfile/`).
- Moved pages include `aliases:` for old URLs (e.g. `/getting-started/` → `/docs/getting-started/`).
- Search, dark/light mode, and right-rail table of contents are provided by the Book theme (`BookSearch`, `BookTheme`, `BookToC` in `docs/hugo.toml`).
- Run `make docs` (or the CI check) before pushing; it must succeed with no errors.

## How docs deployment works

- Push to `main` that touches `docs/**` (or the workflow file) triggers `.github/workflows/pages.yaml`.
- It runs `nix develop --command make docs` (exact same env as your machine) then uses the official `actions/deploy-pages` flow.
- The published site is at the `baseURL` declared in `docs/hugo.toml` (`https://builderhub.github.io/Yamlfile/`).

## Code layout (relevant to docs)

- `pkg/spec/v1alpha1/` — the types + parser (update syntax-reference.md when changing surface). The JSON Schema under `docs/static/schema/` is generated from these types.
- `pkg/convert/` — graph + LLB emission (the source of truth for parallelism helpers, script mounting, secret handling, variable expansion, and the current same-file target/copy implementation; multi-file loading is future work).
- `cmd/yamlfile-frontend/` — the BuildKit gateway entrypoint, Dockerfile (release builds), and Yamlfile (dogfooded image build; `make docker-build-from-yamlfile`).
- `hack/gen-schema/` — the (stdlib-only) generator that produces `docs/static/schema/v1alpha1.json` from the live Go types. It is invoked automatically by `make docs`.

## Contributing

1. `nix develop`
2. `make ci`
3. Make your change + add/adjust tests or docs.
4. `make docs` (if you touched content or the v1alpha1 types — this also regenerates the JSON Schema).
5. Open PR against `main`.

All changes to the v1alpha1 surface should be reflected in `docs/content/docs/syntax-reference.md` and usually a short note or example under `docs/content/docs/features/`.
