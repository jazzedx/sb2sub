package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `json:"server" yaml:"server"`
	Protocols ProtocolsConfig `json:"protocols" yaml:"protocols"`
	Stats     StatsConfig     `json:"stats" yaml:"stats"`
}

type ServerConfig struct {
	Domain             string `json:"domain" yaml:"domain"`
	CertificateFile    string `json:"certificate_file" yaml:"certificate_file"`
	CertificateKeyFile string `json:"certificate_key_file" yaml:"certificate_key_file"`
}

type ProtocolsConfig struct {
	VLESS     VLESSConfig     `json:"vless" yaml:"vless"`
	Hysteria2 Hysteria2Config `json:"hysteria2" yaml:"hysteria2"`
}

type VLESSConfig struct {
	Enabled           bool   `json:"enabled" yaml:"enabled"`
	Listen            string `json:"listen" yaml:"listen"`
	ListenPort        int    `json:"listen_port" yaml:"listen_port"`
	ServerName        string `json:"server_name" yaml:"server_name"`
	RealityPublicKey  string `json:"reality_public_key" yaml:"reality_public_key"`
	RealityPrivateKey string `json:"reality_private_key" yaml:"reality_private_key"`
	RealityShortID    string `json:"reality_short_id" yaml:"reality_short_id"`
	RealityHandshake  string `json:"reality_handshake" yaml:"reality_handshake"`
	RealityServerPort int    `json:"reality_server_port" yaml:"reality_server_port"`
	InboundTag        string `json:"inbound_tag" yaml:"inbound_tag"`
}

type Hysteria2Config struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	Listen       string `json:"listen" yaml:"listen"`
	ListenPort   int    `json:"listen_port" yaml:"listen_port"`
	UpMbps       int    `json:"up_mbps" yaml:"up_mbps"`
	DownMbps     int    `json:"down_mbps" yaml:"down_mbps"`
	InboundTag   string `json:"inbound_tag" yaml:"inbound_tag"`
	ObfsType     string `json:"obfs_type" yaml:"obfs_type"`
	ObfsPassword string `json:"obfs_password" yaml:"obfs_password"`
}

type StatsConfig struct {
	Listen string `json:"listen" yaml:"listen"`
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
