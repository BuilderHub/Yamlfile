package convert

import (
	"fmt"

	spec "github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
)

// buildDependencyGraph returns for each target the list of targets it directly depends on
// (from "from:" and any step that has a "from" reference like copy.from).
func buildDependencyGraph(y *spec.Yamlfile) (map[string][]string, error) {
	deps := make(map[string][]string, len(y.Targets))
	for name, t := range y.Targets {
		d := collectDirectDeps(t, y.Targets)
		// also handle builds: refs at top level? resolved at use time
		deps[name] = d
	}
	return deps, nil
}

// resolveBase classifies a "from:" / "copy.from" value.
// If it exactly names a target defined in this Yamlfile, it is a sibling target
// (return true + the key). Otherwise it is treated as an image ref for llb.Image
// (this allows both bare names like "alpine" and fully qualified images).
// Multi-file "build:target" refs are left for higher-level handling (they won't
// match a local target key).
func resolveBase(ref string, targets map[string]spec.TargetSpec) (isTarget bool, keyOrRef string) {
	if ref == "" {
		return false, ""
	}
	if ref == "scratch" {
		return false, "scratch"
	}
	if _, ok := targets[ref]; ok {
		return true, ref
	}
	return false, ref
}

func collectDirectDeps(t spec.TargetSpec, targets map[string]spec.TargetSpec) []string {
	var out []string
	if t.From != "" {
		if isT, k := resolveBase(t.From, targets); isT {
			out = append(out, k)
		}
	}
	for _, s := range t.Steps {
		if s.Copy != nil && s.Copy.From != "" {
			if isT, k := resolveBase(s.Copy.From, targets); isT {
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
				// might be "comp:foo" multi-file ref; allow for now (resolved higher)
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
