//revive:disable:package-comments
package v1alpha1

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load parses a v1alpha1 Yamlfile document from bytes.
// It enforces apiVersion and performs minimal structural validation.
// Unknown fields are retained via inline extensions maps for forward compatibility.
func Load(data []byte) (*Yamlfile, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty Yamlfile")
	}

	var y Yamlfile
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(false) // allow extensions / future fields; we validate known surface ourselves
	if err := dec.Decode(&y); err != nil {
		return nil, fmt.Errorf("yaml decode: %w", err)
	}

	if y.APIVersion == "" {
		// default for convenience in early v1alpha1 (still require explicit in strict mode?)
		y.APIVersion = "v1alpha1"
	}
	if y.APIVersion != "v1alpha1" {
		return nil, fmt.Errorf("unsupported apiVersion %q (this frontend supports v1alpha1)", y.APIVersion)
	}

	if len(y.Targets) == 0 {
		return nil, fmt.Errorf("no targets defined (at least one target is required)")
	}

	// Basic name validation (reserved words, dupes already prevented by map)
	for name := range y.Targets {
		if name == "" || strings.Contains(name, ":") || strings.Contains(name, "/") {
			return nil, fmt.Errorf("invalid target name %q (no empty, no ':' or '/' )", name)
		}
	}

	// Step validation: each step must declare exactly one kind (run/copy/env/arg/workdir).
	// This prevents ambiguous or empty steps at parse time (Go struct allows multiple
	// because of how yaml unmarshal + our discriminated union works).
	for tname, t := range y.Targets {
		for i, s := range t.Steps {
			kinds := 0
			if s.Run != nil {
				kinds++
			}
			if s.Copy != nil {
				kinds++
			}
			if s.Env != nil {
				kinds++
			}
			if s.Arg != nil {
				kinds++
			}
			if s.Workdir != nil {
				kinds++
			}
			if kinds != 1 {
				return nil, fmt.Errorf("target %q step %d must specify exactly one of: run, copy, env, arg, or workdir", tname, i)
			}
		}
	}

	// TODO(v1): deeper validation (cycles live in convert/graph; circular file refs, missing builds, etc.)

	return &y, nil
}

// MustLoad is for tests.
func MustLoad(data []byte) *Yamlfile {
	y, err := Load(data)
	if err != nil {
		panic(err)
	}
	return y
}
