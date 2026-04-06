#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

sb2sub_exec() {
  bash "${SCRIPT_DIR}/sb2sub" "$@"
}

show_help() {
  cat <<'EOF'
sb2sub 管理菜单

顶层菜单:
  1. 快捷安装 / 修复
  2. 核心管理
  3. 域名与证书
  4. 协议与端口设置
  5. 用户与订阅管理
  6. 流量与维护

快捷命令:
  生成 sing-box 配置
  生成默认配置
  刷新流量显示
EOF
}

menu_quick_install() {
  while true; do
    cat <<'EOF'

[快捷安装 / 修复]
1. 执行安装 / 修复
2. 检查环境
3. 生成 sing-box 配置
0. 返回上一级
EOF
    case "$(sb2sub_prompt '请选择' '0')" in
    1) sb2sub_exec install ;;
    2) sb2sub_exec validate ;;
    3) sb2sub_exec generate-config ;;
    0) return 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

menu_service() {
  while true; do
    cat <<'EOF'

[核心管理]
1. 查看状态
2. 启动服务
3. 停止服务
4. 重启服务
5. 设置开机自启
6. 取消开机自启
7. 查看日志
0. 返回上一级
EOF
    case "$(sb2sub_prompt '请选择' '0')" in
    1) sb2sub_exec service status ;;
    2) sb2sub_exec service start ;;
    3) sb2sub_exec service stop ;;
    4) sb2sub_exec service restart ;;
    5) sb2sub_exec service enable ;;
    6) sb2sub_exec service disable ;;
    7) sb2sub_exec service logs ;;
    0) return 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

menu_domain_cert() {
  while true; do
    cat <<'EOF'

[域名与证书]
1. 查看当前配置
2. 设置域名
3. 保存 Cloudflare Token
4. 申请证书
5. 续期证书
6. 查看证书状态
0. 返回上一级
EOF
    case "$(sb2sub_prompt '请选择' '0')" in
    1) sb2sub_exec server show ;;
    2) sb2sub_exec server domain ;;
    3) sb2sub_exec server cert set-cloudflare ;;
    4) sb2sub_exec server cert issue ;;
    5) sb2sub_exec server cert renew ;;
    6) sb2sub_exec server cert status ;;
    0) return 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

menu_protocol_port() {
  while true; do
    cat <<'EOF'

[协议与端口设置]
1. 查看当前配置
2. 开关 VLESS
3. 开关 Hysteria2
4. 修改 VLESS 端口
5. 修改 Hysteria2 端口
0. 返回上一级
EOF
    case "$(sb2sub_prompt '请选择' '0')" in
    1) sb2sub_exec server show ;;
    2) sb2sub_exec server protocol --name vless ;;
    3) sb2sub_exec server protocol --name hysteria2 ;;
    4) sb2sub_exec server port --name vless ;;
    5) sb2sub_exec server port --name hysteria2 ;;
    0) return 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

menu_user_sub() {
  while true; do
    cat <<'EOF'

[用户与订阅管理]
1. 查看用户列表
2. 新增用户
3. 查看用户详情
4. 启用用户
5. 禁用用户
6. 重置用户凭据
7. 删除用户
8. 查看订阅列表
9. 新增订阅
10. 查看订阅详情
11. 启用订阅
12. 禁用订阅
13. 重置订阅链接
14. 删除订阅
0. 返回上一级
EOF
    case "$(sb2sub_prompt '请选择' '0')" in
    1) sb2sub_exec user list ;;
    2) sb2sub_exec user add ;;
    3) sb2sub_exec user show ;;
    4) sb2sub_exec user enable ;;
    5) sb2sub_exec user disable ;;
    6) sb2sub_exec user reset ;;
    7) sb2sub_exec user delete ;;
    8) sb2sub_exec sub list ;;
    9) sb2sub_exec sub add ;;
    10) sb2sub_exec sub show ;;
    11) sb2sub_exec sub enable ;;
    12) sb2sub_exec sub disable ;;
    13) sb2sub_exec sub reset ;;
    14) sb2sub_exec sub delete ;;
    0) return 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

menu_traffic_maintenance() {
  while true; do
    cat <<'EOF'

[流量与维护]
1. 查看流量总览
2. 清零全部流量
3. 检查环境
4. 重新生成配置
5. 拉取最新版
0. 返回上一级
EOF
    case "$(sb2sub_prompt '请选择' '0')" in
    1) sb2sub_exec traffic show ;;
    2) sb2sub_exec traffic reset ;;
    3) sb2sub_exec validate ;;
    4) sb2sub_exec generate-config ;;
    5) sb2sub_exec update ;;
    0) return 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

show_main_menu() {
  cat <<'EOF'
sb2sub 管理菜单
1. 快捷安装 / 修复
2. 核心管理
3. 域名与证书
4. 协议与端口设置
5. 用户与订阅管理
6. 流量与维护
0. 退出
EOF
}

run_menu() {
  while true; do
    show_main_menu
    case "$(sb2sub_prompt '请选择' '0')" in
    1) menu_quick_install ;;
    2) menu_service ;;
    3) menu_domain_cert ;;
    4) menu_protocol_port ;;
    5) menu_user_sub ;;
    6) menu_traffic_maintenance ;;
    0) exit 0 ;;
    *) sb2sub_error "无效选择" ;;
    esac
  done
}

case "${1:-menu}" in
--help|-h|help)
  show_help
  ;;
render-singbox|generate-config)
  exec bash "${SCRIPT_DIR}/install.sh" generate-config
  ;;
refresh-traffic)
  exec bash "${SCRIPT_DIR}/install.sh" refresh-traffic
  ;;
menu)
  run_menu
  ;;
*)
  sb2sub_error "未知命令: ${1}"
  exit 1
  ;;
esac
