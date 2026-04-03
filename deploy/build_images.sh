#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for profile in deploy_allocated deploy_shared; do
  echo "==> building ${profile}"
  bash "${script_dir}/${profile}/build_image.sh"
done
