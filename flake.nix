{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, systems, treefmt-nix }:
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
          git
          go
          go-task
          hostname
          istioctl
          jq
          kubectl
          minikube
          neovim
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
