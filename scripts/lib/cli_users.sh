#!/usr/bin/env bash

sb2sub_user_help() {
  cat <<'EOF'
用户管理

可用命令:
  sb2sub user add [--name 用户名] [--note 备注] [--quota 10G] [--days 30]
  sb2sub user list
  sb2sub user show --name 用户名
  sb2sub user enable --name 用户名
  sb2sub user disable --name 用户名
  sb2sub user delete --name 用户名
  sb2sub user reset --name 用户名
  sb2sub user quota --name 用户名 --value 20G
  sb2sub user expire --name 用户名 --days 60
EOF
}

sb2sub_user_where_clause() {
  local ref=$1
  if [[ "${ref}" =~ ^[0-9]+$ ]]; then
    printf 'id = %s\n' "${ref}"
  else
    printf 'username = %s\n' "$(sb2sub_sql_quote "${ref}")"
  fi
}

sb2sub_user_get_row() {
  local ref=$1
  sb2sub_ensure_database
  sb2sub_db_query "
    SELECT id, username, note, enabled, quota_bytes, used_upload_bytes, used_download_bytes,
           expires_at, vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
    FROM users
    WHERE $(sb2sub_user_where_clause "${ref}")
    LIMIT 1;
  "
}

sb2sub_user_require_name() {
  local provided=${1:-}
  if [[ -n "${provided}" ]]; then
    printf '%s\n' "${provided}"
    return 0
  fi

  provided=$(sb2sub_prompt "用户名")
  if [[ -z "${provided}" ]]; then
    sb2sub_error "用户名不能为空"
    return 1
  fi
  printf '%s\n' "${provided}"
}

sb2sub_user_assert_exists() {
  local ref=$1
  local row
  row=$(sb2sub_user_get_row "${ref}")
  if [[ -z "${row}" ]]; then
    sb2sub_error "未找到用户: ${ref}"
    return 1
  fi
}

