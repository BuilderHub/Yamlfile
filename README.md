# Yamlfile

**BuildKit frontend** for intuitive, declarative, parallel, multi-file, secret-aware builds that are easier to reason about than large Dockerfiles.

Inspired by the complexity of [coreweave/ml-containers](https://github.com/coreweave/ml-containers) (hundreds of lines of parallel downloaders, secret env mounts, shared scripts via contexts, cross-stage wheel copies, etc.).

## Status

**What you can do today**

- Define named `targets` that depend on images or other targets in the same file using `from:`.
- Use `run` with a one-line command, multi-line inline script, or a `script:` from your build context (Yamlfile handles loading and mounting it securely — it won't end up in the final image layers).
- Copy files from the build context or previous targets.
- Pass secrets securely to individual `run` steps (as files or environment variables).
- Set image configuration with `env:`, `arg:`, `workdir:`, `label:`, and `entrypoint:` steps.
- Use variable expansion (`$VAR` or `${VAR}`) in many places.
- Declare a default or per-target platform.

**Multi-file builds (not yet supported)**

- You can write a `builds:` section to declare other Yamlfiles, and use `component:target` references (e.g. `from: "torch:base"`).
- These are recognized and will produce a clear error if you try to use them. Full support for loading and building targets from other Yamlfiles is not yet implemented. For now, keep all targets in a single file.

**Behavior notes**

- Targets are always built in the order required by their dependencies.
- Independent targets (ones that don't depend on each other) can run without being forced into a strict sequence.
- Multi-platform builds are not yet supported (a single platform is used per build).

For the latest on planned features, see the status box in the [Syntax Reference](https://builderhub.github.io/Yamlfile/docs/syntax-reference/).

## Documentation

Full docs (syntax, features, examples, getting started) are published via GitHub Pages:

https://builderhub.github.io/Yamlfile/

## License

MIT (BuilderHub)
