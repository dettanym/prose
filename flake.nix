{
  description = "Development environment for prose";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
    pnpm2nix-nzbr.url = "github:nzbr/pnpm2nix-nzbr";
    pnpm2nix-nzbr.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs =
    {
      self,
      nixpkgs,
      systems,
      treefmt-nix,
      pnpm2nix-nzbr,
    }:
    let
      eachSystem = f: nixpkgs.lib.genAttrs (import systems) (system: f nixpkgs.legacyPackages.${system});
      treefmtEval = eachSystem (pkgs: treefmt-nix.lib.evalModule pkgs ./treefmt.nix);

      pnpmPackages = eachSystem (
        pkgs:
        let
          inherit (pnpm2nix-nzbr.packages.${pkgs.system}) mkPnpmPackage;
        in
        {
          tsx = mkPnpmPackage rec {
            pname = "tsx";
            version = "4.7.1";

            src = pkgs.fetchFromGitHub {
              owner = "privatenumber";
              repo = pname;
              rev = "v${version}";
              hash = "sha256-9+qQmDZ0WxO4IXVPA6IjhjPbBlhU9yIWk9FpUamMYVM=";
            };

            distDir = ".";
            installInPlace = true;
            extraBuildInputs = [ pkgs.makeWrapper ];

            postInstall = ''
              makeWrapper ${pkgs.nodejs}/bin/node $out/bin/${pname} \
                --add-flags $out/dist/cli.cjs
            '';
          };
        }
      );
    in
    {

      packages.x86_64-linux.hello = nixpkgs.legacyPackages.x86_64-linux.hello;

      packages.x86_64-linux.default = self.packages.x86_64-linux.hello;

      devShells.x86_64-linux.default = nixpkgs.legacyPackages.x86_64-linux.mkShell {

        packages = with nixpkgs.legacyPackages.x86_64-linux; [
          black
          curl
          docker
          envsubst
          fluxcd
          getopt
          git
          go
          go-task
          hostname
          isort
          istioctl
          jq
          kubectl
          minikube
          neovim
          nodejs
          nodePackages.pnpm
          nodePackages.prettier
          open-policy-agent
          protobuf_25
          protoc-gen-go
          protoc-gen-go-grpc
          (python3.withPackages (ps: [
            ps.matplotlib
            ps.numpy
          ]))
          ripgrep
          pnpmPackages.x86_64-linux.tsx
          vegeta
          yq-go
          zstd
        ];

        shellHook = ''
          alias v='ls -alhF --color'
          alias kc='kubectl'
        '';
      };

      # for `nix fmt`
      formatter = eachSystem (pkgs: treefmtEval.${pkgs.system}.config.build.wrapper);

      # for `nix flake check`
      checks = eachSystem (pkgs: {
        formatting = treefmtEval.${pkgs.system}.config.build.check self;
      });
    };
}
