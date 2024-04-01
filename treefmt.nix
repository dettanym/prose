{ pkgs, ... }:
{
  # Used to find the project root
  projectRootFile = "flake.nix";

  programs.black.enable = true;
  programs.isort.enable = true;
  programs.isort.profile = "black";
}
