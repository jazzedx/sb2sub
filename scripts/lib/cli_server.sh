#!/usr/bin/env bash

sb2sub_server_help() {
  cat <<'EOF'
服务器配置

可用命令:
  sb2sub server show
  sb2sub server domain --value example.com
  sb2sub server protocol --name vless|hysteria2 --enabled true|false
  sb2sub server port --name vless|hysteria2 --value 443
  sb2sub server cert set-cloudflare --token <token>
  sb2sub server cert issue
  sb2sub server cert renew
  sb2sub server cert status
  sb2sub server reload
EOF
}

sb2sub_server_require_protocol() {
  case "${1:-}" in
  vless|hysteria2)
    printf '%s\n' "$1"
    ;;
  *)
    sb2sub_error "协议名称只能是 vless 或 hysteria2"
    return 1
    ;;
  esac
}

sb2sub_server_show() {
  sb2sub_ensure_config_file
  local vless_enabled
  local hysteria2_enabled
  vless_enabled=$(sb2sub_config_get_protocol_enabled vless)
  hysteria2_enabled=$(sb2sub_config_get_protocol_enabled hysteria2)
  printf '域名: %s\n' "$(sb2sub_config_get_domain)"
  printf '证书文件: %s\n' "$(sb2sub_config_get_certificate_file)"
  printf '证书私钥: %s\n' "$(sb2sub_config_get_certificate_key_file)"
  printf 'VLESS: %s (端口 %s)\n' \
    "$(sb2sub_bool_to_word "${vless_enabled}")" \
    "$(sb2sub_config_get_protocol_port vless)"
  printf 'Hysteria2: %s (端口 %s)\n' \
    "$(sb2sub_bool_to_word "${hysteria2_enabled}")" \
    "$(sb2sub_config_get_protocol_port hysteria2)"
  printf '\n证书状态:\n'
  sb2sub_show_cert_status
}

sb2sub_server_domain() {
  local value=""
  local auto_confirm=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --value)
      value=${2:-}
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done
  if [[ -z "${value}" ]]; then
    value=$(sb2sub_prompt "新的域名" "$(sb2sub_config_get_domain)")
  fi
  sb2sub_config_set_domain "${value}"
  printf '已更新域名: %s\n' "${value}"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_server_protocol() {
  local protocol=""
  local enabled=""
  local auto_confirm=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name)
      protocol=${2:-}
      shift 2
      ;;
    --enabled)
      enabled=$(sb2sub_bool_from_flag "${2:-}")
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done
  protocol=$(sb2sub_server_require_protocol "${protocol}")
  if [[ -z "${enabled}" ]]; then
    enabled=$(sb2sub_prompt_yesno "是否开启 ${protocol}" "y")
  fi
  if [[ "${enabled}" == "1" ]]; then
    sb2sub_config_set_protocol_enabled "${protocol}" "true"
  else
    sb2sub_config_set_protocol_enabled "${protocol}" "false"
  fi
  printf '已更新协议开关: %s -> %s\n' "${protocol}" "$(sb2sub_bool_to_word "${enabled}")"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_server_port() {
  local protocol=""
  local value=""
  local auto_confirm=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name)
      protocol=${2:-}
      shift 2
      ;;
    --value)
      value=${2:-}
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done
  protocol=$(sb2sub_server_require_protocol "${protocol}")
  if [[ -z "${value}" ]]; then
    value=$(sb2sub_prompt "新的端口" "$(sb2sub_config_get_protocol_port "${protocol}")")
  fi
  if [[ ! "${value}" =~ ^[0-9]+$ ]]; then
    sb2sub_error "端口必须是数字"
    return 1
  fi
  sb2sub_config_set_protocol_port "${protocol}" "${value}"
  printf '已更新端口: %s -> %s\n' "${protocol}" "${value}"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_server_cert() {
  local action=${1:-status}
  shift || true
  local auto_confirm=0

  case "${action}" in
  set-cloudflare)
    local token=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
      --token)
        token=${2:-}
        shift 2
        ;;
      *)
        sb2sub_error "未知参数: $1"
        return 1
        ;;
      esac
    done
    if [[ -z "${token}" ]]; then
      token=$(sb2sub_prompt "Cloudflare API Token")
    fi
    sb2sub_cert_write_cloudflare_env "${token}"
    printf '已保存 Cloudflare Token\n'
    ;;
  issue)
    while [[ $# -gt 0 ]]; do
      case "$1" in
      --yes|-y)
        auto_confirm=1
        shift
        ;;
      *)
        sb2sub_error "未知参数: $1"
        return 1
        ;;
      esac
    done
    sb2sub_cert_issue
    printf '已申请证书\n'
    sb2sub_apply_runtime_changes "${auto_confirm}"
    ;;
  renew)
    while [[ $# -gt 0 ]]; do
      case "$1" in
      --yes|-y)
        auto_confirm=1
        shift
        ;;
      *)
        sb2sub_error "未知参数: $1"
        return 1
        ;;
      esac
    done
    sb2sub_cert_renew
    printf '已续期证书\n'
    sb2sub_apply_runtime_changes "${auto_confirm}"
    ;;
  status)
    sb2sub_show_cert_status
    ;;
  *)
    sb2sub_error "未知的证书命令: ${action}"
    return 1
    ;;
  esac
}

sb2sub_handle_server_command() {
  local action=${1:-show}
  shift || true

  case "${action}" in
  help|--help|-h)
    sb2sub_server_help
    ;;
  show)
    sb2sub_server_show
    ;;
  domain)
    sb2sub_server_domain "$@"
    ;;
  protocol)
    sb2sub_server_protocol "$@"
    ;;
  port)
    sb2sub_server_port "$@"
    ;;
  cert)
    sb2sub_server_cert "$@"
    ;;
  reload)
    sb2sub_apply_runtime_changes "${SB2SUB_AUTO_CONFIRM:-0}"
    ;;
  *)
    sb2sub_error "未知的服务器命令: ${action}"
    return 1
    ;;
  esac
}
