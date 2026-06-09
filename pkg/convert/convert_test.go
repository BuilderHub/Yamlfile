package convert

import (
	"context"
	"strings"
	"testing"

	spec "github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// TestExpand_Basic covers the internal expander used for env/arg/workdir values.
func TestExpand_Basic(t *testing.T) {
	env := map[string]string{
		"FOO":  "bar",
		"BASE": "/app",
		"PATH": "/usr/bin:/bin",
	}
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"$FOO", "bar"},
		{"${FOO}", "bar"},
		{"${BASE}/bin", "/app/bin"},
		{"/x:${PATH}", "/x:/usr/bin:/bin"},
		{"$UNSET", ""},     // missing -> empty (Dockerfile-like)
		{"${UNSET}", ""},
	}
	for _, c := range cases {
		got, err := expand(c.in, env)
		if err != nil {
			t.Errorf("expand(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestToLLB_ArgEnvWorkdir_Basic exercises the new surface through ToLLB (no real buildkit solve).
// We supply a no-op ScriptLoader and verify we get a result without error and that the
// final image config for the leaf target contains the expanded env/workdir effects.
func TestToLLB_ArgEnvWorkdir_Basic(t *testing.T) {
	y := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Targets: map[string]spec.TargetSpec{
			"base": {
				From: "alpine:3.19",
				Steps: []spec.Step{
					{Arg: &spec.ArgSpec{Vars: map[string]string{"VER": "1.2.3"}}},
					{Env: &spec.EnvSpec{Vars: map[string]string{"APP": "demo-${VER}"}}},
					{Workdir: &spec.WorkdirSpec{Path: "/work"}},
					{Run: &spec.RunSpec{Command: "echo ${APP} > /marker.txt", Workdir: "/work/src"}},
				},
			},
		},
	}

	opt := ConvertOpt{
		BuildArgs: map[string]string{"VER": "9.9.9"}, // CLI should win over the arg default
		Platform:  &ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		ScriptLoader: func(string) ([]byte, error) {
			return nil, nil
		},
	}

	res, err := ToLLB(context.Background(), y, "base", opt)
	if err != nil {
		t.Fatalf("ToLLB: %v", err)
	}
	r, ok := res["base"]
	if !ok {
		t.Fatal("no result for base")
	}
	if r.Image == nil {
		t.Fatal("no image config")
	}

	// The CLI build arg should have produced APP=demo-9.9.9 (via env after arg resolution)
	foundApp := false
	for _, e := range r.Image.Config.Env {
		if strings.HasPrefix(e, "APP=") {
			if !strings.Contains(e, "9.9.9") {
				t.Errorf("expected APP to contain overridden 9.9.9, got %s", e)
			}
			foundApp = true
		}
	}
	if !foundApp {
		t.Errorf("APP not found in final envs: %v", r.Image.Config.Env)
	}

	// The last workdir step should have set the persistent WorkingDir.
	if r.Image.Config.WorkingDir != "/work" {
		t.Errorf("expected WorkingDir=/work, got %q", r.Image.Config.WorkingDir)
	}
}

func TestToLLB_LabelEntrypoint_Basic(t *testing.T) {
	y := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Targets: map[string]spec.TargetSpec{
			"release": {
				From: "scratch",
				Steps: []spec.Step{
					{
						Label: &spec.LabelSpec{
							Vars: map[string]string{
								"org.opencontainers.image.title": "yamlfile",
								"moby.buildkit.frontend.network.none": "true",
							},
						},
					},
					{
						Entrypoint: &spec.EntrypointSpec{
							Command: "/bin/yamlfile-frontend",
						},
					},
				},
			},
		},
	}

	opt := ConvertOpt{
		Platform: &ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		ScriptLoader: func(string) ([]byte, error) {
			return nil, nil
		},
	}

	res, err := ToLLB(context.Background(), y, "release", opt)
	if err != nil {
		t.Fatalf("ToLLB: %v", err)
	}
	r, ok := res["release"]
	if !ok {
		t.Fatal("no result for release")
	}
	if r.Image == nil {
		t.Fatal("no image config")
	}
	if r.Image.Config.Labels["org.opencontainers.image.title"] != "yamlfile" {
		t.Errorf("unexpected labels: %v", r.Image.Config.Labels)
	}
	if len(r.Image.Config.Entrypoint) != 1 || r.Image.Config.Entrypoint[0] != "/bin/yamlfile-frontend" {
		t.Errorf("unexpected entrypoint: %v", r.Image.Config.Entrypoint)
	}
}

// TestToLLB_EntrypointScript verifies that entrypoint.script bakes the script
// content into the image (unlike run.script) and sets the entrypoint accordingly.
func TestToLLB_EntrypointScript(t *testing.T) {
	scriptContent := []byte("#!/bin/sh\necho hello from entrypoint script\nexec \"$@\"\n")

	y := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Targets: map[string]spec.TargetSpec{
			"app": {
				From: "alpine",
				Steps: []spec.Step{
					{
						Entrypoint: &spec.EntrypointSpec{
							Script: "entrypoint.sh",
						},
					},
				},
			},
		},
	}

	opt := ConvertOpt{
		Platform: &ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		ScriptLoader: func(path string) ([]byte, error) {
			if path == "entrypoint.sh" {
				return scriptContent, nil
			}
			return nil, nil
		},
	}

	res, err := ToLLB(context.Background(), y, "app", opt)
	if err != nil {
		t.Fatalf("ToLLB: %v", err)
	}
	r, ok := res["app"]
	if !ok {
		t.Fatal("no result for app")
	}
	if r.Image == nil {
		t.Fatal("no image config")
	}
	// Entrypoint should be set to the baked script path
	if len(r.Image.Config.Entrypoint) != 1 || !strings.HasPrefix(r.Image.Config.Entrypoint[0], "/.yamlfile-entrypoint-") {
		t.Errorf("unexpected entrypoint: %v", r.Image.Config.Entrypoint)
	}
}

