// Package convert implements translation of Yamlfile v1alpha1 specs into
// BuildKit LLB. It handles target dependency graphs (for parallelism and
// pruning), RUN/COPY/ENV/ARG/WORKDIR steps, $VAR/${VAR} expansion (via
// buildkit shell lexer) for env/arg/workdir values, baked-in script loading
// (via temporary scratch + readonly mounts), and secure secret mounts (file or env forms).
package convert

//revive:disable:exported
//revive:disable:unused-parameter

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/moby/buildkit/frontend/dockerui"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)


// ConvertOpt carries context from the frontend (build args, platform, etc.).
type ConvertOpt struct {
	BuildArgs map[string]string
	Platform  *ocispecs.Platform
	// Context is the main build context state (for COPY from context)
	Context llb.State
	// Scripts (populated inside ToLLB after the reachable set for the chosen target
	// is known) maps script path -> bytes for run.script injection. Only scripts
	// referenced by reachable targets are loaded (via ScriptLoader if provided).
	Scripts map[string][]byte
	// ScriptLoader (optional) is called inside ToLLB (post-reachable) to load
	// script content on demand. This scopes loading to only the targets that will
	// actually be built for the requested --target (important for multi-target
	// Yamlfies that reference scripts only in some branches).
	ScriptLoader func(path string) ([]byte, error)
}

// Result contains the built LLB state + image config for a target.
type Result struct {
	State llb.State
	Image *ocispecs.Image // minimal config for export
}

