# Yamlfile

**BuildKit frontend** (v1alpha1) for intuitive, declarative, parallel, multi-file, secret-aware builds that are easier to reason about than large Dockerfiles.

Inspired by the complexity of [coreweave/ml-containers](https://github.com/coreweave/ml-containers) (hundreds of lines of parallel downloaders, secret env mounts, shared scripts via contexts, cross-stage wheel copies, etc.).

## Status

First iteration (v1alpha1). Core: explicit targets, run (command / inline / script), copy, per-run secrets (file + env forms), env: convenience step, dependency graph + cycle detection + reachable ordering. Multi-file orchestration (`builds:` + `component:target`), platform overrides, and intra-build parallel execution are planned (grammar/graph prep exists for forward-compat).

## Documentation

Full docs (syntax, features, examples, getting started) are published via GitHub Pages:

https://builderhub.github.io/Yamlfile/

## License

MIT (BuilderHub)
