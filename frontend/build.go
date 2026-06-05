//revive:disable:package-comments
package frontend

// Package frontend implements the BuildKit gateway frontend for Yamlfile (v1alpha1).

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/builderhub/yamlfile/pkg/convert"
	spec "github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
	"github.com/moby/buildkit/frontend/dockerui"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// Build is the BuildKit gateway BuildFunc for the yamlfile frontend (v1alpha1).
func Build(ctx context.Context, c gwclient.Client) (*gwclient.Result, error) {
	dc, err := dockerui.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("dockerui client: %w", err)
	}

	src, err := dc.ReadEntrypoint(ctx, "yamlfile")
	if err != nil {
		return nil, fmt.Errorf("read Yamlfile (use -f Yamlfile.yaml or pass --build-arg BUILDKIT_SYNTAX=...): %w", err)
	}

	y, err := spec.Load(src.Data)
	if err != nil {
		return nil, fmt.Errorf("parse v1alpha1 Yamlfile: %w", err)
	}

	// target from opt, or "default" target if present, or the single target, else error.
	target := dc.Target
	if target == "" {
		if _, hasDefault := y.Targets["default"]; hasDefault {
			target = "default"
		} else if len(y.Targets) == 1 {
			for k := range y.Targets {
				target = k
				break
			}
		} else {
			return nil, fmt.Errorf("multiple targets defined in Yamlfile; specify --target NAME (or define a target named \"default\")")
		}
	}

	results, err := convert.BuildWithDockerUI(ctx, dc, y, target, c)
	if err != nil {
		return nil, fmt.Errorf("convert target %s: %w", target, err)
	}

	r, ok := results[target]
	if !ok {
		return nil, fmt.Errorf("internal error: no result produced for target %q", target)
	}
	if r.State.Output() == nil {
		// fallback empty
		return gwclient.NewResult(), nil
	}

	def, err := r.State.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("marshal llb: %w", err)
	}

	sr, err := c.Solve(ctx, gwclient.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, err
	}

	// Attach image config for the chosen target (single-plat MVP)
	if r.Image != nil {
		dt, err := json.Marshal(r.Image)
		if err != nil {
			return nil, fmt.Errorf("marshal image config for target %s: %w", target, err)
		}
		sr.AddMeta("containerimage.config", dt)
	}

	// TODO: dc.HandleSubrequest for outline/lint/targets etc.
	// TODO: multi-plat via dc.Build(...) + rb.Finalize() + per-plat metadata

	return sr, nil
}
