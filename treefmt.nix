{ pkgs, options, ... }:
{
  # Used to find the project root
  projectRootFile = "flake.nix";

  programs.statix.enable = true;
  programs.nixfmt-rfc-style.enable = true;

  programs.black.enable = true;
  programs.isort.enable = true;
  programs.isort.profile = "black";

  programs.biome.enable = true;
  programs.biome.settings = {
    javascript.formatter = {
      indentStyle = "space";
      semicolons = "asNeeded";
    };
  };
  programs.prettier.enable = true;
  programs.prettier.settings = {
    semi = false;
  };
  programs.prettier.excludes = options.programs.biome.includes.default;

  programs.shellcheck.enable = true;
  programs.shfmt.enable = true;
}
