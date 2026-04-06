package render

import (
	"fmt"
	"net/url"
	"strings"

	"sb2sub/internal/config"
	"sb2sub/internal/model"
)

func RenderShadowrocket(cfg config.Config, user model.User) ([]byte, error) {
	lines := make([]string, 0, 2)
	if user.Enabled && user.VLESSEnabled {
		query := url.Values{}
		query.Set("encryption", "none")
		query.Set("security", "reality")
		query.Set("sni", cfg.Protocols.VLESS.ServerName)
		query.Set("pbk", cfg.Protocols.VLESS.RealityPublicKey)
		query.Set("sid", cfg.Protocols.VLESS.RealityShortID)
		query.Set("fp", "chrome")
		query.Set("type", "tcp")
		query.Set("flow", "xtls-rprx-vision")

		lines = append(lines, fmt.Sprintf(
			"vless://%s@%s:%d?%s#%s",
			user.VLESSUUID,
			cfg.Server.Domain,
			cfg.Protocols.VLESS.ListenPort,
			query.Encode(),
			url.QueryEscape(fmt.Sprintf("%s-VLESS-Reality", user.Username)),
		))
	}

	if user.Enabled && user.Hysteria2Enabled {
		query := url.Values{}
		query.Set("sni", cfg.Server.Domain)
		query.Set("obfs", cfg.Protocols.Hysteria2.ObfsType)
		query.Set("obfs-password", cfg.Protocols.Hysteria2.ObfsPassword)
		query.Set("upmbps", fmt.Sprintf("%d", cfg.Protocols.Hysteria2.UpMbps))
		query.Set("downmbps", fmt.Sprintf("%d", cfg.Protocols.Hysteria2.DownMbps))

		lines = append(lines, fmt.Sprintf(
			"hysteria2://%s@%s:%d?%s#%s",
			user.Hysteria2Password,
			cfg.Server.Domain,
			cfg.Protocols.Hysteria2.ListenPort,
			query.Encode(),
			url.QueryEscape(fmt.Sprintf("%s-Hysteria2", user.Username)),
		))
	}

	return []byte(strings.Join(lines, "\n")), nil
}
