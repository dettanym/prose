#!/usr/bin/env bash

###
# This file attempts to run the command within a nix environment, if tools are
# present. If nix or nix-portable are not present, then this scripts tries to
# run directly, assuming all of the tools are already installed.
#
# To use this, add a new sh script (e.g. in the current folder) and use the
# following line at the top of the script as shebang command. The path to this
# `env.sh` file can be specified relatively to the new file.
# `#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/env.sh" bash "$0" "$@"'`
###

set -euo pipefail

function usage() {
  # TODO: fill in usage details
  #  now if `--` is present in the list of parameters to this script, this env.sh
  #  parses the list to find the program that is being executed and to change
  #  to the folder containing the script. This is needed for npm/pnpm commands
  #  which search folders for `package.json` and `node_modules/`.
  cat <<-EOF
		Usage:
	EOF
}

# based on https://stackoverflow.com/a/39376824
OPTS=$(getopt -o "-h" --long "help" -n "env.sh" -- "$@")
if [[ $? != 0 ]]; then
  echo "Error in command line arguments." >&2
  exit 1
fi

declare -a COMMAND

eval set -- "$OPTS"
while true; do
  case "$1" in
    -h | --help ) usage; exit; ;;
    -- ) shift; break ;;
    * ) COMMAND+=( "$1" ); shift ;;
  esac
done

ORIGINAL_PWD="$(pwd)"
ENV_DIR="$(dirname "$0")"

if [[ "$#" -gt 0 && ("$1" == "./"* || "$1" == "../"* ) ]]; then
  PROGRAM="$1"
  shift

  RELATIVE=$(realpath -s --relative-to="$ENV_DIR" "$ORIGINAL_PWD/$PROGRAM")

  cd "$ENV_DIR"
  COMMAND+=("$RELATIVE" "$ORIGINAL_PWD")
fi

COMMAND+=("$@")

args=(
  "nix"
  "develop"
  "--ignore-environment"
  "--keep" "KUBECONFIG"
  "--keep" "HOME"
  "--keep" "USER"
  "--command"
)

if [[ "${#COMMAND[@]}" -eq 0 ]]; then
  usage
  exit
fi

if [[ -n "${IN_NIX_SHELL+x}" ]]; then
  echo 'already within nix'
  exec /usr/bin/env "${COMMAND[@]}"
elif command -v nix &>/dev/null; then
  echo 'using nix'
  exec "${args[@]}" "${COMMAND[@]}"
elif command -v nix-portable &>/dev/null; then
  echo 'using nix-portable'
  exec nix-portable "${args[@]}" "${COMMAND[@]}"
else
  echo 'trying to run script without nix'
  exec /usr/bin/env "${COMMAND[@]}"
fi
