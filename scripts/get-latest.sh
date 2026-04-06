#!/usr/bin/env bash
set -euo pipefail

SB2SUB_RELEASE_REPOSITORY=${SB2SUB_RELEASE_REPOSITORY:-jazzedx/sb2sub}
SB2SUB_RELEASE_API=${SB2SUB_RELEASE_API:-"https://api.github.com/repos/${SB2SUB_RELEASE_REPOSITORY}/releases/latest"}
SB2SUB_RELEASE_DOWNLOAD_BASE=${SB2SUB_RELEASE_DOWNLOAD_BASE:-"https://github.com/${SB2SUB_RELEASE_REPOSITORY}/releases/download"}
SB2SUB_INSTALL_DIR=${SB2SUB_INSTALL_DIR:-/opt/sb2sub}
SB2SUB_BIN_LINK_DIR=${SB2SUB_BIN_LINK_DIR:-/usr/local/bin}
SB2SUB_SYSTEMD_DIR=${SB2SUB_SYSTEMD_DIR:-/etc/systemd/system}
SB2SUB_BASE_DIR=${SB2SUB_BASE_DIR:-"${SB2SUB_INSTALL_DIR}/runtime"}

sb2sub_arch() {
  case "$(uname -m)" in
  x86_64 | amd64)
    printf 'amd64\n'
    ;;
  aarch64 | arm64)
    printf 'arm64\n'
    ;;
  *)
    printf '不支持的机器类型: %s\n' "$(uname -m)" >&2
    return 1
    ;;
  esac
}

sb2sub_latest_version() {
  curl -fsSL "${SB2SUB_RELEASE_API}" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

ARCH=$(sb2sub_arch)
VERSION=$(sb2sub_latest_version)
if [[ -z "${VERSION}" ]]; then
  printf '无法获取最新版本\n' >&2
  exit 1
fi

ARCHIVE_NAME="sb2sub_${VERSION}_linux_${ARCH}.tar.gz"
ARCHIVE_FILE="${TMP_DIR}/${ARCHIVE_NAME}"
DOWNLOAD_URL="${SB2SUB_RELEASE_DOWNLOAD_BASE}/${VERSION}/${ARCHIVE_NAME}"

curl -fsSL -o "${ARCHIVE_FILE}" "${DOWNLOAD_URL}"
tar -xzf "${ARCHIVE_FILE}" -C "${TMP_DIR}"

mkdir -p "$(dirname -- "${SB2SUB_INSTALL_DIR}")"
rm -rf "${SB2SUB_INSTALL_DIR}"
mv "${TMP_DIR}/sb2sub" "${SB2SUB_INSTALL_DIR}"

SB2SUB_BASE_DIR="${SB2SUB_BASE_DIR}" \
SB2SUB_BIN_LINK_DIR="${SB2SUB_BIN_LINK_DIR}" \
SB2SUB_SYSTEMD_DIR="${SB2SUB_SYSTEMD_DIR}" \
  bash "${SB2SUB_INSTALL_DIR}/scripts/install.sh" install

printf '已安装最新版本 %s\n' "${VERSION}"
printf '可直接使用: sb2sub status\n'
