#!/usr/bin/env sh

set -eu

mode="${1:-run}"
template_file="${XRAY_CONFIG_TEMPLATE_FILE:-/app/configs/server.json}"
runtime_file="${XRAY_RUNTIME_CONFIG_FILE:-}"
redirect_address="${XRAY_REDIRECT_ADDRESS:-}"
redirect_token="${XRAY_REDIRECT_ADDRESS_TOKEN:-}"
config_file="${template_file}"
child_pid=""

if [ ! -f "${template_file}" ]; then
  echo "missing config template ${template_file}" >&2
  exit 1
fi

cleanup() {
  if [ "${config_file}" = "${runtime_file}" ] && [ -n "${runtime_file}" ]; then
    rm -f "${runtime_file}"
  fi
}

forward_term() {
  if [ -n "${child_pid}" ]; then
    kill -TERM "${child_pid}" 2>/dev/null || true
  fi
}

forward_int() {
  if [ -n "${child_pid}" ]; then
    kill -INT "${child_pid}" 2>/dev/null || true
  fi
}

trap cleanup EXIT
trap 'forward_term; exit 143' TERM HUP QUIT
trap 'forward_int; exit 130' INT

if [ -n "${redirect_token}" ] && grep -Fq -- "${redirect_token}" "${template_file}"; then
  if [ -z "${redirect_address}" ]; then
    echo "XRAY_REDIRECT_ADDRESS must be set when ${redirect_token} is used in ${template_file}" >&2
    exit 1
  fi

  if [ -z "${runtime_file}" ]; then
    runtime_file="$(mktemp /tmp/server.runtime.XXXXXX)"
  fi

  rendered="$(cat "${template_file}")"
  escaped_token="$(printf '%s' "${redirect_token}" | sed 's/[][\/.^$*|]/\\&/g')"
  escaped_address="$(printf '%s' "${redirect_address}" | sed 's/[&|\\]/\\&/g')"
  rendered="$(printf '%s' "${rendered}" | sed "s|${escaped_token}|${escaped_address}|g")"
  printf '%s' "${rendered}" > "${runtime_file}"
  config_file="${runtime_file}"
fi

case "${mode}" in
  run)
    /usr/local/bin/xray run -config "${config_file}" &
    child_pid="$!"
    wait "${child_pid}"
    ;;
  test)
    /usr/local/bin/xray run -test -config "${config_file}"
    ;;
  *)
    echo "unsupported xray entrypoint mode: ${mode}" >&2
    exit 1
    ;;
esac
