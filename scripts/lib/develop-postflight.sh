#!/usr/bin/env bash

set +u
if [ -z "${CODER_DEV_SHIM}" ]; then
    echo '== ERROR: must provide CODER_DEV_SHIM'
    exit 1
fi
# BEGIN POSTFLIGHT
echo '== Waiting for Coder to become ready'
timeout 60s bash -c 'until curl -s --fail http://localhost:3000 > /dev/null 2>&1; do sleep 0.5; done'

#  create the first user, the admin
"${CODER_DEV_SHIM}" login http://127.0.0.1:3000 --username=admin --email=admin@coder.com --password="${CODER_DEV_ADMIN_PASSWORD}" ||
    echo 'Failed to create admin user. To troubleshoot, try running this command manually.'

# || true to always exit code 0. If this fails, whelp.
"${CODER_DEV_SHIM}" users create --email=member@coder.com --username=member --password="${CODER_DEV_ADMIN_PASSWORD}" ||
    echo 'Failed to create regular user. To troubleshoot, try running this command manually.'

# If we have docker available, then let's try to create a template!
template_name=""
if docker info >/dev/null 2>&1; then
    temp_template_dir=$(mktemp -d)
    echo code-server | "${CODER_DEV_SHIM}" templates init "${temp_template_dir}"
    # shellcheck disable=SC1090
    source <(go env | grep GOARCH)
    DOCKER_HOST=$(docker context inspect --format '{{.Endpoints.docker.Host}}')
    printf 'docker_arch: "%s"\ndocker_host: "%s"\n' "${GOARCH}" "${DOCKER_HOST}" | tee "${temp_template_dir}/params.yaml"
    template_name="docker-${GOARCH}"
    "${CODER_DEV_SHIM}" templates create "${template_name}" --directory "${temp_template_dir}" --parameter-file "${temp_template_dir}/params.yaml" --yes
    rm -rfv "${temp_template_dir}"
fi

# BEGIN BANNER
log
log "======================================================================="
log "==                                                                   =="
log "==               Coder is now running in development mode.           =="
log "==                    API   : http://localhost:3000                  =="
log "==                    Web UI: http://localhost:8080                  =="
if [[ -n "${template_name}" ]]; then
    log "==                                                                   =="
    log "==            Docker template ${template_name} is ready to use!          =="
    log "==            Use ./scripts/coder-dev.sh to talk to this instance!   =="
    log "==                                                                   =="
fi
log "======================================================================="
log
# END BANNER

# END POSTFLIGHT
