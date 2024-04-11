{ pkgs, options, ... }:
{
  # Used to find the project root
  projectRootFile = "flake.nix";

  programs.statix.enable = true;
  programs.nixfmt-rfc-style.enable = true;

  programs.ruff.enable = true;
  programs.ruff.format = true;
  programs.isort.enable = true;
  programs.isort.profile = "black";

  programs.biome.enable = true;
  programs.biome.settings = {
    javascript.formatter = {
      indentStyle = "space";
      semicolons = "asNeeded";
    };
    json.formatter = {
      indentStyle = "space";
    };
  };
  programs.prettier.enable = true;
  programs.prettier.settings = {
    semi = false;
  };
  programs.prettier.excludes = [
    "pnpm-lock.yaml"
    "charts/*/templates/*.yaml"
  ] ++ options.programs.biome.includes.default;

  programs.shellcheck.enable = true;
  programs.shfmt.enable = true;
}