// ToLLB builds the target graph for the requested target (or all) and returns the
// final target state + config. For v1alpha1 MVP this is single-platform and serial
// within a target; independent targets are prepared for parallelism by the caller.
func ToLLB(ctx context.Context, y *v1alpha1.Yamlfile, target string, opt ConvertOpt) (map[string]Result, error) {
	deps, err := buildDependencyGraph(y)
	if err != nil {
		return nil, err
	}
	if err := validateNoCycles(deps); err != nil {
		return nil, err
	}

	// Concrete target is required for v1alpha1 (frontend always supplies one; direct
	// callers must pick). This avoids non-deterministic map order and silent "not-yet-built".
	if target == "" {
		return nil, fmt.Errorf("ToLLB: target name is required (use --target or a single/default target)")
	}
	if _, ok := y.Targets[target]; !ok {
		return nil, fmt.Errorf("target %q not found in Yamlfile", target)
	}

	reachable, err := reachableTargets(y, target, deps)
	if err != nil {
		return nil, err
	}

	// Scoped script preload: only load scripts referenced by *reachable* targets
	// for the chosen target. This prevents a missing script in an unrelated target
	// (common in multi-target Yamlfies) from failing the whole build.
	scripts := map[string][]byte{}
	if opt.ScriptLoader != nil {
		for _, name := range reachable {
			t := y.Targets[name]
			for _, stp := range t.Steps {
				if stp.Run != nil && stp.Run.Script != "" {
					p := stp.Run.Script
					if _, ok := scripts[p]; ok {
						continue
					}
					b, err := opt.ScriptLoader(p)
					if err != nil {
						return nil, fmt.Errorf("load script %s for run (target %s): %w", p, name, err)
					}
					scripts[p] = b
				}
			}
		}
	}
	opt.Scripts = scripts

	// For MVP we build serially in reachable order (which is post-order deps first).
	// Parallel roots can be built concurrently by the frontend using errgroup (see step 7).
	states := make(map[string]llb.State, len(reachable))
	images := make(map[string]*ocispecs.Image, len(reachable))

	for _, name := range reachable {
		t := y.Targets[name]
		base := llb.Scratch()
		if t.From != "" {
			if isT, k := resolveBase(t.From, y.Targets); isT {
				if st, ok := states[k]; ok {
					base = st
				} else {
					return nil, fmt.Errorf("target %s depends on unknown or not-yet-built %s", name, t.From)
				}
			} else {
				base = llb.Image(t.From)
			}
		}

		st := base
		img := emptyImage(opt.Platform)

		// Per-target variable context for $VAR / ${VAR} expansion inside env/arg/workdir values.
		// Seeded from CLI build args (highest precedence for expansion), our synthetic base PATH,
		// and (for sibling from:) the exact final ENVs + WorkingDir of the prior target.
		// This makes documented patterns like PATH: /app/bin:${PATH} and cross-arg/env refs work.
		currentVars := map[string]string{}
		argScope := map[string]string{} // build args + arg: defaults; injected into run execs (not persisted to image ENV)
		for k, v := range opt.BuildArgs {
			currentVars[k] = v
			argScope[k] = v
		}
		for _, e := range img.Config.Env {
			if k, v := splitEnv(e); k != "" {
				currentVars[k] = v
			}
		}
		if isT, k := resolveBase(t.From, y.Targets); isT && k != "scratch" {
			if pimg, ok := images[k]; ok && pimg != nil {
				// Inherit exact envs for expansion and make the child's starting image config correct.
				img.Config.Env = append([]string{}, pimg.Config.Env...)
				img.Config.WorkingDir = pimg.Config.WorkingDir
				for _, e := range pimg.Config.Env {
					if key, val := splitEnv(e); key != "" {
						currentVars[key] = val
					}
				}
			}
		}

		for _, step := range t.Steps {
			if step.Run != nil {
				// Expand per-run env values and workdir using the context at this point in the target.
				// We operate on a copy so we never mutate the parsed spec.
				r := step.Run
				rCopy := *r
				if len(r.Env) > 0 {
					rCopy.Env = make(map[string]string, len(r.Env))
					for k, v := range r.Env {
						ev, err := expand(v, currentVars)
						if err != nil {
							return nil, fmt.Errorf("expand env %s for run in target %s: %w", k, name, err)
						}
						rCopy.Env[k] = ev
						currentVars[k] = ev // run.env persists today; keep expansion context in sync
					}
				}
				if r.Workdir != "" {
					wd, err := expand(r.Workdir, currentVars)
					if err != nil {
						return nil, fmt.Errorf("expand workdir for run in target %s: %w", name, err)
					}
					rCopy.Workdir = wd
				}
				st, img = dispatchRun(st, img, &rCopy, opt, argScope)
			}
			if step.Copy != nil {
				// Expand copy.from for symmetry (e.g. from: a build arg). No impact on static dep graph.
				c := step.Copy
				fromRef := c.From
				if c.From != "" {
					if f, err := expand(c.From, currentVars); err == nil && f != "" {
						fromRef = f
					}
				}
				c2 := *c
				c2.From = fromRef
				var err error
				st, err = dispatchCopy(st, &c2, states, opt)
				if err != nil {
					return nil, err
				}
			}
			if step.Env != nil {
				for k, v := range step.Env.Vars {
					ev, err := expand(v, currentVars)
					if err != nil {
						return nil, fmt.Errorf("expand env %s in target %s: %w", k, name, err)
					}
					st = st.AddEnv(k, ev)
					img.Config.Env = upsertEnv(img.Config.Env, k, ev)
					currentVars[k] = ev
				}
			}
			if step.Arg != nil {
				// arg: provides defaults (only if not supplied via CLI BuildArgs) and participates in expansion.
				// Values are available for later ${} expansion and are injected into subsequent run execs.
				for k, d := range step.Arg.Vars {
					if _, ok := currentVars[k]; !ok {
						dv, err := expand(d, currentVars)
						if err != nil {
							return nil, fmt.Errorf("expand arg %s in target %s: %w", k, name, err)
						}
						currentVars[k] = dv
						argScope[k] = dv
					}
				}
			}
			if step.Workdir != nil {
				p := step.Workdir.Path
				ep, err := expand(p, currentVars)
				if err != nil {
					return nil, fmt.Errorf("expand workdir in target %s: %w", name, err)
				}
				st = st.Dir(ep)
				img.Config.WorkingDir = ep
			}
		}

		states[name] = st
		images[name] = img
	}

	res := make(map[string]Result, len(reachable))
	for _, n := range reachable {
		res[n] = Result{State: states[n], Image: images[n]}
	}
	return res, nil
}

