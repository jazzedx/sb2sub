#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_DIR=$(cd -- "${SCRIPT_DIR}/.." && pwd)

VERSION=${1:-}
if [[ -z "${VERSION}" ]]; then
  printf '用法: %s <version>\n' "${0##*/}" >&2
  exit 1
fi

OUTPUT_DIR=${OUTPUT_DIR:-"${REPO_DIR}/dist"}
TARGET_ARCH=${TARGET_ARCH:-$(go env GOARCH)}
ARCHIVE_FILE="${OUTPUT_DIR}/sb2sub_${VERSION}_linux_${TARGET_ARCH}.tar.gz"

if [[ ! -f "${ARCHIVE_FILE}" ]]; then
  printf '未找到发布包: %s\n' "${ARCHIVE_FILE}" >&2
  exit 1
fi

TMP_DIR=$(mktemp -d)
TOOLS_DIR="${TMP_DIR}/tools"
BASE_DIR="${TMP_DIR}/runtime"
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${TOOLS_DIR}"
for tool in cat dirname mkdir pwd; do
  ln -s "$(command -v "${tool}")" "${TOOLS_DIR}/${tool}"
done

tar -xzf "${ARCHIVE_FILE}" -C "${TMP_DIR}"

PATH="${TOOLS_DIR}" SB2SUB_BASE_DIR="${BASE_DIR}" /bin/bash "${TMP_DIR}/sb2sub/scripts/install.sh" validate >/dev/null
PATH="${TOOLS_DIR}" SB2SUB_BASE_DIR="${BASE_DIR}" /bin/bash "${TMP_DIR}/sb2sub/scripts/install.sh" generate-config >/dev/null

if [[ ! -s "${BASE_DIR}/etc/sing-box.json" ]]; then
  printf '生成配置失败，缺少 %s\n' "${BASE_DIR}/etc/sing-box.json" >&2
  exit 1
fi

version_output=$("${TMP_DIR}/sb2sub/bin/sb2subd" --version)
case "${version_output}" in
*"${VERSION}"*)
  ;;
*)
  printf '版本输出不包含 %s: %s\n' "${VERSION}" "${version_output}" >&2
  exit 1
  ;;
esac

printf '发布包校验通过: %s\n' "${ARCHIVE_FILE}"
