{
  description = "Development environment for prose";

  outputs =
    {
      self,
      nixpkgs,
      systems,
      treefmt-nix,
      pre-commit-hooks,
    }:
    let
      eachSystem = f: nixpkgs.lib.genAttrs (import systems) (system: f nixpkgs.legacyPackages.${system});
      treefmtEval = eachSystem (pkgs: treefmt-nix.lib.evalModule pkgs ./treefmt.nix);
    in
    {

      packages.x86_64-linux.hello = nixpkgs.legacyPackages.x86_64-linux.hello;

      packages.x86_64-linux.default = self.packages.x86_64-linux.hello;

      devShells.x86_64-linux.default = nixpkgs.legacyPackages.x86_64-linux.mkShell {

        packages = with nixpkgs.legacyPackages.x86_64-linux; [
          curl
          docker
          envsubst
          fluxcd
          getopt
          git
          go
          go-task
          hostname
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
            ps.srsly
          ]))
          pipenv
          ripgrep
          vegeta
          yq-go
          zstd
        ];

        buildInputs = self.checks.x86_64-linux.pre-commit-check.enabledPackages;

        shellHook = ''
          ${self.checks.x86_64-linux.pre-commit-check.shellHook}

          alias v='ls -alhF --color'
          alias kc='kubectl'
        '';
      };

      # for `nix fmt`
      formatter = eachSystem (pkgs: treefmtEval.${pkgs.system}.config.build.wrapper);

      # for `nix flake check`
      checks = eachSystem (pkgs: {
        pre-commit-check = pre-commit-hooks.lib.${pkgs.system}.run {
          src = ./.;
          hooks = {
            treefmt = {
              enable = true;
              package = pkgs.lib.mkBefore treefmtEval.${pkgs.system}.config.build.wrapper;
            };
          };
        };
      });
    };

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
    pre-commit-hooks.url = "github:cachix/pre-commit-hooks.nix";
    pre-commit-hooks.inputs.nixpkgs.follows = "nixpkgs";
  };
}