func dispatchRun(st llb.State, img *ocispecs.Image, r *v1alpha1.RunSpec, opt ConvertOpt, extraExecEnvs map[string]string) (llb.State, *ocispecs.Image) {
	var args []string
	scriptContent, hasScript := opt.Scripts[r.Script]

	if r.Command != "" {
		args = []string{"/bin/sh", "-c", r.Command}
	} else if r.Inline != "" {
		args = []string{"/bin/sh", "-c", r.Inline}
	} else if hasScript {
		// Baked-in: inject script via temp scratch + readonly mount. No COPY layer in image.
		scriptPath := "/.yamlfile-script-" + sanitize(r.Script)
		scriptSt := llb.Scratch().File(llb.Mkfile(scriptPath, 0755, scriptContent))
		runOpts := []llb.RunOption{
			llb.Args([]string{"/bin/sh", scriptPath}),
			llb.AddMount(scriptPath, scriptSt, llb.SourcePath(scriptPath), llb.Readonly),
		}
		if r.Workdir != "" {
			runOpts = append(runOpts, llb.Dir(r.Workdir))
		}
		for k, v := range r.Env {
			runOpts = append(runOpts, llb.AddEnv(k, v))
		}
		for k, v := range extraExecEnvs {
			runOpts = append(runOpts, llb.AddEnv(k, v))
		}
		for _, sm := range r.Secrets {
			runOpts = append(runOpts, secretRunOpt(sm))
		}
		exec := st.Run(runOpts...)
		newSt := exec.Root()
		for k, v := range r.Env {
			img.Config.Env = upsertEnv(img.Config.Env, k, v)
		}
		return newSt, img
	} else if r.Script != "" {
		// Fallback: assume script was copied by user or is on PATH in base
		args = []string{"/bin/sh", r.Script}
	} else {
		return st, img
	}

	runOpts := []llb.RunOption{llb.Args(args)}
	if r.Workdir != "" {
		runOpts = append(runOpts, llb.Dir(r.Workdir))
	}
	for k, v := range r.Env {
		runOpts = append(runOpts, llb.AddEnv(k, v))
	}
	for k, v := range extraExecEnvs {
		runOpts = append(runOpts, llb.AddEnv(k, v))
	}
	for _, sm := range r.Secrets {
		runOpts = append(runOpts, secretRunOpt(sm))
	}

	exec := st.Run(runOpts...)
	newSt := exec.Root()

	for k, v := range r.Env {
		img.Config.Env = upsertEnv(img.Config.Env, k, v)
	}
	return newSt, img
}

func secretRunOpt(sm v1alpha1.SecretMount) llb.RunOption {
	id := sm.ID
	var target *string
	opts := []llb.SecretOption{llb.SecretID(id)}

	if sm.Optional {
		opts = append(opts, llb.SecretOptional)
	}

	// File dest logic (mirrors dockerfile2llb/convert_secrets + llb semantics):
	// - explicit target: always a file mount at that path
	// - pure env (no target): leave target=nil (no file mount, only SecretAsEnvName)
	// - no target and no env: default file at /run/secrets/<id>
	if sm.Target != "" {
		t := sm.Target
		target = &t
	} else if sm.Env == "" {
		t := "/run/secrets/" + id
		target = &t
	}

	if sm.Env != "" {
		opts = append(opts, llb.SecretAsEnvName(sm.Env))
	}

	if sm.Mode != nil || sm.UID != nil || sm.GID != nil {
		mode := 0400
		if sm.Mode != nil {
			mode = *sm.Mode
		}
		uid, gid := 0, 0
		if sm.UID != nil {
			uid = *sm.UID
		}
		if sm.GID != nil {
			gid = *sm.GID
		}
		opts = append(opts, llb.SecretFileOpt(uid, gid, mode))
	}
	return llb.AddSecretWithDest(id, target, opts...)
}

func sanitize(s string) string {
	// very rough for filename in mount
	return strings.ReplaceAll(strings.ReplaceAll(s, "/", "_"), "..", "_")
}