sb2sub_user_add() {
  local username=""
  local note=""
  local quota_raw=""
  local days=""
  local enabled=1
  local vless_enabled=""
  local hysteria2_enabled=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
      username=${2:-}
      shift 2
      ;;
    --note)
      note=${2:-}
      shift 2
      ;;
    --quota)
      quota_raw=${2:-}
      shift 2
      ;;
    --days)
      days=${2:-}
      shift 2
      ;;
    --enabled)
      enabled=$(sb2sub_bool_from_flag "${2:-}")
      shift 2
      ;;
    --vless)
      vless_enabled=$(sb2sub_bool_from_flag "${2:-}")
      shift 2
      ;;
    --hysteria2)
      hysteria2_enabled=$(sb2sub_bool_from_flag "${2:-}")
      shift 2
      ;;
    --yes|-y)
      auto_confirm=1
      shift
      ;;
    -h|--help)
      sb2sub_user_help
      return 0
      ;;
    *)
      sb2sub_error "未知参数: $1"
      return 1
      ;;
    esac
  done

  username=$(sb2sub_user_require_name "${username}")
  if [[ -z "${note}" ]]; then
    note=$(sb2sub_prompt "备注" "")
  fi
  if [[ -z "${quota_raw}" ]]; then
    quota_raw=$(sb2sub_prompt "总流量上限" "0")
  fi
  if [[ -z "${days}" ]]; then
    days=$(sb2sub_prompt "有效天数" "30")
  fi
  if [[ -z "${vless_enabled}" ]]; then
    vless_enabled=$(sb2sub_prompt_yesno "开启 VLESS-Reality" "y")
  fi
  if [[ -z "${hysteria2_enabled}" ]]; then
    hysteria2_enabled=$(sb2sub_prompt_yesno "开启 Hysteria2" "y")
  fi

  local quota_bytes
  local expires_at
  local created_at
  local vless_uuid
  local hysteria2_password

  quota_bytes=$(sb2sub_parse_size "${quota_raw:-0}")
  expires_at=$(sb2sub_iso_after_days "${days:-30}")
  created_at=$(sb2sub_now_iso)
  vless_uuid=$(sb2sub_generate_uuid)
  hysteria2_password=$(sb2sub_generate_token 24)

  sb2sub_ensure_database
  sb2sub_db_exec "
    INSERT INTO users (
      username, note, enabled, created_at, updated_at, expires_at,
      quota_bytes, used_upload_bytes, used_download_bytes,
      vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
    ) VALUES (
      $(sb2sub_sql_quote "${username}"),
      $(sb2sub_sql_quote "${note}"),
      ${enabled},
      $(sb2sub_sql_quote "${created_at}"),
      $(sb2sub_sql_quote "${created_at}"),
      $(sb2sub_sql_quote "${expires_at}"),
      ${quota_bytes},
      0,
      0,
      $(sb2sub_sql_quote "${vless_uuid}"),
      $(sb2sub_sql_quote "${hysteria2_password}"),
      ${vless_enabled},
      ${hysteria2_enabled}
    );
  "

  printf '已创建用户: %s\n' "${username}"
  printf 'VLESS UUID: %s\n' "${vless_uuid}"
  printf 'Hysteria2 密码: %s\n' "${hysteria2_password}"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_user_list() {
  sb2sub_ensure_database
  printf '用户名\t状态\t总量\t剩余\t到期时间\t备注\n'
  while IFS=$'\t' read -r id username note enabled quota used_up used_down expires_at vless_uuid hysteria2_password vless_enabled hysteria2_enabled; do
    [[ -n "${id}" ]] || continue
    local total=$((used_up + used_down))
    local remain=$((quota - total))
    if (( remain < 0 )); then
      remain=0
    fi
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
      "${username}" \
      "$(sb2sub_bool_to_word "${enabled}")" \
      "$(sb2sub_human_bytes "${total}")" \
      "$(sb2sub_human_bytes "${remain}")" \
      "${expires_at}" \
      "${note}"
  done < <(sb2sub_db_query "
    SELECT id, username, note, enabled, quota_bytes, used_upload_bytes, used_download_bytes,
           expires_at, vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
    FROM users
    ORDER BY id ASC;
  ")
}

sb2sub_user_show() {
  local ref=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
      ref=${2:-}
      shift 2
      ;;
    *)
      ref=$1
      shift
      ;;
    esac
  done
  ref=$(sb2sub_user_require_name "${ref}")

  local row
  row=$(sb2sub_user_get_row "${ref}")
  if [[ -z "${row}" ]]; then
    sb2sub_error "未找到用户: ${ref}"
    return 1
  fi

  IFS=$'\t' read -r id username note enabled quota used_up used_down expires_at vless_uuid hysteria2_password vless_enabled hysteria2_enabled <<<"${row}"
  printf '用户名: %s\n' "${username}"
  printf '备注: %s\n' "${note}"
  printf '状态: %s\n' "$(sb2sub_bool_to_word "${enabled}")"
  printf '总流量上限: %s\n' "$(sb2sub_human_bytes "${quota}")"
  printf '已用上传: %s\n' "$(sb2sub_human_bytes "${used_up}")"
  printf '已用下载: %s\n' "$(sb2sub_human_bytes "${used_down}")"
  printf '已用总量: %s\n' "$(sb2sub_human_bytes "$((used_up + used_down))")"
  printf '到期时间: %s\n' "${expires_at}"
  printf 'VLESS-Reality: %s\n' "$(sb2sub_bool_to_word "${vless_enabled}")"
  printf 'VLESS UUID: %s\n' "$(sb2sub_mask_middle "${vless_uuid}")"
  printf 'Hysteria2: %s\n' "$(sb2sub_bool_to_word "${hysteria2_enabled}")"
  printf 'Hysteria2 密码: %s\n' "$(sb2sub_mask_middle "${hysteria2_password}")"
}

