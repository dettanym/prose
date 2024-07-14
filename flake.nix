{
  description = "Development environment for prose";

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.treefmt-nix.flakeModule
        inputs.pre-commit-hooks.flakeModule
      ];

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      perSystem =
        # Per-system attributes can be defined here. The self' and inputs'
        # module parameters provide easy access to attributes of the same
        # system. For description of all other function arguments, see
        # `https://flake.parts/module-arguments#persystem-module-parameters`.
        {
          config,
          self',
          inputs',
          pkgs,
          system,
          ...
        }:
        {
          packages.hello = pkgs.hello;
          packages.default = self'.packages.hello;

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
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

            buildInputs = config.pre-commit.settings.enabledPackages;

            shellHook = ''
              ${config.pre-commit.installationScript}

              alias v='ls -alhF --color'
              alias kc='kubectl'
            '';
          };

          # can use via `nix fmt`
          treefmt = {
            # Used to find the project root
            projectRootFile = "flake.nix";
            # pre-commit runs treefmt as a check, so we should not run it again
            flakeCheck = false;
          } // (import ./treefmt.nix { config = config.treefmt; });

          # can use via `nix flake check`
          pre-commit.settings.hooks = {
            regenerate-pre-commit-config = {
              enable = true;
              name = "Regenerate `.pre-commit-config.yaml` file";
              entry =
                let
                  script = pkgs.writeShellScript "pre-commit-config-update.sh" ''
                    CURRENT="$(readlink -sf .pre-commit-config.yaml)"
                    nix develop --offline --quiet >/dev/null 2>&1 -c echo ""
                    NEW="$(readlink -sf .pre-commit-config.yaml)"

                    if [[ "''${CURRENT}" != "''${NEW}" ]]; then
                      echo "Regenerated \`.pre-commit-config.yaml\` file. Need to re-run commit."
                      exit 1
                    fi
                  '';
                in
                script.outPath;
              fail_fast = true;
              always_run = true;
            };
            treefmt.enable = true;
          };
        };
    };

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
    pre-commit-hooks.url = "github:cachix/pre-commit-hooks.nix";
    pre-commit-hooks.inputs.nixpkgs.follows = "nixpkgs";
  };
}
