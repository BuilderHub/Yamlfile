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
