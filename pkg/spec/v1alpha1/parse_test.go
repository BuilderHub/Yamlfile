package v1alpha1

import (
	"testing"
)

func TestLoad_Basic(t *testing.T) {
	y, err := Load([]byte(`
apiVersion: v1alpha1
targets:
  foo:
    from: alpine
    steps:
      - run:
          command: echo hi
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if y.APIVersion != "v1alpha1" {
		t.Errorf("want v1alpha1, got %s", y.APIVersion)
	}
	if _, ok := y.Targets["foo"]; !ok {
		t.Error("missing target foo")
	}
}

func TestLoad_RejectsBadVersion(t *testing.T) {
	_, err := Load([]byte(`apiVersion: v2
targets: {a: {from: scratch}}`))
	if err == nil || !contains(err.Error(), "unsupported apiVersion") {
		t.Errorf("expected unsupported apiVersion error, got %v", err)
	}
}

func TestLoad_DefaultsAPIVersion(t *testing.T) {
	y, err := Load([]byte(`targets: {a: {from: scratch}}`))
	if err != nil {
		t.Fatal(err)
	}
	if y.APIVersion != "v1alpha1" {
		t.Errorf("expected defaulted v1alpha1, got %s", y.APIVersion)
	}
}

func TestLoad_ExtensionsPreserved(t *testing.T) {
	y, err := Load([]byte(`
apiVersion: v1alpha1
targets:
  t:
    from: alpine
    futureKey: value
    steps:
      - run:
          command: echo
          futureRun: 123
`))
	if err != nil {
		t.Fatal(err)
	}
	if y.Targets["t"].Extensions == nil {
		t.Error("expected target extensions")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && containsHelper(s, sub)))
}
func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestLoad_AllowsArgAndWorkdirSteps(t *testing.T) {
	y, err := Load([]byte(`
apiVersion: v1alpha1
targets:
  t:
    from: alpine
    steps:
      - arg:
          vars:
            FOO: "1"
      - workdir:
          path: /app
      - env:
          vars:
            BAR: "2"
      - run:
          command: echo ok
          workdir: /app/src
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(y.Targets["t"].Steps) != 4 {
		t.Errorf("expected 4 steps, got %d", len(y.Targets["t"].Steps))
	}
}

func TestLoad_AllowsLabelAndEntrypointSteps(t *testing.T) {
	y, err := Load([]byte(`
apiVersion: v1alpha1
targets:
  t:
    from: scratch
    steps:
      - label:
          vars:
            org.opencontainers.image.title: yamlfile
      - entrypoint:
          command: "/bin/yamlfile-frontend"
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	steps := y.Targets["t"].Steps
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Label == nil || steps[0].Label.Vars["org.opencontainers.image.title"] != "yamlfile" {
		t.Errorf("unexpected label step: %+v", steps[0].Label)
	}
	if steps[1].Entrypoint == nil || steps[1].Entrypoint.Command != "/bin/yamlfile-frontend" {
		t.Errorf("unexpected entrypoint step: %+v", steps[1].Entrypoint)
	}
}

func TestLoad_RejectsEntrypointWithMultipleForms(t *testing.T) {
	_, err := Load([]byte(`
apiVersion: v1alpha1
targets:
  t:
    from: scratch
    steps:
      - entrypoint:
          command: "/bin/foo"
          inline: echo foo
`))
	if err == nil || !contains(err.Error(), "exactly one of") {
		t.Errorf("expected exactly one of error, got %v", err)
	}
}

func TestLoad_RejectsMixedStepKinds(t *testing.T) {
	_, err := Load([]byte(`
apiVersion: v1alpha1
targets:
  t:
    from: alpine
    steps:
      - run: {command: "true"}
        env: {vars: {X: "1"}}
`))
	if err == nil || !contains(err.Error(), "exactly one of") {
		t.Errorf("expected 'exactly one of' error, got %v", err)
	}
}
