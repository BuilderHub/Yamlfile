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
