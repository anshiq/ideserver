{
  description = "VPS platform — persistent Incus LXC containers on a single NixOS host (Go + Postgres)";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        vps = pkgs.buildGoModule {
          pname = "vps";
          version = "0.1.0";
          # Exclude web/ (a separate Go module boundary in its own right, see
          # web/go.mod) so its node_modules never gets copied into the
          # derivation's source.
          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.difference ./. ./web;
          };
          vendorHash = "sha256-pId2KwARlY+NTOQuZGNKP9FqJx1SjcC7EFTL89FMlgo=";
          subPackages = [ "cmd/api" "cmd/hostd" "cmd/edge" "cmd/sshgw" ];
        };
        web = pkgs.buildNpmPackage {
          pname = "vps-web";
          version = "0.1.0";
          src = ./web;
          npmDepsHash = "sha256-GAdFnzqHM8G7rv6Ot9z/SXNdqOBguH25Gx55M+9cliQ=";
          nodejs = pkgs.nodejs_22;
          nativeBuildInputs = [ pkgs.makeWrapper ];
          # next.config.ts sets output:"standalone", so `next build` (run by
          # the default npmBuildScript) already produces a self-contained
          # server at .next/standalone/server.js. buildNpmPackage's default
          # installPhase instead re-runs `npm install`, which drops dotfiles
          # like .next entirely (npm's own file-inclusion default) — so we
          # replace it with a plain copy of the standalone output plus the
          # static assets and public/ it doesn't bundle, and wrap `node
          # server.js` as bin/vps-web (what the NixOS module's ExecStart
          # expects, run with cwd there so its relative asset paths resolve).
          installPhase = ''
            runHook preInstall
            mkdir -p $out/lib/vps-web
            cp -r .next/standalone/. $out/lib/vps-web/
            mkdir -p $out/lib/vps-web/.next
            cp -r .next/static $out/lib/vps-web/.next/static
            if [ -d public ]; then cp -r public $out/lib/vps-web/public; fi
            makeWrapper ${pkgs.nodejs_22}/bin/node $out/bin/vps-web \
              --add-flags "$out/lib/vps-web/server.js" \
              --run "cd $out/lib/vps-web"
            runHook postInstall
          '';
        };
      in {
        packages = { inherit vps web; default = vps; };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            nodejs_22 pnpm
            go_1_25 gopls golangci-lint delve gofumpt
            sqlc goose
            incus incus-client
            postgresql_17
            just direnv gitleaks trivy govulncheck
            openssh
          ];
          shellHook = ''
            export DATABASE_URL="postgres://vps:vps@localhost:5432/vps?sslmode=disable"
            export DOMAIN="example.com"
            echo "VPS dev shell — run 'just' for tasks (postgres only: just deps-up)"
          '';
        };
      }
    ) // {
      nixosModules.default = import ./deploy/nixos/module.nix;
    };
}
