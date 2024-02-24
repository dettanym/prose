#!/usr/bin/env bash

###
# This file attempts to run the command within a nix environment, if tools are
# present. If nix or nix-portable are not present, then this scripts tries to
# run directly, assuming all of the tools are already installed.
#
# To use this, add a new sh script (e.g. in the current folder) and use the
# following line at the top of the script as shebang command. The path to this
# `env.sh` file can be specified relatively to the new file.
# `#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" "$0" "$@"'`
###

set -euo pipefail

args=(
  "nix"
  "develop"
  "--ignore-environment"
  "--keep" "KUBECONFIG"
  "--keep" "HOME"
  "--command" "bash"
)

if [[ -n "${IN_NIX_SHELL+x}" ]]; then
  echo 'already within nix'
  exec "${args[@]}" "$@"
elif command -v nix &>/dev/null; then
  echo 'using nix'
  exec "${args[@]}" "$@"
elif command -v nix-portable &>/dev/null; then
  echo 'using nix-portable'
  exec nix-portable "${args[@]}" "$@"
else
  echo 'trying to run script without nix'
  exec /usr/bin/env bash "$@"
fi
