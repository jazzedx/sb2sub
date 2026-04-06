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
TARGETS=${TARGETS:-"linux/amd64"}
COMMIT=${COMMIT:-$(git -C "${REPO_DIR}" rev-parse --short HEAD 2>/dev/null || printf 'none')}
BUILD_TIME=${BUILD_TIME:-$(date -u '+%Y-%m-%dT%H:%M:%SZ')}
STAGE_DIR="${OUTPUT_DIR}/stage"

mkdir -p "${OUTPUT_DIR}"
rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}"

sb2sub_copy_release_files() {
  local stage_root=$1

  mkdir -p "${stage_root}/scripts" "${stage_root}/bin"
  cp "${REPO_DIR}/README.md" "${stage_root}/README.md"
  cp "${REPO_DIR}/scripts/get-latest.sh" "${stage_root}/scripts/get-latest.sh"
  cp "${REPO_DIR}/scripts/install.sh" "${stage_root}/scripts/install.sh"
  cp "${REPO_DIR}/scripts/menu.sh" "${stage_root}/scripts/menu.sh"
  cp "${REPO_DIR}/scripts/sample-config.yaml" "${stage_root}/scripts/sample-config.yaml"
  cp "${REPO_DIR}/scripts/sb2sub" "${stage_root}/scripts/sb2sub"
  cp -R "${REPO_DIR}/scripts/lib" "${stage_root}/scripts/lib"
  cp -R "${REPO_DIR}/packaging" "${stage_root}/packaging"
  cp -R "${REPO_DIR}/templates" "${stage_root}/templates"
  chmod 755 "${stage_root}/scripts/get-latest.sh" "${stage_root}/scripts/install.sh" "${stage_root}/scripts/menu.sh" "${stage_root}/scripts/sb2sub"
  printf '%s\n' "${VERSION}" >"${stage_root}/VERSION"
}

sb2sub_build_binary() {
  local goos=$1
  local goarch=$2
  local output=$3
  local cc_value=
  local ldflags=

  case "${goarch}" in
  amd64)
    cc_value=${CC_AMD64:-${CC:-}}
    ;;
  arm64)
    cc_value=${CC_ARM64:-${CC:-}}
    ;;
  *)
    cc_value=${CC:-}
    ;;
  esac

  ldflags="-s -w -X sb2sub/internal/buildinfo.version=${VERSION} -X sb2sub/internal/buildinfo.commit=${COMMIT} -X sb2sub/internal/buildinfo.builtAt=${BUILD_TIME}"

  if [[ -n "${cc_value}" ]]; then
    env CGO_ENABLED=1 GOOS="${goos}" GOARCH="${goarch}" CC="${cc_value}" \
      go build -trimpath -ldflags "${ldflags}" -o "${output}" ./cmd/sb2subd
    return 0
  fi

  env CGO_ENABLED=1 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -ldflags "${ldflags}" -o "${output}" ./cmd/sb2subd
}

for target in ${TARGETS}; do
  IFS=/ read -r goos goarch <<EOF
${target}
EOF

  archive_base="sb2sub_${VERSION}_${goos}_${goarch}"
  stage_parent="${STAGE_DIR}/${goos}_${goarch}"
  stage_root="${stage_parent}/sb2sub"
  archive_file="${OUTPUT_DIR}/${archive_base}.tar.gz"

  rm -rf "${stage_parent}" "${archive_file}"
  mkdir -p "${stage_root}"

  sb2sub_copy_release_files "${stage_root}"
  (
    cd "${REPO_DIR}"
    sb2sub_build_binary "${goos}" "${goarch}" "${stage_root}/bin/sb2subd"
  )
  chmod 755 "${stage_root}/bin/sb2subd"

  tar -C "${stage_parent}" -czf "${archive_file}" sb2sub
  printf '已生成 %s\n' "${archive_file}"
done

(
  cd "${OUTPUT_DIR}"
  sha256sum ./*.tar.gz >checksums.txt
)

rm -rf "${STAGE_DIR}"
printf '已生成 %s/checksums.txt\n' "${OUTPUT_DIR}"
