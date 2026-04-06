#!/usr/bin/env bash
set -euo pipefail

SB2SUB_SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
SB2SUB_REPO_DIR=$(cd -- "${SB2SUB_SCRIPT_DIR}/../.." && pwd)
SB2SUB_BASE_DIR=${SB2SUB_BASE_DIR:-"${SB2SUB_REPO_DIR}/runtime"}
SB2SUB_CONFIG_DIR="${SB2SUB_BASE_DIR}/etc"
SB2SUB_DATA_DIR="${SB2SUB_BASE_DIR}/var"
SB2SUB_LOG_DIR="${SB2SUB_BASE_DIR}/log"
SB2SUB_CONFIG_FILE="${SB2SUB_CONFIG_DIR}/config.yaml"
SB2SUB_DB_FILE="${SB2SUB_DATA_DIR}/sb2sub.db"
SB2SUB_SINGBOX_FILE="${SB2SUB_CONFIG_DIR}/sing-box.json"
SB2SUB_STATE_DIR="${SB2SUB_DATA_DIR}/state"
SB2SUB_BIN_DIR="${SB2SUB_REPO_DIR}/bin"
SB2SUB_BIN_LINK_DIR=${SB2SUB_BIN_LINK_DIR:-/usr/local/bin}
SB2SUB_SYSTEMD_DIR=${SB2SUB_SYSTEMD_DIR:-/etc/systemd/system}
SB2SUB_DAEMON_SERVICE_NAME=${SB2SUB_DAEMON_SERVICE_NAME:-sb2subd.service}
SB2SUB_DAEMON_SERVICE_FILE="${SB2SUB_SYSTEMD_DIR}/${SB2SUB_DAEMON_SERVICE_NAME}"
SB2SUB_CERT_ENV_FILE="${SB2SUB_CONFIG_DIR}/cloudflare.env"
SB2SUB_CRON_DIR=${SB2SUB_CRON_DIR:-/etc/cron.d}
SB2SUB_CERT_RENEW_FILE="${SB2SUB_CRON_DIR}/sb2sub-cert-renew"

sb2sub_note() {
  printf '[sb2sub] %s\n' "$*"
}

sb2sub_error() {
  printf '[sb2sub] 错误: %s\n' "$*" >&2
}

sb2sub_ensure_runtime_dirs() {
  mkdir -p "${SB2SUB_CONFIG_DIR}" "${SB2SUB_DATA_DIR}" "${SB2SUB_LOG_DIR}" "${SB2SUB_STATE_DIR}"
}

sb2sub_daemon_path() {
  local daemon_bin="${SB2SUB_BIN_DIR}/sb2subd"
  if [[ -x "${daemon_bin}" ]]; then
    printf '%s\n' "${daemon_bin}"
    return 0
  fi
  return 1
}

sb2sub_run_daemon() {
  local daemon_bin
  if daemon_bin=$(sb2sub_daemon_path); then
    "${daemon_bin}" "$@"
    return 0
  fi

  if command -v go >/dev/null 2>&1; then
    (
      cd "${SB2SUB_REPO_DIR}"
      go run ./cmd/sb2subd "$@"
    )
    return 0
  fi

  sb2sub_error "未找到 sb2subd，可执行文件不存在且本机未安装 Go"
  return 1
}

sb2sub_validate_environment() {
  local daemon_bin
  if daemon_bin=$(sb2sub_daemon_path); then
    sb2sub_note "已检测到内置程序: ${daemon_bin}"
  elif command -v go >/dev/null 2>&1; then
    sb2sub_note "未检测到内置程序，将使用本机 Go 运行"
  else
    sb2sub_error "缺少运行条件：需要内置 sb2subd 或本机 Go"
    return 1
  fi
}

sb2sub_require_sqlite3() {
  if ! command -v sqlite3 >/dev/null 2>&1; then
    sb2sub_error "当前系统缺少 sqlite3"
    return 1
  fi
}

sb2sub_db_exec() {
  sb2sub_require_sqlite3
  sqlite3 "${SB2SUB_DB_FILE}" "PRAGMA foreign_keys = ON; $1"
}

sb2sub_db_query() {
  sb2sub_require_sqlite3
  sqlite3 -noheader -separator $'\t' "${SB2SUB_DB_FILE}" "PRAGMA foreign_keys = ON; $1"
}

sb2sub_ensure_database() {
  sb2sub_ensure_runtime_dirs
  sb2sub_db_exec "
    CREATE TABLE IF NOT EXISTS users (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      username TEXT NOT NULL UNIQUE,
      note TEXT NOT NULL DEFAULT '',
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      expires_at TEXT NOT NULL,
      quota_bytes INTEGER NOT NULL DEFAULT 0,
      used_upload_bytes INTEGER NOT NULL DEFAULT 0,
      used_download_bytes INTEGER NOT NULL DEFAULT 0,
      vless_uuid TEXT NOT NULL,
      hysteria2_password TEXT NOT NULL,
      vless_enabled INTEGER NOT NULL DEFAULT 1,
      hysteria2_enabled INTEGER NOT NULL DEFAULT 1
    );
    CREATE TABLE IF NOT EXISTS subscriptions (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      user_id INTEGER NOT NULL,
      name TEXT NOT NULL,
      type TEXT NOT NULL,
      token TEXT NOT NULL UNIQUE,
      custom_path TEXT NOT NULL DEFAULT '',
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      last_accessed_at TEXT,
      FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    );
  "
}

sb2sub_sql_escape() {
  printf '%s' "$1" | sed "s/'/''/g"
}

sb2sub_sql_quote() {
  printf "'%s'" "$(sb2sub_sql_escape "$1")"
}

