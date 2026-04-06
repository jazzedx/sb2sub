#!/usr/bin/env bash

sb2sub_list_subscriptions() {
  if [[ -f "${SB2SUB_DB_FILE}" ]]; then
    sqlite3 "${SB2SUB_DB_FILE}" 'select id, user_id, name, type, enabled, token from subscriptions order by id;'
  else
    sb2sub_note "数据库尚未初始化"
  fi
}
