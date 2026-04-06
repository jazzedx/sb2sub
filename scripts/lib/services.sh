#!/usr/bin/env bash

sb2sub_install_command_link() {
  mkdir -p "${SB2SUB_BIN_LINK_DIR}"
  cat >"${SB2SUB_BIN_LINK_DIR}/sb2sub" <<EOF
#!/usr/bin/env bash
exec bash "${SB2SUB_REPO_DIR}/scripts/sb2sub" "\$@"
EOF
  chmod 755 "${SB2SUB_BIN_LINK_DIR}/sb2sub"
  sb2sub_note "已写入全局命令 ${SB2SUB_BIN_LINK_DIR}/sb2sub"
}

sb2sub_install_daemon_service() {
  local template_file="${SB2SUB_REPO_DIR}/packaging/systemd/sb2subd.service"

  mkdir -p "${SB2SUB_SYSTEMD_DIR}"
  sed \
    -e "s|__SB2SUB_HOME__|${SB2SUB_REPO_DIR}|g" \
    -e "s|__SB2SUB_BASE_DIR__|${SB2SUB_BASE_DIR}|g" \
    "${template_file}" >"${SB2SUB_DAEMON_SERVICE_FILE}"
  sb2sub_note "已写入服务文件 ${SB2SUB_DAEMON_SERVICE_FILE}"
}

sb2sub_require_systemctl() {
  if ! command -v systemctl >/dev/null 2>&1; then
    sb2sub_error "当前系统无法使用 systemctl"
    return 1
  fi
}

sb2sub_systemctl() {
  sb2sub_require_systemctl
  systemctl "$@"
}

sb2sub_reload_service() {
  sb2sub_systemctl daemon-reload
}

sb2sub_enable_service() {
  sb2sub_systemctl enable "${SB2SUB_DAEMON_SERVICE_NAME}"
}

sb2sub_disable_service() {
  sb2sub_systemctl disable "${SB2SUB_DAEMON_SERVICE_NAME}"
}

sb2sub_start_service() {
  sb2sub_systemctl start "${SB2SUB_DAEMON_SERVICE_NAME}"
}

sb2sub_stop_service() {
  sb2sub_systemctl stop "${SB2SUB_DAEMON_SERVICE_NAME}"
}

sb2sub_restart_service() {
  sb2sub_systemctl restart "${SB2SUB_DAEMON_SERVICE_NAME}"
}

sb2sub_status_service() {
  sb2sub_systemctl status --no-pager "${SB2SUB_DAEMON_SERVICE_NAME}"
}

sb2sub_logs_service() {
  if ! command -v journalctl >/dev/null 2>&1; then
    sb2sub_error "当前系统无法使用 journalctl"
    return 1
  fi
  journalctl -u "${SB2SUB_DAEMON_SERVICE_NAME}" -n 100 --no-pager
}

sb2sub_install_singbox() {
  sb2sub_ensure_runtime_dirs
  printf '%s\n' "installed" >"${SB2SUB_STATE_DIR}/singbox.status"
  sb2sub_note "已标记 sing-box 为已安装"
}

sb2sub_reinstall_singbox() {
  sb2sub_install_singbox
  sb2sub_note "已标记 sing-box 为已重装"
}

sb2sub_update_singbox() {
  sb2sub_ensure_runtime_dirs
  printf '%s\n' "updated" >"${SB2SUB_STATE_DIR}/singbox.status"
  sb2sub_note "已标记 sing-box 为已更新"
}

sb2sub_uninstall_singbox() {
  rm -f "${SB2SUB_STATE_DIR}/singbox.status"
  sb2sub_note "已移除 sing-box 标记"
}