func dispatchCopy(st llb.State, c *v1alpha1.CopySpec, states map[string]llb.State, opt ConvertOpt) (llb.State, error) {
	if c.From == "" || c.From == "context" {
		// context copy (MVP: rely on opt.Context if provided; otherwise no-op for smoke)
		if opt.Context.Output() != nil {
			for _, src := range c.Src {
				st = st.File(llb.Copy(opt.Context, src, c.Dest, &llb.CopyInfo{CreateDestPath: true}))
			}
		}
		return st, nil
	}
	srcState, ok := states[c.From]
	if !ok {
		return st, fmt.Errorf("copy from unknown target %q (ensure the source target is defined earlier or use a valid build context)", c.From)
	}
	for _, src := range c.Src {
		st = st.File(llb.Copy(srcState, src, c.Dest, &llb.CopyInfo{CreateDestPath: true}))
	}
	return st, nil
}

func emptyImage(p *ocispecs.Platform) *ocispecs.Image {
	if p == nil {
		p = &ocispecs.Platform{OS: "linux", Architecture: "amd64"}
	}
	return &ocispecs.Image{
		Platform: *p,
		Config: ocispecs.ImageConfig{
			Env:        []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			WorkingDir: "/",
		},
	}
}

func upsertEnv(envs []string, k, v string) []string {
	prefix := k + "="
	for i, e := range envs {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			envs[i] = k + "=" + v
			return envs
		}
	}
	return append(envs, k+"="+v)
}

// mapEnvGetter adapts a string map to buildkit's shell.EnvGetter for ProcessWord.
type mapEnvGetter map[string]string

func (m mapEnvGetter) Get(key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func (m mapEnvGetter) Keys() []string {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// expand performs Dockerfile-style $VAR / ${VAR} expansion (with quoting support)
// using BuildKit's own shell lexer. This ensures identical behavior to the official
// Dockerfile frontend. Missing variables expand to empty (standard for such contexts).
// It is a cheap no-op when the input contains no $.
func expand(s string, env map[string]string) (string, error) {
	if s == "" || !strings.Contains(s, "$") {
		return s, nil
	}
	lex := shell.NewLex('\\')
	result, _, err := lex.ProcessWord(s, mapEnvGetter(env))
	if err != nil {
		return "", err
	}
	return result, nil
}

// splitEnv splits a "KEY=val" entry from an image Env slice. The value may contain =.
func splitEnv(e string) (k, v string) {
	if idx := strings.IndexByte(e, '='); idx >= 0 {
		return e[:idx], e[idx+1:]
	}
	return e, ""
}

// BuildWithDockerUI is the high-level entry used by frontend/build.go.
// It leverages dockerui for platforms/args/context and calls ToLLB.
func BuildWithDockerUI(ctx context.Context, dc *dockerui.Client, y *v1alpha1.Yamlfile, target string, c gwclient.Client) (map[string]Result, error) {
	// For MVP we take first (or requested) platform
	plats := dc.TargetPlatforms
	if len(plats) == 0 {
		plats = []ocispecs.Platform{{OS: "linux", Architecture: "amd64"}}
	}
	opt := ConvertOpt{
		BuildArgs: dc.BuildArgs,
		Platform:  &plats[0],
	}
	if mc, err := dc.MainContext(ctx); err == nil && mc != nil {
		opt.Context = *mc
	}

	// Provide a loader that will be called *inside* ToLLB only for scripts belonging
	// to the reachable targets for the chosen `target`. This prevents failing a
	// multi-target build because of a missing script in an unrelated target.
	opt.ScriptLoader = func(path string) ([]byte, error) {
		return loadFileFromContext(ctx, dc, c, path)
	}

	return ToLLB(ctx, y, target, opt)
}

// loadFileFromContext mirrors dockerui ReadEntrypoint logic but for arbitrary context file (used for scripts).
func loadFileFromContext(ctx context.Context, dc *dockerui.Client, c gwclient.Client, filename string) ([]byte, error) {
	// Use a restricted local for just this file + dockerignore sibling if present.
	sessionID := dc.BuildOpts().SessionID
	lsrc := llb.Local(dockerui.DefaultLocalNameContext, llb.FollowPaths([]string{filename, filename + ".dockerignore"}), llb.SessionID(sessionID), llb.SharedKeyHint("context"))
	def, err := lsrc.Marshal(ctx, llb.WithCaps(dc.BuildOpts().Caps))
	if err != nil {
		return nil, err
	}
	res, err := c.Solve(ctx, gwclient.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, err
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, err
	}
	return ref.ReadFile(ctx, gwclient.ReadRequest{Filename: filename})
}
