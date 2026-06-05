{
  description = "yamlfile - BuildKit frontend for intuitive, parallel, secret-aware multi-target Yamlfiles (v1alpha1)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          default = pkgs.hello; # placeholder; real image built via Makefile + docker buildx
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.go-tools
            pkgs.golangci-lint
            pkgs.revive
            pkgs.hugo
            pkgs.delve
            pkgs.docker
            pkgs.docker-buildx
            pkgs.skopeo
            pkgs.jq
          ];
          shellHook = ''
            echo "yamlfile dev shell (Go $(go version))"
            echo "IMPORTANT: run docker buildx commands from the *BuilderHub repo root* (not inside yamlfile/)"
            echo "  docker buildx build -f yamlfile/cmd/yamlfile-frontend/Dockerfile -t localhost:5000/yamlfile:dev --load ."
            echo "  docker buildx build -f yamlfile/examples/minimal.Yamlfile --build-arg BUILDKIT_SYNTAX=localhost:5000/yamlfile:dev --output type=local,dest=/tmp/out ."
            echo "Or use 'make -C yamlfile docker-build' (handles context)."
            echo "Docs: make docs-serve (or nix develop --command hugo server -s docs)"
          '';
        };
      });
}
