#!/usr/bin/env bash

set -euo pipefail

xray_deploy_install_docker_ubuntu() {
  if [[ ${EUID} -ne 0 ]]; then
    exec sudo -E bash "$0" "$@"
  fi

  . /etc/os-release

  case "${ID}" in
    ubuntu|debian)
      ;;
    *)
      echo "unsupported distro: ${ID}" >&2
      exit 1
      ;;
  esac

  local repo_os="${ID}"

  apt-get update
  apt-get install -y ca-certificates curl gnupg

  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://download.docker.com/linux/${repo_os}/gpg" \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg

  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${repo_os} ${VERSION_CODENAME} stable" \
    > /etc/apt/sources.list.d/docker.list

  apt-get update
  apt-get install -y \
    containerd.io \
    docker-buildx-plugin \
    docker-ce \
    docker-ce-cli \
    docker-compose-plugin

  systemctl enable --now docker

  if [[ -n "${SUDO_USER:-}" ]]; then
    usermod -aG docker "${SUDO_USER}"
    echo "docker installed; log out and back in once so ${SUDO_USER} can run docker without sudo"
  else
    echo "docker installed"
  fi
}

xray_deploy_repo_dir() {
  local profile_dir="$1"
  (cd "${profile_dir}/../.." && pwd)
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

xray_deploy_profile_source_config() {
  local repo_dir="$1"
  local profile_name="$2"

  case "${profile_name}" in
    deploy_allocated)
      printf '%s\n' "${repo_dir}/configs/server_allocated.json"
      ;;
    deploy_shared)
      printf '%s\n' "${repo_dir}/configs/server_shared.json"
      ;;
    *)
      echo "unsupported deploy profile ${profile_name}" >&2
      return 1
      ;;
  esac
}

xray_deploy_build_docker_image() {
  local image_name="$1"
  local dockerfile="$2"
  local build_context="$3"

  if docker buildx version >/dev/null 2>&1; then
    docker buildx build --load \
      -t "${image_name}" \
      -f "${dockerfile}" \
      "${build_context}"
  else
    docker build \
      -t "${image_name}" \
      -f "${dockerfile}" \
      "${build_context}"
  fi
}

xray_deploy_save_docker_image() {
  local image_name="$1"
  local image_tar="$2"

  mkdir -p "$(dirname "${image_tar}")"
  case "${image_tar}" in
    *.tar.gz|*.tgz)
      docker save "${image_name}" | gzip -c > "${image_tar}"
      ;;
    *)
      docker save -o "${image_tar}" "${image_name}"
      ;;
  esac
}

xray_deploy_load_docker_image() {
  local image_name="$1"
  local image_tar="$2"

  if [[ -f "${image_tar}" ]]; then
    case "${image_tar}" in
      *.tar.gz|*.tgz)
        gzip -dc "${image_tar}" | docker load
        ;;
      *)
        docker load -i "${image_tar}"
        ;;
    esac
    return 0
  fi

  docker image inspect "${image_name}" >/dev/null 2>&1
}

xray_deploy_validate_runtime_config() {
  local image_name="$1"
  local env_file="$2"
  local config_dir="$3"
  local script_file="$4"

  docker run --rm \
    --entrypoint /bin/sh \
    --network host \
    --env-file "${env_file}" \
    -v "${config_dir}:/app/configs:ro" \
    -v "${script_file}:/app/deploy/xray_entrypoint.sh:ro" \
    "${image_name}" \
    /app/deploy/xray_entrypoint.sh test
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
    echo "build_image.sh expects a full repository checkout" >&2
    echo "on a server with only $(basename "${profile_dir}"), use the prebuilt image tar and start.sh" >&2
    exit 1
  fi

  local profile_name source_config target_config
  profile_name="$(basename "${profile_dir}")"
  source_config="$(xray_deploy_profile_source_config "${repo_dir}" "${profile_name}")"
  target_config="${profile_dir}/configs/server.json"

  if [[ ! -f "${source_config}" ]]; then
    echo "missing source config ${source_config}" >&2
    exit 1
  fi

  if [[ ! -f "${env_file}" ]]; then
    cp "${env_example_file}" "${env_file}"
    echo "created ${env_file} from ${env_example_file}"
  fi

  xray_deploy_load_env "${env_file}"

  mkdir -p "$(dirname "${target_config}")"
  cp "${source_config}" "${target_config}"
  echo "synced ${target_config} from ${source_config}"

  local image_name="${XRAY_DOCKER_IMAGE:-${default_image_name}}"
  local image_tar="${default_image_tar}"
  local docker_wait_timeout="${XRAY_DOCKER_WAIT_TIMEOUT_SECONDS:-60}"

  xray_deploy_wait_for_docker "${docker_wait_timeout}" || exit 1

  xray_deploy_build_docker_image \
    "${image_name}" \
    "${repo_dir}/Dockerfile" \
    "${repo_dir}"
  xray_deploy_save_docker_image "${image_name}" "${image_tar}"
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
  local image_tar="${default_image_tar}"
  local config_dir="${XRAY_CONFIG_DIR:-./configs}"
  local config_file
  local docker_wait_timeout="${XRAY_DOCKER_WAIT_TIMEOUT_SECONDS:-60}"
  local runtime_config_dir="/app/configs"
  local entrypoint_script="${profile_dir}/xray_entrypoint.sh"

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

  if ! xray_deploy_load_docker_image "${image_name}" "${image_tar}"; then
    echo "missing Docker image ${image_name}" >&2
    echo "expected either a loaded image or ${image_tar}" >&2
    exit 1
  fi

  if [[ ! -f "${entrypoint_script}" ]]; then
    echo "missing ${entrypoint_script}" >&2
    exit 1
  fi

  if ! xray_deploy_validate_runtime_config "${image_name}" "${env_file}" \
      "${config_dir}" "${entrypoint_script}"; then
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
