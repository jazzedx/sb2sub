#!/usr/bin/env bash

sb2sub_sub_help() {
  cat <<'EOF'
订阅管理

可用命令:
  sb2sub sub add [--user 用户名] [--type clash|shadowrocket|both] [--name 名称]
  sb2sub sub list
  sb2sub sub show --name 名称
  sb2sub sub enable --name 名称
  sb2sub sub disable --name 名称
  sb2sub sub delete --name 名称
  sb2sub sub reset --name 名称
EOF
}

sb2sub_sub_where_clause() {
  local ref=$1
  if [[ "${ref}" =~ ^[0-9]+$ ]]; then
    printf 's.id = %s\n' "${ref}"
  else
    printf 's.name = %s\n' "$(sb2sub_sql_quote "${ref}")"
  fi
}

sb2sub_sub_get_row() {
  local ref=$1
  sb2sub_ensure_database
  sb2sub_db_query "
    SELECT s.id, s.name, u.username, s.type, s.token, s.custom_path, s.enabled, s.created_at, s.updated_at
    FROM subscriptions s
    JOIN users u ON u.id = s.user_id
    WHERE $(sb2sub_sub_where_clause "${ref}")
    LIMIT 1;
  "
}

sb2sub_sub_require_name() {
  local provided=${1:-}
  if [[ -n "${provided}" ]]; then
    printf '%s\n' "${provided}"
    return 0
  fi
  provided=$(sb2sub_prompt "订阅名称")
  if [[ -z "${provided}" ]]; then
    sb2sub_error "订阅名称不能为空"
    return 1
  fi
  printf '%s\n' "${provided}"
}

sb2sub_sub_require_user() {
  local provided=${1:-}
  if [[ -n "${provided}" ]]; then
    printf '%s\n' "${provided}"
    return 0
  fi
  provided=$(sb2sub_prompt "所属用户名")
  if [[ -z "${provided}" ]]; then
    sb2sub_error "用户名不能为空"
    return 1
  fi
  printf '%s\n' "${provided}"
}

sb2sub_sub_assert_exists() {
  local ref=$1
  local row
  row=$(sb2sub_sub_get_row "${ref}")
  if [[ -z "${row}" ]]; then
    sb2sub_error "未找到订阅: ${ref}"
    return 1
  fi
}

sb2sub_sub_validate_type() {
  case "${1:-}" in
  clash|shadowrocket|both)
    printf '%s\n' "$1"
    ;;
  *)
    sb2sub_error "不支持的订阅类型: ${1:-}"
    return 1
    ;;
  esac
}

sb2sub_sub_add() {
  local user_ref=""
  local sub_type=""
  local name=""
  local token=""
  local custom_path=""
  local enabled=1
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --user)
      user_ref=${2:-}
      shift 2
      ;;
    --type)
      sub_type=${2:-}
      shift 2
      ;;
    --name)
      name=${2:-}
      shift 2
      ;;
    --token)
      token=${2:-}
      shift 2
      ;;
    --path|--custom-path)
      custom_path=${2:-}
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
    -h|--help)
      sb2sub_sub_help
      return 0
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done

  user_ref=$(sb2sub_sub_require_user "${user_ref}")
  if [[ -z "${sub_type}" ]]; then
    sub_type=$(sb2sub_prompt "客户端类型(clash/shadowrocket/both)" "clash")
  fi
  sub_type=$(sb2sub_sub_validate_type "${sub_type}")
  name=$(sb2sub_sub_require_name "${name}")
  if [[ -z "${custom_path}" ]]; then
    custom_path=$(sb2sub_prompt "自定义路径(留空则自动生成)" "")
  fi
  custom_path=$(sb2sub_normalize_path "${custom_path}")
  if [[ -z "${token}" ]]; then
    token=$(sb2sub_generate_token 32)
  fi

  local user_row
  local user_id
  user_row=$(sb2sub_user_get_row "${user_ref}")
  if [[ -z "${user_row}" ]]; then
    sb2sub_error "未找到用户: ${user_ref}"
    return 1
  fi
  IFS=$'\t' read -r user_id username note enabled_user quota used_up used_down expires_at vless_uuid hysteria2_password vless_enabled hysteria2_enabled <<<"${user_row}"

  local now
  now=$(sb2sub_now_iso)
  sb2sub_ensure_database
  sb2sub_db_exec "
    INSERT INTO subscriptions (
      user_id, name, type, token, custom_path, enabled, created_at, updated_at, last_accessed_at
    ) VALUES (
      ${user_id},
      $(sb2sub_sql_quote "${name}"),
      $(sb2sub_sql_quote "${sub_type}"),
      $(sb2sub_sql_quote "${token}"),
      $(sb2sub_sql_quote "${custom_path}"),
      ${enabled},
      $(sb2sub_sql_quote "${now}"),
      $(sb2sub_sql_quote "${now}"),
      NULL
    );
  "

  printf '已创建订阅: %s\n' "${name}"
  printf '订阅地址: %s\n' "$(sb2sub_subscription_url "${token}" "${custom_path}")"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_sub_list() {
  sb2sub_ensure_database
  printf '订阅名称\t所属用户\t类型\t状态\t订阅地址\n'
  while IFS=$'\t' read -r id name username sub_type token custom_path enabled created_at updated_at; do
    [[ -n "${id}" ]] || continue
    printf '%s\t%s\t%s\t%s\t%s\n' \
      "${name}" \
      "${username}" \
      "${sub_type}" \
      "$(sb2sub_bool_to_word "${enabled}")" \
      "$(sb2sub_mask_middle "$(sb2sub_subscription_url "${token}" "${custom_path}")")"
  done < <(sb2sub_db_query "
    SELECT s.id, s.name, u.username, s.type, s.token, s.custom_path, s.enabled, s.created_at, s.updated_at
    FROM subscriptions s
    JOIN users u ON u.id = s.user_id
    ORDER BY s.id ASC;
  ")
}

sb2sub_sub_show() {
  local ref=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name)
      ref=${2:-}
      shift 2
      ;;
    *)
      ref=$1
      shift
      ;;
    esac
  done
  ref=$(sb2sub_sub_require_name "${ref}")

  local row
  row=$(sb2sub_sub_get_row "${ref}")
  if [[ -z "${row}" ]]; then
    sb2sub_error "未找到订阅: ${ref}"
    return 1
  fi

  IFS=$'\t' read -r id name username sub_type token custom_path enabled created_at updated_at <<<"${row}"
  printf '订阅名称: %s\n' "${name}"
  printf '所属用户: %s\n' "${username}"
  printf '类型: %s\n' "${sub_type}"
  printf '状态: %s\n' "$(sb2sub_bool_to_word "${enabled}")"
  printf '路径: %s\n' "${custom_path:-sub/${token}}"
  printf '订阅地址: %s\n' "$(sb2sub_mask_middle "$(sb2sub_subscription_url "${token}" "${custom_path}")")"
  printf '创建时间: %s\n' "${created_at}"
  printf '更新时间: %s\n' "${updated_at}"
}