// TestToLLB_PlatformOverrides verifies that defaults.platform and per-target platform
// (grammar fields) are wired: target.platform wins, then defaults, then opt fallback.
// We assert on the resulting image config's Platform (the part that is exported).
func TestToLLB_PlatformOverrides(t *testing.T) {
	baseY := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Targets: map[string]spec.TargetSpec{
			"base": {
				From: "alpine:3.19",
				Steps: []spec.Step{
					{Run: &spec.RunSpec{Command: "true"}},
				},
			},
		},
	}

	opt := ConvertOpt{
		Platform: &ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		ScriptLoader: func(string) ([]byte, error) { return nil, nil },
	}

	var gotPlat ocispecs.Platform

	// 1. default opt only
	res, err := ToLLB(context.Background(), baseY, "base", opt)
	if err != nil {
		t.Fatalf("ToLLB (opt only): %v", err)
	}
	gotPlat = res["base"].Image.Platform
	if gotPlat.Architecture != "amd64" {
		t.Errorf("opt fallback: want amd64, got %s", gotPlat.Architecture)
	}

	// 2. defaults.platform
	yDef := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Defaults:   &spec.Defaults{Platform: "linux/arm64"},
		Targets:    baseY.Targets,
	}
	res, err = ToLLB(context.Background(), yDef, "base", opt)
	if err != nil {
		t.Fatalf("ToLLB (defaults): %v", err)
	}
	gotPlat = res["base"].Image.Platform
	if gotPlat.Architecture != "arm64" {
		t.Errorf("defaults.platform: want arm64, got %s", gotPlat.Architecture)
	}

	// 3. per-target overrides defaults
	yTgt := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Defaults:   &spec.Defaults{Platform: "linux/arm64"},
		Targets: map[string]spec.TargetSpec{
			"base": {
				From:     "alpine:3.19",
				Platform: "linux/amd64/v3", // should win
				Steps:    baseY.Targets["base"].Steps,
			},
		},
	}
	res, err = ToLLB(context.Background(), yTgt, "base", opt)
	if err != nil {
		t.Fatalf("ToLLB (target override): %v", err)
	}
	gotPlat = res["base"].Image.Platform
	if gotPlat.Architecture != "amd64" || gotPlat.Variant != "v3" {
		t.Errorf("target.platform: want amd64/v3, got %+v", gotPlat)
	}
}

// TestToLLB_CrossFileRefError ensures that "comp:target" syntax (when comp is declared in builds:)
// produces a clear "not yet supported" error instead of a confusing image pull or "unknown target".
// This completes grammar recognition for the declared multi-file surface.
func TestToLLB_CrossFileRefError(t *testing.T) {
	y := &spec.Yamlfile{
		APIVersion: "v1alpha1",
		Builds: map[string]spec.BuildRef{
			"torch": {File: "torch/Yamlfile", Target: "base"},
		},
		Targets: map[string]spec.TargetSpec{
			"final": {
				From: "torch:base", // cross
				Steps: []spec.Step{
					{Copy: &spec.CopySpec{From: "torch:base", Src: []string{"/x"}, Dest: "/x"}},
				},
			},
		},
	}
	opt := ConvertOpt{
		Platform: &ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		ScriptLoader: func(string) ([]byte, error) { return nil, nil },
	}

	_, err := ToLLB(context.Background(), y, "final", opt)
	if err == nil || !strings.Contains(err.Error(), "cross-file reference") || !strings.Contains(err.Error(), "not yet supported") {
		t.Errorf("expected cross-file not-yet-supported error, got: %v", err)
	}
}

// TestToLLB_InvalidPlatformError verifies that malformed platform strings in the
// grammar (defaults.platform or per-target platform) produce a clear error instead
// of silent fallback. This is required for the platform wiring feature to be safe
// and trustworthy (user-declared intent must not be silently ignored).
func TestToLLB_InvalidPlatformError(t *testing.T) {
	opt := ConvertOpt{
		Platform: &ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		ScriptLoader: func(string) ([]byte, error) { return nil, nil },
	}

	cases := []struct {
		name string
		y    *spec.Yamlfile
		want string // substring expected in error
	}{
		{
			name: "syntactically invalid defaults.platform (incomplete specifier)",
			y: &spec.Yamlfile{
				APIVersion: "v1alpha1",
				Defaults:   &spec.Defaults{Platform: "linux/"}, // known to fail Parse
				Targets: map[string]spec.TargetSpec{
					"t": {From: "scratch", Steps: []spec.Step{{Run: &spec.RunSpec{Command: "true"}}}},
				},
			},
			want: "defaults.platform",
		},
		{
			name: "syntactically invalid target.platform (too many segments)",
			y: &spec.Yamlfile{
				APIVersion: "v1alpha1",
				Defaults:   &spec.Defaults{Platform: "linux/arm64"},
				Targets: map[string]spec.TargetSpec{
					"t": {
						From:     "scratch",
						Platform: "linux/amd64/extra/extra", // cannot parse
						Steps:    []spec.Step{{Run: &spec.RunSpec{Command: "true"}}},
					},
				},
			},
			want: "platform for target t",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ToLLB(context.Background(), c.y, "t", opt)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", c.name)
			}
			if !strings.Contains(err.Error(), "invalid platform") || !strings.Contains(err.Error(), c.want) {
				t.Errorf("expected error containing 'invalid platform' and %q, got: %v", c.want, err)
			}
		})
	}
}
