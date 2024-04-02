#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" pnpm exec tsx -- "$0" "$@"'

import { $ } from "zx";

await $`ls -alh`;
