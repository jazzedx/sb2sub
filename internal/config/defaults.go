package config

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Domain:             "example.com",
			CertificateFile:    "/etc/sb2sub/certs/fullchain.pem",
			CertificateKeyFile: "/etc/sb2sub/certs/privkey.pem",
		},
		Protocols: ProtocolsConfig{
			VLESS: VLESSConfig{
				Enabled:           true,
				Listen:            "::",
				ListenPort:        443,
				ServerName:        "www.cloudflare.com",
				RealityPublicKey:  "reality-public-key",
				RealityPrivateKey: "reality-private-key",
				RealityShortID:    "01234567",
				RealityHandshake:  "www.cloudflare.com",
				RealityServerPort: 443,
				InboundTag:        "vless-reality",
			},
			Hysteria2: Hysteria2Config{
				Enabled:      true,
				Listen:       "::",
				ListenPort:   8443,
				UpMbps:       200,
				DownMbps:     200,
				InboundTag:   "hysteria2",
				ObfsType:     "salamander",
				ObfsPassword: "change-me",
			},
		},
		Stats: StatsConfig{
			Listen: "127.0.0.1:10085",
		},
	}
}
