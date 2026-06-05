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
            echo "Run docker buildx from the yamlfile/ directory (or use make):"
            echo "  docker buildx build -f cmd/yamlfile-frontend/Dockerfile -t localhost:5000/yamlfile:dev --load ."
            echo "  docker buildx build -f examples/minimal.Yamlfile --build-arg BUILDKIT_SYNTAX=localhost:5000/yamlfile:dev --output type=local,dest=/tmp/out ."
            echo "Or: make docker-build"
            echo "Docs: make docs-serve (serves at http://localhost:1313/) or nix develop --command hugo server -s docs --baseURL http://localhost:1313/"
          '';
        };
      });
}