sb2sub_user_update_enabled() {
  local desired=$1
  shift
  local ref=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
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
  ref=$(sb2sub_user_require_name "${ref}")
  sb2sub_user_assert_exists "${ref}"

  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE users
    SET enabled = ${desired}, updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE $(sb2sub_user_where_clause "${ref}");
  "
  printf '已更新用户状态: %s -> %s\n' "${ref}" "$(sb2sub_bool_to_word "${desired}")"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_user_delete() {
  local ref=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
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
  ref=$(sb2sub_user_require_name "${ref}")
  sb2sub_user_assert_exists "${ref}"
  sb2sub_confirm_or_exit "即将删除用户 ${ref}，是否继续" "${auto_confirm}"
  sb2sub_ensure_database
  sb2sub_db_exec "DELETE FROM users WHERE $(sb2sub_user_where_clause "${ref}");"
  printf '已删除用户: %s\n' "${ref}"
  sb2sub_apply_runtime_changes 1
}

sb2sub_user_reset() {
  local ref=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
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
  ref=$(sb2sub_user_require_name "${ref}")
  sb2sub_user_assert_exists "${ref}"

  local vless_uuid
  local hysteria2_password
  vless_uuid=$(sb2sub_generate_uuid)
  hysteria2_password=$(sb2sub_generate_token 24)

  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE users
    SET vless_uuid = $(sb2sub_sql_quote "${vless_uuid}"),
        hysteria2_password = $(sb2sub_sql_quote "${hysteria2_password}"),
        updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE $(sb2sub_user_where_clause "${ref}");
  "
  printf '已重置用户凭据: %s\n' "${ref}"
  printf 'VLESS UUID: %s\n' "${vless_uuid}"
  printf 'Hysteria2 密码: %s\n' "${hysteria2_password}"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_user_set_quota() {
  local ref=""
  local quota_raw=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
      ref=${2:-}
      shift 2
      ;;
    --value|--quota)
      quota_raw=${2:-}
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
  ref=$(sb2sub_user_require_name "${ref}")
  sb2sub_user_assert_exists "${ref}"
  if [[ -z "${quota_raw}" ]]; then
    quota_raw=$(sb2sub_prompt "新的流量上限" "0")
  fi
  local quota_bytes
  quota_bytes=$(sb2sub_parse_size "${quota_raw}")
  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE users
    SET quota_bytes = ${quota_bytes},
        updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE $(sb2sub_user_where_clause "${ref}");
  "
  printf '已更新流量上限: %s -> %s\n' "${ref}" "$(sb2sub_human_bytes "${quota_bytes}")"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_user_set_expire() {
  local ref=""
  local days=""
  local auto_confirm=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --name|--user)
      ref=${2:-}
      shift 2
      ;;
    --days)
      days=${2:-}
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
  ref=$(sb2sub_user_require_name "${ref}")
  sb2sub_user_assert_exists "${ref}"
  if [[ -z "${days}" ]]; then
    days=$(sb2sub_prompt "延长多少天" "30")
  fi
  local expires_at
  expires_at=$(sb2sub_iso_after_days "${days}")
  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE users
    SET expires_at = $(sb2sub_sql_quote "${expires_at}"),
        updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE $(sb2sub_user_where_clause "${ref}");
  "
  printf '已更新到期时间: %s -> %s\n' "${ref}" "${expires_at}"
  sb2sub_apply_runtime_changes "${auto_confirm}"
}

sb2sub_handle_user_command() {
  local action=${1:-help}
  shift || true

  case "${action}" in
  help|--help|-h)
    sb2sub_user_help
    ;;
  add)
    sb2sub_user_add "$@"
    ;;
  list)
    sb2sub_user_list
    ;;
  show)
    sb2sub_user_show "$@"
    ;;
  enable)
    sb2sub_user_update_enabled 1 "$@"
    ;;
  disable)
    sb2sub_user_update_enabled 0 "$@"
    ;;
  delete|remove|rm)
    sb2sub_user_delete "$@"
    ;;
  reset)
    sb2sub_user_reset "$@"
    ;;
  quota)
    sb2sub_user_set_quota "$@"
    ;;
  expire)
    sb2sub_user_set_expire "$@"
    ;;
  *)
    sb2sub_error "未知的用户命令: ${action}"
    return 1
    ;;
  esac
}
