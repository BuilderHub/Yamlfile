package convert

import (
	"strings"
	"testing"

	spec "github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
)

func TestGraph_BasicDepsAndReachable(t *testing.T) {
	y := &spec.Yamlfile{
		Targets: map[string]spec.TargetSpec{
			"base":  {From: "alpine"},
			"build": {From: "base"},
			"final": {From: "build"},
			"other": {From: "alpine"},
		},
	}
	deps, _ := buildDependencyGraph(y)
	if len(deps["build"]) != 1 || deps["build"][0] != "base" {
		t.Errorf("build deps: %v", deps["build"])
	}

	r, _ := reachableTargets(y, "final", deps)
	if !containsAll(r, []string{"base", "build", "final"}) || containsStr(r, "other") {
		t.Errorf("reachable for final: %v", r)
	}
}

func TestGraph_Cycle(t *testing.T) {
	deps := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}
	if err := validateNoCycles(deps); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle error, got %v", err)
	}
}

func TestGraph_ParallelRoots(t *testing.T) {
	y := &spec.Yamlfile{Targets: map[string]spec.TargetSpec{
		"d1": {From: "alpine"},
		"d2": {From: "alpine"},
		"b":  {From: "d1"},
	}}
	deps, _ := buildDependencyGraph(y)
	r, _ := reachableTargets(y, "", deps)
	roots := parallelRoots(r, deps)
	if len(roots) != 2 {
		t.Errorf("expected 2 parallel roots (d1,d2), got %v", roots)
	}
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
func containsAll(list, need []string) bool {
	for _, n := range need {
		if !containsStr(list, n) {
			return false
		}
	}
	return true
}

func TestResolveBase_CrossFileAndImages(t *testing.T) {
	targets := map[string]spec.TargetSpec{
		"base": {From: "alpine"},
		"app":  {From: "base"},
	}
	builds := map[string]spec.BuildRef{
		"torch": {File: "torch/Yamlfile", Target: "base"},
		"other": {File: "sub/Yamlfile"},
	}

	// local sibling (no :)
	isT, isC, k := resolveBase("base", targets, builds)
	if !isT || isC || k != "base" {
		t.Errorf("local sibling: got %v,%v,%q", isT, isC, k)
	}

	// cross via builds: (has : and lhs matches builds key) — should be detected as cross, not local or image
	isT, isC, k = resolveBase("torch:base", targets, builds)
	if isT || !isC || k != "torch:base" {
		t.Errorf("cross torch:base: got %v,%v,%q", isT, isC, k)
	}
	isT, isC, k = resolveBase("other:foo", targets, builds)
	if isT || !isC || k != "other:foo" {
		t.Errorf("cross other:foo: got %v,%v,%q", isT, isC, k)
	}

	// image refs (with : but lhs not a build key; or no :)
	isT, isC, k = resolveBase("golang:1.25", targets, builds)
	if isT || isC || k != "golang:1.25" {
		t.Errorf("image golang:tag: got %v,%v,%q", isT, isC, k)
	}
	isT, isC, k = resolveBase("alpine", targets, builds)
	if isT || isC || k != "alpine" {
		t.Errorf("image bare: got %v,%v,%q", isT, isC, k)
	}
	isT, isC, k = resolveBase("reg.io/foo/bar:v1", targets, builds)
	if isT || isC || k != "reg.io/foo/bar:v1" {
		t.Errorf("image fq: got %v,%v,%q", isT, isC, k)
	}

	// scratch
	isT, isC, k = resolveBase("scratch", targets, builds)
	if isT || isC || k != "scratch" {
		t.Errorf("scratch: got %v,%v,%q", isT, isC, k)
	}

	// unknown cross (lhs not declared in builds) is treated as image (will fail later at use if intended as cross)
	isT, isC, k = resolveBase("missing:bar", targets, builds)
	if isT || isC || k != "missing:bar" {
		t.Errorf("unknown-colon: got %v,%v,%q (should be image ref)", isT, isC, k)
	}
}