sb2sub_sub_update_enabled() {
  local desired=$1
  shift
  local ref=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name)
      ref=${2:-}
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    *)
      ref=$1
      shift
      ;;
    esac
  done
  ref=$(sb2sub_sub_require_name "${ref}")
  sb2sub_sub_assert_exists "${ref}"

  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE subscriptions
    SET enabled = ${desired}, updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE name = $(sb2sub_sql_quote "${ref}");
  "
  printf '已更新订阅状态: %s -> %s\n' "${ref}" "$(sb2sub_bool_to_word "${desired}")"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_sub_delete() {
  local ref=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name)
      ref=${2:-}
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    *)
      ref=$1
      shift
      ;;
    esac
  done
  ref=$(sb2sub_sub_require_name "${ref}")
  sb2sub_sub_assert_exists "${ref}"
  sb2sub_confirm_or_exit "即将删除订阅 ${ref}，是否继续" "${auto_confirm}"
  sb2sub_ensure_database
  sb2sub_db_exec "DELETE FROM subscriptions WHERE name = $(sb2sub_sql_quote "${ref}");"
  printf '已删除订阅: %s\n' "${ref}"
  sb2sub_apply_runtime_changes 1
}

sb2sub_sub_reset() {
  local ref=""
  local auto_confirm=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name)
      ref=${2:-}
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    *)
      ref=$1
      shift
      ;;
    esac
  done
  ref=$(sb2sub_sub_require_name "${ref}")

  local row
  local token
  row=$(sb2sub_sub_get_row "${ref}")
  if [[ -z "${row}" ]]; then
    sb2sub_error "未找到订阅: ${ref}"
    return 1
  fi
  IFS=$'\t' read -r id name username sub_type old_token custom_path enabled created_at updated_at <<<"${row}"
  token=$(sb2sub_generate_token 32)

  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE subscriptions
    SET token = $(sb2sub_sql_quote "${token}"),
        updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE id = ${id};
  "

  printf '已重置订阅链接: %s\n' "${name}"
  printf '订阅地址: %s\n' "$(sb2sub_subscription_url "${token}" "${custom_path}")"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_handle_sub_command() {
  local action=${1:-help}
  shift || true

  case "${action}" in
  help|--help|-h)
    sb2sub_sub_help
    ;;
  add)
    sb2sub_sub_add "$@"
    ;;
  list)
    sb2sub_sub_list
    ;;
  show)
    sb2sub_sub_show "$@"
    ;;
  enable)
    sb2sub_sub_update_enabled 1 "$@"
    ;;
  disable)
    sb2sub_sub_update_enabled 0 "$@"
    ;;
  delete|remove|rm)
    sb2sub_sub_delete "$@"
    ;;
  reset)
    sb2sub_sub_reset "$@"
    ;;
  *)
    sb2sub_error "未知的订阅命令: ${action}"
    return 1
    ;;
  esac
}
