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
make docker-build  # current-arch image tagged localhost... or REGISTRY=... TAG=...
```

See `Makefile` for the full set (including multi-arch push flows used in release).

## Documentation

```bash
make docs        # build to docs/public (used by the GitHub Pages deploy)
make docs-serve  # live reload at http://localhost:1313
```

- Edit files under `docs/content/`.
- Frontmatter `title:` + `weight:` controls ordering in lists/ToC.
- Internal links: use the `relref` shortcode (e.g. `[text]({{</* relref "/getting-started" */>}})) so they resolve correctly under the `baseURL` sub-path (e.g. `/yamlfile/`).
- Run `make docs` (or the CI check) before pushing; it must succeed with no errors.
- The site is intentionally lightweight (custom layouts + a few partials + small CSS). A full theme (e.g. via Hugo modules) can be adopted later.

## How docs deployment works

- Push to `main` that touches `docs/**` (or the workflow file) triggers `.github/workflows/deploy-docs.yaml`.
- It runs `nix develop --command make docs` (exact same env as your machine) then uses the official `actions/deploy-pages` flow.
- The published site is at the `baseURL` declared in `docs/hugo.toml` (`https://builderhub.github.io/yamlfile/`).

## Code layout (relevant to docs)

- `pkg/spec/v1alpha1/` — the types + parser (update syntax-reference.md when changing surface).
- `pkg/convert/` — graph + LLB emission (the source of truth for parallelism helpers, script mounting, secret handling, and the current same-file target/copy implementation; multi-file loading is future work).
- `cmd/yamlfile-frontend/` — the BuildKit gateway entrypoint + Dockerfile.

## Contributing

1. `nix develop`
2. `make ci`
3. Make your change + add/adjust tests or docs.
4. `make docs` (if you touched content).
5. Open PR against `main`.

All changes to the v1alpha1 surface should be reflected in `docs/content/syntax-reference.md` and usually a short note or example under `docs/content/features/`.
