#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
env_file="${1:-${script_dir}/.env}"
# shellcheck disable=SC1091
source "${script_dir}/common.sh"

xray_deploy_install_docker_ubuntu "$@"

if [[ ${EUID} -eq 0 ]]; then
  bash "${script_dir}/start.sh" "${env_file}"
  exit 0
fi

if command -v sg >/dev/null 2>&1; then
  sg docker "bash '${script_dir}/start.sh' '${env_file}'"
  exit 0
fi

echo "docker installed, but 'sg' is unavailable" >&2
echo "run these commands in a new shell session:" >&2
echo "  newgrp docker" >&2
echo "  bash ${script_dir}/start.sh ${env_file}" >&2
exit 1