sb2sub_prompt() {
  local label=$1
  local default_value=${2:-}
  local answer=

  if [[ -n "${default_value}" ]]; then
    printf '%s [%s]: ' "${label}" "${default_value}" >&2
  else
    printf '%s: ' "${label}" >&2
  fi
  IFS= read -r answer
  if [[ -z "${answer}" ]]; then
    answer=${default_value}
  fi
  printf '%s\n' "${answer}"
}

sb2sub_prompt_yesno() {
  local label=$1
  local default_value=${2:-y}
  local prompt_suffix="Y/n"
  local answer=

  if [[ "${default_value}" != "y" ]]; then
    prompt_suffix="y/N"
  fi
  printf '%s [%s]: ' "${label}" "${prompt_suffix}" >&2
  IFS= read -r answer
  answer=${answer:-${default_value}}
  case "${answer}" in
  y|Y|yes|YES)
    printf '1\n'
    ;;
  *)
    printf '0\n'
    ;;
  esac
}

sb2sub_bool_from_flag() {
  case "${1:-}" in
  1|true|TRUE|True|y|Y|yes|YES|on|ON)
    printf '1\n'
    ;;
  0|false|FALSE|False|n|N|no|NO|off|OFF)
    printf '0\n'
    ;;
  *)
    sb2sub_error "无法识别开关值: ${1:-}"
    return 1
    ;;
  esac
}

sb2sub_bool_to_word() {
  case "${1:-0}" in
  1|true|TRUE|True|y|Y|yes|YES|on|ON)
    printf '开启\n'
    ;;
  *)
    printf '关闭\n'
    ;;
  esac
}

sb2sub_confirm_or_exit() {
  local label=$1
  local auto_confirm=${2:-0}
  if [[ "${auto_confirm}" == "1" || "${SB2SUB_AUTO_CONFIRM:-0}" == "1" ]]; then
    return 0
  fi

  if [[ "$(sb2sub_prompt_yesno "${label}" "y")" != "1" ]]; then
    sb2sub_error "操作已取消"
    return 1
  fi
}

sb2sub_now_iso() {
  date -u '+%Y-%m-%dT%H:%M:%SZ'
}

sb2sub_generate_uuid() {
  if [[ -r /proc/sys/kernel/random/uuid ]]; then
    tr -d '\n' </proc/sys/kernel/random/uuid
    printf '\n'
    return 0
  fi
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]'
    return 0
  fi
  sb2sub_error "当前系统无法生成 UUID"
  return 1
}

sb2sub_generate_token() {
  local length=${1:-24}
  tr -dc 'a-z0-9' </dev/urandom | head -c "${length}"
  printf '\n'
}

sb2sub_normalize_path() {
  local value=${1#/}
  value=${value%/}
  printf '%s\n' "${value}"
}

sb2sub_subscription_url() {
  local token=$1
  local custom_path=${2:-}
  local domain=${3:-}
  local effective_domain=${domain}
  local path

  if [[ -z "${effective_domain}" ]]; then
    effective_domain=$(sb2sub_config_get_domain 2>/dev/null || true)
  fi
  if [[ -z "${effective_domain}" ]]; then
    effective_domain="example.com"
  fi

  if [[ -n "${custom_path}" ]]; then
    path=$(sb2sub_normalize_path "${custom_path}")
  else
    path="sub/${token}"
  fi

  printf 'https://%s/%s\n' "${effective_domain}" "${path}"
}

sb2sub_mask_middle() {
  local value=$1
  local length=${#value}
  if (( length <= 8 )); then
    printf '%s\n' "${value}"
    return 0
  fi
  printf '%s****%s\n' "${value:0:4}" "${value:length-4}"
}

sb2sub_human_bytes() {
  local bytes=${1:-0}
  local units=(B KB MB GB TB PB)
  local value=${bytes}
  local unit_index=0

  while (( value >= 1024 && unit_index < ${#units[@]} - 1 )); do
    value=$((value / 1024))
    unit_index=$((unit_index + 1))
  done

  printf '%s%s\n' "${value}" "${units[unit_index]}"
}

sb2sub_parse_size() {
  local raw=${1^^}
  local number=${raw%[KMGTPB]}
  local suffix=${raw:${#number}}
  if [[ "${raw}" =~ ^[0-9]+$ ]]; then
    printf '%s\n' "${raw}"
    return 0
  fi
  if [[ ! "${number}" =~ ^[0-9]+$ ]]; then
    sb2sub_error "无法识别流量值: ${1}"
    return 1
  fi
  case "${suffix}" in
  K) printf '%s\n' $((number * 1024)) ;;
  M) printf '%s\n' $((number * 1024 * 1024)) ;;
  G) printf '%s\n' $((number * 1024 * 1024 * 1024)) ;;
  T) printf '%s\n' $((number * 1024 * 1024 * 1024 * 1024)) ;;
  P) printf '%s\n' $((number * 1024 * 1024 * 1024 * 1024 * 1024)) ;;
  B|"") printf '%s\n' "${number}" ;;
  *)
    sb2sub_error "无法识别流量单位: ${1}"
    return 1
    ;;
  esac
}

sb2sub_iso_after_days() {
  local days=${1:-30}
  date -u -d "+${days} days" '+%Y-%m-%dT%H:%M:%SZ'
}

sb2sub_apply_runtime_changes() {
  local auto_confirm=${1:-0}
  sb2sub_confirm_or_exit "即将重载配置并重启服务，是否继续" "${auto_confirm}" || return 1
  sb2sub_generate_config
  if [[ -f "${SB2SUB_DAEMON_SERVICE_FILE}" ]]; then
    sb2sub_restart_service
  fi
  sb2sub_note "已生效"
}
