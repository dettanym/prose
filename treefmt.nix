{ pkgs, ... }:
{
  # Used to find the project root
  projectRootFile = "flake.nix";

  programs.black.enable = true;
  programs.isort.enable = true;
  programs.isort.profile = "black";

  programs.prettier.enable = true;
  programs.prettier.settings = {
    semi = false;
  };
  programs.prettier.includes = [ "*.mts" ];

  programs.shellcheck.enable = true;
  programs.shfmt.enable = true;
}
