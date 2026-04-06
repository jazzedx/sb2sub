#!/usr/bin/env bash

sb2sub_refresh_traffic() {
  sb2sub_ensure_database
  sb2sub_note "已刷新当前流量统计"
}

sb2sub_show_traffic_table() {
  local filter_clause=${1:-1=1}

  sb2sub_ensure_database
  printf '用户名\t状态\t上传\t下载\t总量\t剩余\t到期时间\n'
  while IFS=$'\t' read -r username enabled quota used_up used_down expires_at; do
    [[ -n "${username}" ]] || continue
    local total=$((used_up + used_down))
    local remain=$((quota - total))
    if (( remain < 0 )); then
      remain=0
    fi
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
      "${username}" \
      "$(sb2sub_bool_to_word "${enabled}")" \
      "$(sb2sub_human_bytes "${used_up}")" \
      "$(sb2sub_human_bytes "${used_down}")" \
      "$(sb2sub_human_bytes "${total}")" \
      "$(sb2sub_human_bytes "${remain}")" \
      "${expires_at}"
  done < <(sb2sub_db_query "
    SELECT username, enabled, quota_bytes, used_upload_bytes, used_download_bytes, expires_at
    FROM users
    WHERE ${filter_clause}
    ORDER BY id ASC;
  ")
}

sb2sub_reset_traffic_usage() {
  local filter_clause=${1:-1=1}
  sb2sub_ensure_database
  sb2sub_db_exec "
    UPDATE users
    SET used_upload_bytes = 0,
        used_download_bytes = 0,
        updated_at = $(sb2sub_sql_quote "$(sb2sub_now_iso)")
    WHERE ${filter_clause};
  "
}
