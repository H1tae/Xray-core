#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
env_file="${1:-${script_dir}/.env}"

bash "${script_dir}/bootstrap_server.sh" "${env_file}"
