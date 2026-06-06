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
