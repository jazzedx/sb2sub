#!/usr/bin/env bash

sb2sub_write_default_config() {
  sb2sub_ensure_runtime_dirs
  cat >"${SB2SUB_CONFIG_FILE}" <<EOF
server:
  domain: example.com
  certificate_file: ${SB2SUB_CONFIG_DIR}/certs/fullchain.pem
  certificate_key_file: ${SB2SUB_CONFIG_DIR}/certs/privkey.pem
protocols:
  vless:
    enabled: true
    listen: "::"
    listen_port: 443
    server_name: www.cloudflare.com
    reality_public_key: reality-public-key
    reality_private_key: reality-private-key
    reality_short_id: "01234567"
    reality_handshake: www.cloudflare.com
    reality_server_port: 443
    inbound_tag: vless-reality
  hysteria2:
    enabled: true
    listen: "::"
    listen_port: 8443
    up_mbps: 200
    down_mbps: 200
    inbound_tag: hysteria2
    obfs_type: salamander
    obfs_password: change-me
stats:
  listen: 127.0.0.1:10085
EOF
  sb2sub_note "已写入 ${SB2SUB_CONFIG_FILE}"
}

sb2sub_ensure_config_file() {
  if [[ ! -s "${SB2SUB_CONFIG_FILE}" ]]; then
    sb2sub_write_default_config
  fi
}

sb2sub_config_get_server_value() {
  local key=$1
  sb2sub_ensure_config_file
  awk -v key="${key}" '
    /^server:/ { in_server=1; next }
    in_server && /^[^[:space:]]/ { in_server=0 }
    in_server && $1 == key ":" {
      $1=""
      sub(/^[[:space:]]+/, "", $0)
      print
      exit
    }
  ' "${SB2SUB_CONFIG_FILE}"
}

sb2sub_config_set_server_value() {
  local key=$1
  local value=$2
  local tmp_file

  sb2sub_ensure_config_file
  tmp_file=$(mktemp)
  awk -v key="${key}" -v value="${value}" '
    /^server:/ { in_server=1; print; next }
    in_server && /^[^[:space:]]/ { in_server=0 }
    in_server && $1 == key ":" {
      printf "  %s: %s\n", key, value
      found=1
      next
    }
    { print }
    END {
      if (!found) {
        exit 11
      }
    }
  ' "${SB2SUB_CONFIG_FILE}" >"${tmp_file}" || {
    local status=$?
    rm -f "${tmp_file}"
    return "${status}"
  }
  mv "${tmp_file}" "${SB2SUB_CONFIG_FILE}"
}

sb2sub_config_get_protocol_value() {
  local protocol=$1
  local key=$2
  sb2sub_ensure_config_file
  awk -v protocol="${protocol}" -v key="${key}" '
    /^protocols:/ { in_protocols=1; next }
    in_protocols && /^[^[:space:]]/ { in_protocols=0 }
    in_protocols && $1 == protocol ":" { in_target=1; next }
    in_target && /^  [^[:space:]][^:]*:/ { in_target=0 }
    in_target && $1 == key ":" {
      $1=""
      sub(/^[[:space:]]+/, "", $0)
      print
      exit
    }
  ' "${SB2SUB_CONFIG_FILE}"
}

sb2sub_config_set_protocol_value() {
  local protocol=$1
  local key=$2
  local value=$3
  local tmp_file

  sb2sub_ensure_config_file
  tmp_file=$(mktemp)
  awk -v protocol="${protocol}" -v key="${key}" -v value="${value}" '
    /^protocols:/ { in_protocols=1; print; next }
    in_protocols && /^[^[:space:]]/ { in_protocols=0 }
    in_protocols && $1 == protocol ":" {
      in_target=1
      print
      next
    }
    in_target && /^  [^[:space:]][^:]*:/ { in_target=0 }
    in_target && $1 == key ":" {
      printf "    %s: %s\n", key, value
      found=1
      next
    }
    { print }
    END {
      if (!found) {
        exit 11
      }
    }
  ' "${SB2SUB_CONFIG_FILE}" >"${tmp_file}" || {
    local status=$?
    rm -f "${tmp_file}"
    return "${status}"
  }
  mv "${tmp_file}" "${SB2SUB_CONFIG_FILE}"
}

sb2sub_config_get_domain() {
  sb2sub_config_get_server_value "domain"
}

sb2sub_config_set_domain() {
  sb2sub_config_set_server_value "domain" "$1"
}

sb2sub_config_get_certificate_file() {
  sb2sub_config_get_server_value "certificate_file"
}

sb2sub_config_set_certificate_file() {
  sb2sub_config_set_server_value "certificate_file" "$1"
}

sb2sub_config_get_certificate_key_file() {
  sb2sub_config_get_server_value "certificate_key_file"
}

sb2sub_config_set_certificate_key_file() {
  sb2sub_config_set_server_value "certificate_key_file" "$1"
}

sb2sub_config_get_protocol_enabled() {
  sb2sub_config_get_protocol_value "$1" "enabled"
}

sb2sub_config_set_protocol_enabled() {
  sb2sub_config_set_protocol_value "$1" "enabled" "$2"
}

sb2sub_config_get_protocol_port() {
  sb2sub_config_get_protocol_value "$1" "listen_port"
}

sb2sub_config_set_protocol_port() {
  sb2sub_config_set_protocol_value "$1" "listen_port" "$2"
}

sb2sub_generate_config() {
  sb2sub_ensure_config_file
  sb2sub_run_daemon --mode render-singbox --base-dir "${SB2SUB_BASE_DIR}" >"${SB2SUB_SINGBOX_FILE}"
  sb2sub_note "已写入 ${SB2SUB_SINGBOX_FILE}"
}
