{ pkgs, options, ... }:
{
  # Used to find the project root
  projectRootFile = "flake.nix";

  settings.global.excludes = [ ".archive/**" ];

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
    overrides = [
      {
        # Note: need to add `*(../)` at the beginning, to match using any
        # patterns in prettier settings. That is because the file path expands
        # to a relative value with `../` at the beginning, and in micromatch
        # extglob (`**/`) does not match against `../`.
        files = [ "*(../)**/evaluation/results.md" ];
        options = {
          proseWrap = "always";
        };
      }
    ];
  };
  programs.prettier.excludes = [
    "pnpm-lock.yaml"
    "charts/*/templates/*.yaml"
  ] ++ options.programs.biome.includes.default;

  programs.shellcheck.enable = true;
  programs.shfmt.enable = true;
}
