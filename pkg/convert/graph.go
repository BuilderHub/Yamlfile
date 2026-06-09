package convert

import (
	"fmt"
	"strings"

	spec "github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
)

// buildDependencyGraph returns for each target the list of targets it directly depends on
// (from "from:" and any step that has a "from" reference like copy.from).
// Cross-file "comp:target" refs (via builds:) are intentionally *not* included here;
// they are external and resolved (or errored) at ToLLB time.
func buildDependencyGraph(y *spec.Yamlfile) (map[string][]string, error) {
	deps := make(map[string][]string, len(y.Targets))
	for name, t := range y.Targets {
		d := collectDirectDeps(t, y.Targets, y.Builds)
		deps[name] = d
	}
	return deps, nil
}

// resolveBase classifies a "from:" / "copy.from" value.
// - Local sibling target (exact key in targets, and target names forbid ":") => isLocalTarget=true
// - "comp:target" where "comp" is a key in the builds: map => isCrossBuild=true (caller must treat as external)
// - Everything else (bare name, "reg:tag", "alpine", "scratch") => image ref (or special)
// Target names are validated to contain no ":" so a ref containing ":" is either image or cross-build.
func resolveBase(ref string, targets map[string]spec.TargetSpec, builds map[string]spec.BuildRef) (isLocalTarget, isCrossBuild bool, keyOrRef string) {
	if ref == "" {
		return false, false, ""
	}
	if ref == "scratch" {
		return false, false, "scratch"
	}
	if _, ok := targets[ref]; ok {
		return true, false, ref
	}
	if idx := strings.Index(ref, ":"); idx > 0 && idx < len(ref)-1 {
		comp := ref[:idx]
		if builds != nil {
			if _, ok := builds[comp]; ok {
				return false, true, ref
			}
		}
	}
	return false, false, ref
}

func collectDirectDeps(t spec.TargetSpec, targets map[string]spec.TargetSpec, builds map[string]spec.BuildRef) []string {
	var out []string
	if t.From != "" {
		if isT, _, k := resolveBase(t.From, targets, builds); isT {
			out = append(out, k)
		}
	}
	for _, s := range t.Steps {
		if s.Copy != nil && s.Copy.From != "" {
			if isT, _, k := resolveBase(s.Copy.From, targets, builds); isT {
				out = append(out, k)
			}
		}
		// run mounts from= later
	}
	return unique(out)
}

func unique(in []string) []string {
	m := make(map[string]struct{}, len(in))
	var out []string
	for _, v := range in {
		if _, ok := m[v]; !ok {
			m[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// reachableTargets returns the transitive closure of targets needed for the given target (or all if "").
func reachableTargets(y *spec.Yamlfile, target string, deps map[string][]string) ([]string, error) {
	if target == "" {
		// default: all (or pick a conventional "default" / last?); for v1alpha1 return all in alpha order isn't required
		all := make([]string, 0, len(y.Targets))
		for n := range y.Targets {
			all = append(all, n)
		}
		return all, nil
	}
	visited := map[string]bool{}
	var order []string
	var dfs func(string) error
	dfs = func(n string) error {
		if visited[n] {
			return nil
		}
		visited[n] = true
		for _, d := range deps[n] {
			if _, ok := y.Targets[d]; !ok {
				// d is not a local sibling target (it is either a cross-file "comp:target"
				// reference declared via builds: or an external image ref). Cross-file refs
				// are intentionally never added to the local dep graph (see collectDirectDeps)
				// and produce clear errors later in ToLLB when encountered on reachable targets.
				continue
			}
			if err := dfs(d); err != nil {
				return err
			}
		}
		order = append(order, n)
		return nil
	}
	if err := dfs(target); err != nil {
		return nil, err
	}
	return order, nil
}

// validateNoCycles does a simple DFS cycle check.
func validateNoCycles(deps map[string][]string) error {
	color := map[string]int{} // 0 white, 1 gray, 2 black
	var dfs func(string) error
	dfs = func(n string) error {
		color[n] = 1 // gray
		for _, d := range deps[n] {
			c := color[d]
			if c == 1 {
				return fmt.Errorf("cycle involving %s -> %s", n, d)
			}
			if c == 0 {
				if err := dfs(d); err != nil {
					return err
				}
			}
		}
		color[n] = 2
		return nil
	}
	for n := range deps {
		if color[n] == 0 {
			if err := dfs(n); err != nil {
				return err
			}
		}
	}
	return nil
}

// parallelRoots returns targets in the reachable set that have no (remaining) unsatisfied deps
// inside the set. The caller can launch them concurrently.
func parallelRoots(reachable []string, deps map[string][]string) []string {
	inDegree := map[string]int{}
	for _, n := range reachable {
		inDegree[n] = 0
	}
	for _, n := range reachable {
		for _, d := range deps[n] {
			if _, ok := inDegree[d]; ok {
				inDegree[n]++
			}
		}
	}
	var roots []string
	for _, n := range reachable {
		if inDegree[n] == 0 {
			roots = append(roots, n)
		}
	}
	return roots
}
