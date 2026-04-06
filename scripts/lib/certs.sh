#!/usr/bin/env bash

sb2sub_cert_acme_path() {
  if command -v acme.sh >/dev/null 2>&1; then
    command -v acme.sh
    return 0
  fi
  if [[ -x "${HOME}/.acme.sh/acme.sh" ]]; then
    printf '%s\n' "${HOME}/.acme.sh/acme.sh"
    return 0
  fi
  return 1
}

sb2sub_cert_write_cloudflare_env() {
  local token=$1
  sb2sub_ensure_runtime_dirs
  cat >"${SB2SUB_CERT_ENV_FILE}" <<EOF
CF_Token=${token}
EOF
  chmod 600 "${SB2SUB_CERT_ENV_FILE}"
}

sb2sub_cert_has_cloudflare_env() {
  [[ -s "${SB2SUB_CERT_ENV_FILE}" ]]
}

sb2sub_cert_issue() {
  local acme_bin
  local domain
  local cert_dir="${SB2SUB_CONFIG_DIR}/certs"

  if ! acme_bin=$(sb2sub_cert_acme_path); then
    sb2sub_error "未找到 acme.sh"
    return 1
  fi
  if ! sb2sub_cert_has_cloudflare_env; then
    sb2sub_error "尚未配置 Cloudflare Token"
    return 1
  fi

  domain=$(sb2sub_config_get_domain)
  mkdir -p "${cert_dir}"

  # shellcheck disable=SC1090
  source "${SB2SUB_CERT_ENV_FILE}"
  "${acme_bin}" --issue --dns dns_cf -d "${domain}" -d "*.${domain}"
  "${acme_bin}" --install-cert -d "${domain}" \
    --key-file "${cert_dir}/privkey.pem" \
    --fullchain-file "${cert_dir}/fullchain.pem"

  sb2sub_config_set_certificate_file "${cert_dir}/fullchain.pem"
  sb2sub_config_set_certificate_key_file "${cert_dir}/privkey.pem"
}

sb2sub_cert_renew() {
  local acme_bin
  local domain
  local cert_dir="${SB2SUB_CONFIG_DIR}/certs"

  if ! acme_bin=$(sb2sub_cert_acme_path); then
    sb2sub_error "未找到 acme.sh"
    return 1
  fi
  if ! sb2sub_cert_has_cloudflare_env; then
    sb2sub_error "尚未配置 Cloudflare Token"
    return 1
  fi

  domain=$(sb2sub_config_get_domain)
  mkdir -p "${cert_dir}"

  # shellcheck disable=SC1090
  source "${SB2SUB_CERT_ENV_FILE}"
  "${acme_bin}" --renew -d "${domain}"
  "${acme_bin}" --install-cert -d "${domain}" \
    --key-file "${cert_dir}/privkey.pem" \
    --fullchain-file "${cert_dir}/fullchain.pem"
}

sb2sub_show_cert_status() {
  local cert_file="${SB2SUB_CONFIG_DIR}/certs/fullchain.pem"
  local key_file="${SB2SUB_CONFIG_DIR}/certs/privkey.pem"
  printf 'Cloudflare Token: %s\n' "$([[ -s "${SB2SUB_CERT_ENV_FILE}" ]] && printf '已配置' || printf '未配置')"
  printf '证书文件: %s\n' "$([[ -f "${cert_file}" ]] && printf '%s' "${cert_file}" || printf '未找到')"
  printf '私钥文件: %s\n' "$([[ -f "${key_file}" ]] && printf '%s' "${key_file}" || printf '未找到')"
  if [[ -f "${cert_file}" ]] && command -v openssl >/dev/null 2>&1; then
    local end_date
    end_date=$(openssl x509 -enddate -noout -in "${cert_file}" 2>/dev/null | sed 's/^notAfter=//')
    if [[ -n "${end_date}" ]]; then
      printf '证书到期: %s\n' "${end_date}"
    fi
  fi
}
