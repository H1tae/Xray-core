#!/usr/bin/env bash

set -euo pipefail

xray_deploy_repo_dir() {
  local profile_dir="$1"
  (cd "${profile_dir}/.." && pwd)
}

xray_deploy_load_env() {
  local env_file="$1"
  set -a
  # shellcheck disable=SC1090
  source "${env_file}"
  set +a
}

xray_deploy_resolve_path() {
  local profile_dir="$1"
  local path="$2"

  if [[ "${path}" = /* ]]; then
    printf '%s\n' "${path}"
  else
    printf '%s\n' "${profile_dir}/${path#./}"
  fi
}

xray_deploy_wait_for_docker() {
  local docker_wait_timeout="$1"
  local deadline docker_output
  deadline=$((SECONDS + docker_wait_timeout))

  while true; do
    if docker info >/dev/null 2>&1; then
      return 0
    fi

    docker_output="$(docker info 2>&1 || true)"
    if [[ "${docker_output}" == *"permission denied"* ]]; then
      echo "docker daemon is running, but access to /var/run/docker.sock is denied" >&2
      echo "add your user to the docker group and start a new shell session" >&2
      echo "example: sudo usermod -aG docker \$USER && newgrp docker" >&2
      return 1
    fi

    if (( SECONDS >= deadline )); then
      echo "docker daemon is unavailable after ${docker_wait_timeout}s" >&2
      echo "start Docker Desktop / docker.service first" >&2
      return 1
    fi

    sleep 2
  done
}

xray_deploy_build_image() {
  local profile_dir="$1"
  local env_file="${profile_dir}/.env"
  local env_example_file="${profile_dir}/.env.example"
  local repo_dir default_image_name default_image_tar

  repo_dir="$(xray_deploy_repo_dir "${profile_dir}")"
  default_image_name="eiravpn-xray:latest"
  default_image_tar="${profile_dir}/eiravpn-xray-image.tar.gz"

  if [[ ! -f "${repo_dir}/Dockerfile" ]]; then
    echo "missing ${repo_dir}/Dockerfile" >&2
    exit 1
  fi

  if [[ ! -x "${repo_dir}/xray" ]]; then
    echo "missing executable ${repo_dir}/xray" >&2
    echo "build or copy the xray binary before running build_image.sh" >&2
    exit 1
  fi

  if [[ ! -f "${env_file}" ]]; then
    cp "${env_example_file}" "${env_file}"
    echo "created ${env_file} from ${env_example_file}"
  fi

  xray_deploy_load_env "${env_file}"

  local image_name="${XRAY_DOCKER_IMAGE:-${default_image_name}}"
  local image_tar="${XRAY_DOCKER_IMAGE_TAR:-${default_image_tar}}"
  local docker_wait_timeout="${XRAY_DOCKER_WAIT_TIMEOUT_SECONDS:-60}"

  image_tar="$(xray_deploy_resolve_path "${profile_dir}" "${image_tar}")"

  xray_deploy_wait_for_docker "${docker_wait_timeout}" || exit 1

  if docker buildx version >/dev/null 2>&1; then
    docker buildx build --load \
      -t "${image_name}" \
      -f "${repo_dir}/Dockerfile" \
      "${repo_dir}"
  else
    docker build \
      -t "${image_name}" \
      -f "${repo_dir}/Dockerfile" \
      "${repo_dir}"
  fi

  mkdir -p "$(dirname "${image_tar}")"
  case "${image_tar}" in
    *.tar.gz|*.tgz)
      docker save "${image_name}" | gzip -c > "${image_tar}"
      ;;
    *)
      docker save -o "${image_tar}" "${image_name}"
      ;;
  esac

  echo "saved ${image_name} to ${image_tar}"
}

xray_deploy_start() {
  local profile_dir="$1"
  local env_file="${2:-${profile_dir}/.env}"
  local default_image_name="eiravpn-xray:latest"
  local default_image_tar="${profile_dir}/eiravpn-xray-image.tar.gz"
  local compose_file="${profile_dir}/docker-compose.yml"

  if [[ ! -f "${env_file}" ]]; then
    echo "missing ${env_file}" >&2
    echo "copy ${profile_dir}/.env.example to ${env_file} first" >&2
    exit 1
  fi

  xray_deploy_load_env "${env_file}"

  local image_name="${XRAY_DOCKER_IMAGE:-${default_image_name}}"
  local image_tar="${XRAY_DOCKER_IMAGE_TAR:-${default_image_tar}}"
  local config_dir="${XRAY_CONFIG_DIR:-.}"
  local config_file
  local docker_wait_timeout="${XRAY_DOCKER_WAIT_TIMEOUT_SECONDS:-60}"
  local runtime_config_dir="/app/configs"

  image_tar="$(xray_deploy_resolve_path "${profile_dir}" "${image_tar}")"
  config_dir="$(xray_deploy_resolve_path "${profile_dir}" "${config_dir}")"
  config_file="${config_dir}/server.json"

  if [[ ! -d "${config_dir}" ]]; then
    echo "missing XRAY_CONFIG_DIR=${config_dir}" >&2
    exit 1
  fi

  if [[ ! -f "${config_file}" ]]; then
    echo "missing ${config_file}" >&2
    exit 1
  fi

  xray_deploy_wait_for_docker "${docker_wait_timeout}" || exit 1

  if [[ -f "${image_tar}" ]]; then
    case "${image_tar}" in
      *.tar.gz|*.tgz)
        gzip -dc "${image_tar}" | docker load
        ;;
      *)
        docker load -i "${image_tar}"
        ;;
    esac
  elif ! docker image inspect "${image_name}" >/dev/null 2>&1; then
    echo "missing Docker image ${image_name}" >&2
    echo "expected either a loaded image or ${image_tar}" >&2
    exit 1
  fi

  if ! docker run --rm \
      --network host \
      -v "${config_dir}:${runtime_config_dir}:ro" \
      "${image_name}" run -test -config "${runtime_config_dir}/server.json"; then
    echo "xray config test failed for ${config_file}" >&2
    exit 1
  fi

  export XRAY_CONFIG_DIR="${config_dir}"

  docker compose \
    --env-file "${env_file}" \
    -f "${compose_file}" \
    up -d
}

xray_deploy_stop() {
  local profile_dir="$1"
  local env_file="${2:-${profile_dir}/.env}"

  docker compose \
    --env-file "${env_file}" \
    -f "${profile_dir}/docker-compose.yml" \
    down
}

xray_deploy_status() {
  local profile_dir="$1"
  local env_file="${2:-${profile_dir}/.env}"

  docker compose \
    --env-file "${env_file}" \
    -f "${profile_dir}/docker-compose.yml" \
    ps
}

xray_deploy_logs() {
  local profile_dir="$1"
  local env_file="${2:-${profile_dir}/.env}"
  local service="${3:-xray}"

  docker compose \
    --env-file "${env_file}" \
    -f "${profile_dir}/docker-compose.yml" \
    logs -f "${service}"
}
