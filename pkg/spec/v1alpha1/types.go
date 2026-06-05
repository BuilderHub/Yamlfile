// Package v1alpha1 defines the Yamlfile v1alpha1 API types for the BuildKit frontend.
//
//revive:disable:package-comments
package v1alpha1

// Yamlfile is the top-level document for apiVersion: v1alpha1.
// Designed for extensibility: unknown fields are captured in Extensions
// (additive features will not break existing documents).
type Yamlfile struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind,omitempty"`
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

// Step is a discriminated union for pipeline steps. New kinds can be added
// without breaking old parsers (unknown Kind will error in v1alpha1 but is captured).
type Step struct {
	Run  *RunSpec  `yaml:"run,omitempty"`
	Copy *CopySpec `yaml:"copy,omitempty"`
	Env  *EnvSpec  `yaml:"env,omitempty"`
	// Add Arg, Workdir, User, Label, Entrypoint, Cmd, Expose etc. as *XXXSpec
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
