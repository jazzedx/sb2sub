#!/usr/bin/env bash

sb2sub_traffic_help() {
  cat <<'EOF'
流量管理

可用命令:
  sb2sub traffic show [--user 用户名]
  sb2sub traffic reset [--user 用户名]
EOF
}

sb2sub_traffic_show() {
  local user_ref=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --user|--name)
      user_ref=${2:-}
      shift 2
      ;;
    -h|--help)
      sb2sub_traffic_help
      return 0
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done

  sb2sub_refresh_traffic >/dev/null
  if [[ -n "${user_ref}" ]]; then
    sb2sub_show_traffic_table "username = $(sb2sub_sql_quote "${user_ref}")"
  else
    sb2sub_show_traffic_table
  fi
}

sb2sub_traffic_reset() {
  local user_ref=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --user|--name)
      user_ref=${2:-}
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    -h|--help)
      sb2sub_traffic_help
      return 0
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done

  if [[ -n "${user_ref}" ]]; then
    sb2sub_confirm_or_exit "即将清零用户 ${user_ref} 的流量，是否继续" "${auto_confirm}"
    sb2sub_reset_traffic_usage "username = $(sb2sub_sql_quote "${user_ref}")"
    printf '已清零用户流量: %s\n' "${user_ref}"
  else
    sb2sub_confirm_or_exit "即将清零全部用户流量，是否继续" "${auto_confirm}"
    sb2sub_reset_traffic_usage
    printf '已清零全部用户流量\n'
  fi
}

sb2sub_handle_traffic_command() {
  local action=${1:-show}
  shift || true

  case "${action}" in
  help|--help|-h)
    sb2sub_traffic_help
    ;;
  show)
    sb2sub_traffic_show "$@"
    ;;
  reset)
    sb2sub_traffic_reset "$@"
    ;;
  *)
    sb2sub_error "未知的流量命令: ${action}"
    return 1
    ;;
  esac
}
