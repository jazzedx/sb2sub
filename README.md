# sb2sub

`sb2sub` 是一个围绕 `sing-box` 构建的轻量单机订阅项目，目标是把安装、服务管理、用户管理、订阅管理和常见维护动作都收拢到一个命令里。

## 当前已完成的基础内容

- 内置 Go 后台，可提供本地管理接口和公开订阅输出
- 中文管理脚本和基础菜单入口
- `sb2sub` 统一命令入口和中文交互菜单
- `sing-box` 运行配置生成
- 基于 SQLite 的用户、订阅和流量计数存储
- 面向 `Clash Verge Rev` 和 `Shadowrocket` 的订阅输出
- 自定义订阅路径、域名、端口和协议开关管理

## 本地开发

```bash
make test
bash scripts/dev-smoke.sh
```

## 发布与部署

打包发布：

```bash
make release VERSION=v0.1.0
make verify-release VERSION=v0.1.0
```

发布包会输出到 `dist/`，里面包含可直接部署的目录结构和内置后台程序，不再要求目标机器先装 Go。

远程机器一条命令安装最新版：

```bash
curl -fsSL https://raw.githubusercontent.com/jazzedx/sb2sub/main/scripts/get-latest.sh | sudo bash
```

安装完成后可以直接使用：

```bash
sb2sub service status
sb2sub menu
```

解压发布包后可直接执行：

```bash
bash scripts/install.sh install
sb2sub menu
```

推送 `v*` 标签后，GitHub Actions 会自动跑测试、生成 `linux/amd64` 和 `linux/arm64` 发布包，并发布到 Release 页面。

## 目录说明

- `cmd/sb2subd`：后台程序入口
- `internal/`：项目内部代码
- `scripts/`：安装、菜单和辅助脚本
- `packaging/systemd/`：系统服务模板
- `templates/`：客户端与运行配置模板说明

## 常用命令

```bash
sb2sub --help
sb2sub menu
sb2sub service status
sb2sub user add --name alice --quota 10G --days 30
sb2sub sub add --user alice --type clash --name alice-clash
sb2sub traffic show
sb2sub server domain --value example.com
sb2sub server cert status
```

## 统一管理入口

```bash
sb2sub service status|start|stop|restart|enable|disable|logs|update
sb2sub user add|list|show|enable|disable|delete|reset|quota|expire
sb2sub sub add|list|show|enable|disable|delete|reset
sb2sub traffic show|reset
sb2sub server show|domain|protocol|port|cert|reload
sb2sub menu
```

## 菜单说明

`sb2sub menu` 会打开中文交互菜单，分成 6 组：

- 快捷安装 / 修复
- 核心管理
- 域名与证书
- 协议与端口设置
- 用户与订阅管理
- 流量与维护

菜单本身只是快捷入口，最终都会调用同一套 `sb2sub` 命令。
