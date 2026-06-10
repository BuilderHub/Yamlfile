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
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
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

	// When the chosen target (or defaults) declares a platform in the Yamlfile we
	// always produce a single-platform result (spec wins; CLI multi will cause the
	// normal "multiple platforms requested but result is not multi-platform" warning).
	// Only when *no* platform is declared for the target do we honor the full CLI
	// --platform list (including multi-arch) using dockerui's Build helper.
	hasSpecPlatform := (y.Defaults != nil && y.Defaults.Platform != "")
	if t, ok := y.Targets[target]; ok && t.Platform != "" {
		hasSpecPlatform = true
	}

	if hasSpecPlatform || len(dc.TargetPlatforms) <= 1 {
		// Single-path (existing behavior for spec-declared platforms or trivial single CLI).
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

		// Attach image config (single-plat path). The effective platform (from grammar
		// or the first CLI platform) was already chosen inside ToLLB / BuildWithDockerUI.
		if r.Image != nil {
			dt, err := json.Marshal(r.Image)
			if err != nil {
				return nil, fmt.Errorf("marshal image config for target %s: %w", target, err)
			}
			sr.AddMeta("containerimage.config", dt)
		}

		// TODO: dc.HandleSubrequest for outline/lint/targets etc.

		return sr, nil
	}

	// Multi-platform path (no platform declared in Yamlfile for this target, >1 CLI platforms).
	// dc.Build iterates TargetPlatforms, calls our func per platform (passing the specific
	// platform as the CLI fallback so ToLLB uses it), and Finalize emits the correct
	// exporter metadata (platforms list + per-plat config keys).
	rb, err := dc.Build(ctx, func(ctx context.Context, platform *ocispecs.Platform, _ int) (*dockerui.BuildResult, error) {
		r, err := convert.BuildOneTargetPlatform(ctx, dc, y, target, platform, c)
		if err != nil {
			return nil, fmt.Errorf("convert target %s: %w", target, err)
		}
		if r.State.Output() == nil {
			// Empty state (unusual for a real target); return a result with no ref.
			return &dockerui.BuildResult{}, nil
		}

		def, err := r.State.Marshal(ctx)
		if err != nil {
			return nil, fmt.Errorf("marshal llb: %w", err)
		}

		rs, err := c.Solve(ctx, gwclient.SolveRequest{Definition: def.ToPB()})
		if err != nil {
			return nil, err
		}
		ref, err := rs.SingleRef()
		if err != nil {
			return nil, err
		}

		return &dockerui.BuildResult{
			Reference: ref,
			Image:     dockerImageFromOCI(r.Image),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("multi-platform build: %w", err)
	}
	return rb.Finalize()
}

// dockerImageFromOCI maps our internal *ocispecs.Image (what ToLLB / emptyImage produce)
// to the *dockerspec.DockerOCIImage expected by dockerui.BuildResult. We only populate
// the fields we actually set (platform + config bits); Docker-specific extensions stay zero.
func dockerImageFromOCI(img *ocispecs.Image) *dockerspec.DockerOCIImage {
	if img == nil {
		return nil
	}
	di := &dockerspec.DockerOCIImage{}
	p := img.Platform
	di.Architecture = p.Architecture
	di.OS = p.OS
	di.OSVersion = p.OSVersion
	di.Variant = p.Variant
	if len(p.OSFeatures) > 0 {
		di.OSFeatures = append([]string{}, p.OSFeatures...)
	}
	di.Config = dockerspec.DockerOCIImageConfig{
		ImageConfig: img.Config,
	}
	return di
}
