#!/usr/bin/env bash

sb2sub_list_users() {
  if [[ -f "${SB2SUB_DB_FILE}" ]]; then
    sqlite3 "${SB2SUB_DB_FILE}" 'select id, username, enabled, quota_bytes, used_upload_bytes + used_download_bytes from users order by id;'
  else
    sb2sub_note "数据库尚未初始化"
  fi
}
