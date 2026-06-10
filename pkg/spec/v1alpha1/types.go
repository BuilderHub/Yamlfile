// Package v1alpha1 defines the Yamlfile v1alpha1 API types for the BuildKit frontend.
//
//revive:disable:package-comments
package v1alpha1

// Yamlfile is the top-level document for apiVersion: v1alpha1.
// Designed for extensibility: unknown fields are captured in Extensions
// (additive features will not break existing documents).
type Yamlfile struct {
	APIVersion string                 `yaml:"apiVersion"`
	Defaults   *Defaults              `yaml:"defaults,omitempty"`
	Targets    map[string]TargetSpec  `yaml:"targets"`
	Builds     map[string]BuildRef    `yaml:"builds,omitempty"`  // multi-file orchestration
	Secrets    map[string]string      `yaml:"secrets,omitempty"` // id -> description (docs/lint)
	Extensions map[string]interface{} `yaml:",inline"`           // forward compat
}

// Defaults holds document-level defaults (extensible).
type Defaults struct {
	Platform   string                 `yaml:"platform,omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}

// TargetSpec describes one buildable target (like a named multi-stage image or artifact).
type TargetSpec struct {
	From     string `yaml:"from,omitempty"` // image ref or sibling target name (or "component:target")
	Platform string `yaml:"platform,omitempty"`
	Steps    []Step `yaml:"steps,omitempty"`
	// Future: args, secrets (inherited), etc.
	Extensions map[string]interface{} `yaml:",inline"`
}

// BuildRef references another Yamlfile (for top-level multi-file coordination + cross copies).
type BuildRef struct {
	File       string                 `yaml:"file"`             // path relative to build context
	Target     string                 `yaml:"target,omitempty"` // specific target inside that file (default: first or "default")
	Extensions map[string]interface{} `yaml:",inline"`
}

// Step is a discriminated union for pipeline steps. New step types can be added
// without breaking old parsers (unknown step kinds will error in v1alpha1 but are captured
// via the Extensions map).
type Step struct {
	Run        *RunSpec        `yaml:"run,omitempty"`
	Copy       *CopySpec       `yaml:"copy,omitempty"`
	Env        *EnvSpec        `yaml:"env,omitempty"`
	Arg        *ArgSpec        `yaml:"arg,omitempty"`
	Workdir    *WorkdirSpec    `yaml:"workdir,omitempty"`
	Label      *LabelSpec      `yaml:"label,omitempty"`
	Entrypoint *EntrypointSpec `yaml:"entrypoint,omitempty"`
	// Future: User, Cmd, Expose etc. as *XXXSpec
	Extensions map[string]interface{} `yaml:",inline"`
}

// RunSpec supports the key "baked-in" features: inline shell or script-from-file
// (frontend loads the script bytes and mounts it; user does not write a copy step).
type RunSpec struct {
	Command string            `yaml:"command,omitempty"` // single line or sh -c form
	Inline  string            `yaml:"inline,omitempty"`  // | multi-line shell
	Script  string            `yaml:"script,omitempty"`  // path in context; frontend injects via Mkfile+mount
	Env     map[string]string `yaml:"env,omitempty"`
	Secrets []SecretMount     `yaml:"secrets,omitempty"`
	Workdir string            `yaml:"workdir,omitempty"` // per-run working directory (transient; use workdir: step for persistent)
	// mounts, network, security, etc. added later (additive)
	Extensions map[string]interface{} `yaml:",inline"`
}

// SecretMount supports secure mounts as file (target) or env (env).
// Mirrors BuildKit --mount=type=secret,id=...,env=... exactly.
type SecretMount struct {
	ID         string                 `yaml:"id"`
	Target     string                 `yaml:"target,omitempty"` // file dest, e.g. /run/secrets/foo (default /run/secrets/<base(id)>)
	Env        string                 `yaml:"env,omitempty"`    // inject as env var inside the run (no file)
	Optional   bool                   `yaml:"optional,omitempty"`
	Mode       *int                   `yaml:"mode,omitempty"`
	UID        *int                   `yaml:"uid,omitempty"`
	GID        *int                   `yaml:"gid,omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}

// CopySpec supports copying from context or other targets (including cross-file via builds).
type CopySpec struct {
	From string   `yaml:"from,omitempty"` // target name, "component:target", or empty=context
	Src  []string `yaml:"src,omitempty"`
	Dest string   `yaml:"dest"`
	// chown, chmod, parents, exclude, link etc. later
	Extensions map[string]interface{} `yaml:",inline"`
}

// EnvSpec is a convenience step (also doable via run.env, but clearer at target level).
type EnvSpec struct {
	Vars       map[string]string      `yaml:"vars,omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}

// ArgSpec declares build-time variables (the equivalent of Dockerfile ARG).
// The value in the map is the default; it is overridden by a matching --build-arg
// supplied at build time. Arg values participate in $VAR / ${VAR} expansion for
// later env:, arg:, workdir:, and run.env values within the same target, and are
// injected into the execution environment of subsequent run steps (but do not
// appear in the final image ENV unless also set via an env: step).
type ArgSpec struct {
	Vars       map[string]string      `yaml:"vars,omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}

// WorkdirSpec sets the persistent working directory for the target (affects the
// llb.State for subsequent steps and the exported image config.WorkingDir).
// Per-run workdir can be set via run.workdir (does not persist after the run).
type WorkdirSpec struct {
	Path       string                 `yaml:"path"`
	Extensions map[string]interface{} `yaml:",inline"`
}

// LabelSpec sets OCI image config labels (the equivalent of Dockerfile LABEL).
type LabelSpec struct {
	Vars       map[string]string      `yaml:"vars,omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}

// EntrypointSpec sets the image config entrypoint (Dockerfile ENTRYPOINT).
// Uses the same invocation fields as RunSpec (command / inline / script).
// For script, the content is baked into the final image (unlike run.script,
// which is a build-time-only mount that does not persist in layers).
type EntrypointSpec struct {
	Command    string                 `yaml:"command,omitempty"` // exec-form argv (shlex-split; no /bin/sh -c)
	Inline     string                 `yaml:"inline,omitempty"`  // shell-form entrypoint
	Script     string                 `yaml:"script,omitempty"`  // baked into image as the entrypoint executable
	Extensions map[string]interface{} `yaml:",inline"`
}
