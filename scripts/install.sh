#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/config.sh
source "${SCRIPT_DIR}/lib/config.sh"
# shellcheck source=lib/services.sh
source "${SCRIPT_DIR}/lib/services.sh"
# shellcheck source=lib/traffic.sh
source "${SCRIPT_DIR}/lib/traffic.sh"

case "${1:-help}" in
help|--help|-h)
  cat <<'EOF'
sb2sub 管理脚本

可用命令:
  install              安装、生成本地运行文件并写入服务
  validate             检查本地环境
  reinstall-singbox    重装 sing-box 标记
  update-singbox       更新 sing-box 标记
  uninstall-singbox    卸载 sing-box 标记
  generate-config      生成配置文件
  refresh-traffic      刷新流量显示
EOF
  ;;
install)
  sb2sub_install_singbox
  sb2sub_generate_config
  sb2sub_install_command_link
  sb2sub_install_daemon_service
  sb2sub_reload_service
  sb2sub_enable_service
  sb2sub_restart_service
  ;;
validate)
  sb2sub_validate_environment
  sb2sub_note "环境检查通过"
  ;;
reinstall-singbox)
  sb2sub_reinstall_singbox
  ;;
update-singbox)
  sb2sub_update_singbox
  ;;
uninstall-singbox)
  sb2sub_uninstall_singbox
  ;;
generate-config)
  sb2sub_generate_config
  ;;
refresh-traffic)
  sb2sub_refresh_traffic
  ;;
*)
  sb2sub_error "未知命令: ${1}"
  exit 1
  ;;
esac
