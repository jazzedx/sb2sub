package render

import (
	"encoding/json"

	"sb2sub/internal/config"
	"sb2sub/internal/model"
)

type RuntimeUser struct {
	Name              string
	UUID              string
	Hysteria2Password string
	Enabled           bool
	VLESSEnabled      bool
	Hysteria2Enabled  bool
}

func RenderSingBox(cfg config.Config, users []RuntimeUser) ([]byte, error) {
	type vlessUser struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
		Flow string `json:"flow"`
	}

	type hysteriaUser struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	inbounds := make([]any, 0, 2)
	vlessUsers := make([]vlessUser, 0, len(users))
	hysteriaUsers := make([]hysteriaUser, 0, len(users))
	statsUsers := make([]string, 0, len(users))

	for _, user := range users {
		if !user.Enabled {
			continue
		}
		statsUsers = append(statsUsers, user.Name)
		if cfg.Protocols.VLESS.Enabled && user.VLESSEnabled {
			vlessUsers = append(vlessUsers, vlessUser{
				Name: user.Name,
				UUID: user.UUID,
				Flow: "xtls-rprx-vision",
			})
		}
		if cfg.Protocols.Hysteria2.Enabled && user.Hysteria2Enabled {
			hysteriaUsers = append(hysteriaUsers, hysteriaUser{
				Name:     user.Name,
				Password: user.Hysteria2Password,
			})
		}
	}

	if cfg.Protocols.VLESS.Enabled {
		inbounds = append(inbounds, map[string]any{
			"type":        "vless",
			"tag":         cfg.Protocols.VLESS.InboundTag,
			"listen":      cfg.Protocols.VLESS.Listen,
			"listen_port": cfg.Protocols.VLESS.ListenPort,
			"users":       vlessUsers,
			"tls": map[string]any{
				"enabled":          true,
				"certificate_path": cfg.Server.CertificateFile,
				"key_path":         cfg.Server.CertificateKeyFile,
				"server_name":      cfg.Protocols.VLESS.ServerName,
				"reality": map[string]any{
					"enabled":     true,
					"private_key": cfg.Protocols.VLESS.RealityPrivateKey,
					"short_id":    []string{cfg.Protocols.VLESS.RealityShortID},
					"handshake": map[string]any{
						"server":      cfg.Protocols.VLESS.RealityHandshake,
						"server_port": cfg.Protocols.VLESS.RealityServerPort,
					},
				},
			},
		})
	}

	if cfg.Protocols.Hysteria2.Enabled {
		inbounds = append(inbounds, map[string]any{
			"type":        "hysteria2",
			"tag":         cfg.Protocols.Hysteria2.InboundTag,
			"listen":      cfg.Protocols.Hysteria2.Listen,
			"listen_port": cfg.Protocols.Hysteria2.ListenPort,
			"users":       hysteriaUsers,
			"up_mbps":     cfg.Protocols.Hysteria2.UpMbps,
			"down_mbps":   cfg.Protocols.Hysteria2.DownMbps,
			"obfs": map[string]any{
				"type":     cfg.Protocols.Hysteria2.ObfsType,
				"password": cfg.Protocols.Hysteria2.ObfsPassword,
			},
			"tls": map[string]any{
				"enabled":          true,
				"certificate_path": cfg.Server.CertificateFile,
				"key_path":         cfg.Server.CertificateKeyFile,
			},
		})
	}

	doc := map[string]any{
		"log": map[string]any{
			"level": "info",
		},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"route": map[string]any{
			"auto_detect_interface": true,
		},
		"experimental": map[string]any{
			"v2ray_api": map[string]any{
				"listen": cfg.Stats.Listen,
				"stats": map[string]any{
					"enabled":  true,
					"inbounds": []string{cfg.Protocols.VLESS.InboundTag, cfg.Protocols.Hysteria2.InboundTag},
					"users":    statsUsers,
				},
			},
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func RuntimeUsersFromModel(users []model.User) []RuntimeUser {
	runtimeUsers := make([]RuntimeUser, 0, len(users))
	for _, user := range users {
		runtimeUsers = append(runtimeUsers, RuntimeUser{
			Name:              user.Username,
			UUID:              user.VLESSUUID,
			Hysteria2Password: user.Hysteria2Password,
			Enabled:           user.Enabled,
			VLESSEnabled:      user.VLESSEnabled,
			Hysteria2Enabled:  user.Hysteria2Enabled,
		})
	}
	return runtimeUsers
}
