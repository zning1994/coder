#!/usr/bin/env bash

# Allow toggling verbose output
[[ -n ${VERBOSE:-""} ]] && set -x
set -euo pipefail

export SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")
. "${SCRIPT_DIR}/scripts/lib/develop-setup.sh"

# Compile the CLI binary once just so we don't waste time compiling things multiple times
go build -o "${CODER_DEV_BIN}" "${PROJECT_ROOT}/cmd/coder"

# Run yarn install, to make sure node_modules are ready to go
"$PROJECT_ROOT/scripts/yarn_install.sh"

# This is a way to run multiple processes in parallel, and have Ctrl-C work correctly
# to kill both at the same time. For more details, see:
# https://stackoverflow.com/questions/3004811/how-do-you-run-multiple-programs-in-parallel-from-a-bash-script
(
	# If something goes wrong, just bail and tear everything down
	# rather than leaving things in an inconsistent state.
	trap 'kill -TERM -$$' ERR
	cdroot
	CODER_HOST=http://127.0.0.1:3000 INSPECT_XSTATE=true yarn --cwd=./site dev || kill -INT -$$ &
	"${CODER_DEV_SHIM}" server --address 127.0.0.1:3000 --in-memory --tunnel || kill -INT -$$ &

	"${SCRIPT_DIR}/scripts/lib/develop-postflight.sh"

	# Wait for both frontend and backend to exit.
	wait
)
