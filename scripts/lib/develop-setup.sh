#!/usr/bin/env bash

(return 0 2>/dev/null) && SOURCED=1 || SOURCED=0
if [ "$SOURCED" -ne 1 ]; then
    echo '== ERROR: this script is only useful if sourced, not executed'
    exit 1
fi

set +u
if [ -z "${SCRIPT_DIR}" ]; then
    echo '== ERROR: must provide SCRIPT_DIR'
    return 1
fi
set -u

# BEGIN preflight setup
source "${SCRIPT_DIR}/lib.sh"
export PROJECT_ROOT=$(cd "$SCRIPT_DIR" && git rev-parse --show-toplevel)
export CODER_DEV_BIN="${PROJECT_ROOT}/.coderv2/coder"
set +u
export CODER_DEV_ADMIN_PASSWORD="${CODER_DEV_ADMIN_PASSWORD:-password}"
set -u

# Preflight checks: ensure we have our required dependencies, and make sure nothing is listening on port 3000 or 8080
dependencies curl git go make yarn
curl --fail http://127.0.0.1:3000 >/dev/null 2>&1 && echo '== ERROR: something is listening on port 3000. Kill it and re-run this script.' && exit 1
curl --fail http://127.0.0.1:8080 >/dev/null 2>&1 && echo '== ERROR: something is listening on port 8080. Kill it and re-run this script.' && exit 1

if [[ ! -e "${PROJECT_ROOT}/site/out/bin/coder.sha1" && ! -e "${PROJECT_ROOT}/site/out/bin/coder.tar.zst" ]]; then
	log
	log "======================================================================="
	log "==   Run 'make bin' before running this command to build binaries.   =="
	log "==       Without these binaries, workspaces will fail to start!      =="
	log "======================================================================="
	log
#	exit 1
fi

# Use the coder dev shim so we don't overwrite the user's existing Coder config.
export CODER_DEV_SHIM="${PROJECT_ROOT}/scripts/coder-dev.sh"

# END preflight setup
set +u
