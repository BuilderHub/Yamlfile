---
title: "Development"
weight: 70
---

## Prerequisites

- Docker with BuildKit (23+ recommended).
- Nix (for the hermetic dev shell that includes Go, hugo, linters, docker-buildx, etc.).

## Setup

```bash
git clone https://github.com/BuilderHub/Yamlfile.git
cd Yamlfile
nix develop
```

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

## Documentation

```bash
make docs        # build to docs/public (used by the GitHub Pages deploy)
make docs-serve  # live reload at http://localhost:1313
```

- Edit files under `docs/content/`.
- Frontmatter `title:` + `weight:` controls ordering in the sidebar toctree and section lists.
- Internal links: use the `relref` shortcode (e.g. `[text]({{</* relref "/getting-started" */>}})) so they resolve correctly under the `baseURL` sub-path (e.g. `/Yamlfile/`).
- Run `make docs` (or the CI check) before pushing; it must succeed with no errors.
- Navigation: a persistent sidebar toctree (`docs/layouts/partials/toctree.html`) replaces the old per-page "On this page" heading TOC. Search uses a Hugo-generated `search-index.json` plus client-side Fuse.js (`docs/static/js/`).
- The site is intentionally lightweight (custom layouts + a few partials + small CSS). A full theme (e.g. via Hugo modules) can be adopted later.

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

All changes to the v1alpha1 surface should be reflected in `docs/content/syntax-reference.md` and usually a short note or example under `docs/content/features/`.
