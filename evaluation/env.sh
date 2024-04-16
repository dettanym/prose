#!/usr/bin/env bash

###
# Run command within nix environment.
#
# Use either on terminal or as a script in shebang line:
# `./env.sh <command>... [--] <command>... <args>...`
# `#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/env.sh" <command>... "$0" "$@"'`
###

set -euo pipefail

function usage() {
  cat <<-EOF
		Usage: ./env.sh [-h | --help] [[<command>...] --] <command>...
		       ./env.sh [<command>...] -- (./<path> | ../<path>) <args>...
		       #!/usr/bin/env -S bash -c '"\$(dirname \$(readlink -f "\$0"))/env.sh" <command>... "\$0" "\$@"'

		Description:
		    Execute commands in nix environment if possible.

		    This script is intended for use within shebang line of various executable
		    scripts. It will reconstruct command specified in shebang line and attempt
		    to run it within nix environment, given nix or nix-portable are available.
		    If nix or nix-portable are not available, it will attempt to run the
		    reconstructed command directly, assuming all required tools are already
		    present on the machine. However, this script can also be used to quickly
		    jump into nix environment.

		    It has a special behavior when \`-- "\$0"\` is being used in the shebang
		    command and "\$0" parameter evaluates to a relative path starting with
		    \`./\` or \`../\`. In this case, the \`env.sh\` will capture the current
		    working directory, change the current working directory to the folder
		    containing the script specified by the relative path and it will pass
		    captured path as a first parameter to the \`"\$0\"\` script. This is useful
		    if the program specified before \`--\` resolves something based on the
		    folder where the \`"\$0"\` script is located. For example, \`pnpm exec\` can
		    resolve cli tools in \`node_modules/\` dir using node.js resolution
		    algorithm when some ancestor folder contains \`node_modules/\` dir.

		    Note, when used in shebang line, one of the two configurable parts of the
		    line is represented by \`<command>...\` parameter in the usage above and the
		    other is the relative location from the current script to \`env.sh\` file.
		    That is all other parts are needed for correct execution of \`env.sh\`
		    script itself. In that example, \`\$(dirname \$(readlink -f "\$0"))\` part
		    is needed to resolve relative location of \`env.sh\` script, and
		    \`"\$0" "\$@"\` part is necessary to pass the script path and the parameter
		    to \`env.sh\`.

		Examples:
		    When running as executable on terminal:
		        ./env.sh -- bash --norc --noprofile
		        ./env.sh python ./some/script.py
		        ./env.sh pnpm exec tsx -- ./some/script.mts

		    When using in shebang line of an executable script:
		        #!/usr/bin/env -S bash -c '"\$(dirname \$(readlink -f "\$0"))/env.sh" bash "\$0" "\$@"'
		        #!/usr/bin/env -S bash -c '"\$(dirname \$(readlink -f "\$0"))/evaluation/env.sh" python "\$0" "\$@"'
		        #!/usr/bin/env -S bash -c '"\$(dirname \$(readlink -f "\$0"))/../env.sh" pnpm exec tsx -- "\$0" "\$@"'
	EOF
}

# based on https://stackoverflow.com/a/39376824
if ! OPTS=$(getopt -o "-h" --long "help" -n "env.sh" -- "$@"); then
  echo "Error in command line arguments." >&2
  exit 1
fi

declare -a COMMAND

eval set -- "$OPTS"
while true; do
  case "$1" in
  -h | --help)
    usage
    exit
    ;;
  --)
    shift
    break
    ;;
  *)
    while [[ $1 != "--" ]]; do
      COMMAND+=("$1")
      shift
    done
    ;;
  esac
done

ORIGINAL_PWD="$(pwd)"
ENV_DIR="$(dirname "$0")"

if [[ $# -gt 0 && ($1 == "./"* || $1 == "../"*) ]]; then
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

if [[ ${#COMMAND[@]} -eq 0 ]]; then
  usage
  exit
fi

if [[ -n ${IN_NIX_SHELL+x} ]]; then
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
