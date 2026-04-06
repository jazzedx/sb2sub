package render

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"sb2sub/internal/config"
	"sb2sub/internal/model"
)

type clashConfig struct {
	AllowLAN           bool               `yaml:"allow-lan"`
	DNS                clashDNSConfig     `yaml:"dns"`
	ExternalController string             `yaml:"external-controller"`
	FindProcessMode    string             `yaml:"find-process-mode"`
	LogLevel           string             `yaml:"log-level"`
	MixedPort          int                `yaml:"mixed-port"`
	Mode               string             `yaml:"mode"`
	UnifiedDelay       bool               `yaml:"unified-delay"`
	Tun                clashTunConfig     `yaml:"tun"`
	Sniffer            clashSnifferConfig `yaml:"sniffer"`
	Proxies            []clashProxy       `yaml:"proxies"`
	ProxyGroups        []clashProxyGroup  `yaml:"proxy-groups"`
	RuleProviders      clashRuleProviders `yaml:"rule-providers"`
	Rules              []string           `yaml:"rules"`
}

type clashProxy struct {
	Name              string            `yaml:"name"`
	Type              string            `yaml:"type"`
	Server            string            `yaml:"server"`
	Port              int               `yaml:"port"`
	Network           string            `yaml:"network,omitempty"`
	UUID              string            `yaml:"uuid,omitempty"`
	Flow              string            `yaml:"flow,omitempty"`
	TLS               *bool             `yaml:"tls,omitempty"`
	UDP               *bool             `yaml:"udp,omitempty"`
	ServerName        string            `yaml:"servername,omitempty"`
	ClientFingerprint string            `yaml:"client-fingerprint,omitempty"`
	RealityOpts       *clashRealityOpts `yaml:"reality-opts,omitempty"`
	Password          string            `yaml:"password,omitempty"`
	SNI               string            `yaml:"sni,omitempty"`
	Up                string            `yaml:"up,omitempty"`
	Down              string            `yaml:"down,omitempty"`
	SkipCertVerify    *bool             `yaml:"skip-cert-verify,omitempty"`
	Obfs              string            `yaml:"obfs,omitempty"`
	ObfsPassword      string            `yaml:"obfs-password,omitempty"`
}

type clashRealityOpts struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
}

type clashProxyGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

type clashTunConfig struct {
	Enable              bool     `yaml:"enable"`
	Stack               string   `yaml:"stack"`
	AutoRoute           bool     `yaml:"auto-route"`
	AutoDetectInterface bool     `yaml:"auto-detect-interface"`
	DNSHijack           []string `yaml:"dns-hijack"`
}

type clashSnifferConfig struct {
	Enable bool             `yaml:"enable"`
	Sniff  clashSniffConfig `yaml:"sniff"`
}

type clashSniffConfig struct {
	HTTP clashHTTPPortGroup `yaml:"HTTP"`
	TLS  clashPortGroupInt  `yaml:"TLS"`
	QUIC clashPortGroupInt  `yaml:"QUIC"`
}

type clashHTTPPortGroup struct {
	Ports               []string `yaml:"ports"`
	OverrideDestination bool     `yaml:"override-destination"`
}

type clashPortGroupInt struct {
	Ports []int `yaml:"ports"`
}

type clashPortGroupString struct {
	Ports []string `yaml:"ports"`
}

type clashDNSConfig struct {
	DefaultNameserver   []string               `yaml:"default-nameserver"`
	Enable              bool                   `yaml:"enable"`
	EnhancedMode        string                 `yaml:"enhanced-mode"`
	FakeIPFilter        []string               `yaml:"fake-ip-filter"`
	FakeIPRange         string                 `yaml:"fake-ip-range"`
	Fallback            []string               `yaml:"fallback"`
	FallbackFilter      clashDNSFallbackFilter `yaml:"fallback-filter"`
	IPv6                bool                   `yaml:"ipv6"`
	Nameserver          []string               `yaml:"nameserver"`
	ProxyServerResolver []string               `yaml:"proxy-server-nameserver"`
}

type clashDNSFallbackFilter struct {
	GeoIP     bool   `yaml:"geoip"`
	GeoIPCode string `yaml:"geoip-code"`
}

type clashRuleProviders struct {
	Ads     clashRuleProvider `yaml:"ads"`
	Direct  clashRuleProvider `yaml:"direct"`
	Private clashRuleProvider `yaml:"private"`
	Proxy   clashRuleProvider `yaml:"proxy"`
	GFW     clashRuleProvider `yaml:"gfw"`
	CNCIDR  clashRuleProvider `yaml:"cncidr"`
	LANCIDR clashRuleProvider `yaml:"lancidr"`
}

type clashRuleProvider struct {
	Type     string `yaml:"type"`
	Behavior string `yaml:"behavior"`
	URL      string `yaml:"url"`
	Path     string `yaml:"path"`
	Interval int    `yaml:"interval"`
}

func RenderClash(cfg config.Config, user model.User) ([]byte, error) {
	nodeNames := userNodeNames(user)
	proxies := make([]clashProxy, 0, 2)

	if user.Enabled && user.VLESSEnabled {
		proxies = append(proxies, clashProxy{
			Name:              fmt.Sprintf("%s-VLESS-Reality", user.Username),
			Type:              "vless",
			Server:            cfg.Server.Domain,
			Port:              cfg.Protocols.VLESS.ListenPort,
			Network:           "tcp",
			UUID:              user.VLESSUUID,
			Flow:              "xtls-rprx-vision",
			TLS:               boolPtr(true),
			UDP:               boolPtr(true),
			ServerName:        cfg.Protocols.VLESS.ServerName,
			ClientFingerprint: "chrome",
			RealityOpts: &clashRealityOpts{
				PublicKey: cfg.Protocols.VLESS.RealityPublicKey,
				ShortID:   cfg.Protocols.VLESS.RealityShortID,
			},
		})
	}

	if user.Enabled && user.Hysteria2Enabled {
		proxies = append(proxies, clashProxy{
			Name:           fmt.Sprintf("%s-Hysteria2", user.Username),
			Type:           "hysteria2",
			Server:         cfg.Server.Domain,
			Port:           cfg.Protocols.Hysteria2.ListenPort,
			Password:       user.Hysteria2Password,
			SNI:            cfg.Server.Domain,
			Up:             fmt.Sprintf("%d Mbps", cfg.Protocols.Hysteria2.UpMbps),
			Down:           fmt.Sprintf("%d Mbps", cfg.Protocols.Hysteria2.DownMbps),
			SkipCertVerify: boolPtr(false),
			Obfs:           cfg.Protocols.Hysteria2.ObfsType,
			ObfsPassword:   cfg.Protocols.Hysteria2.ObfsPassword,
		})
	}

	manualProxies := append([]string{}, nodeNames...)
	if len(manualProxies) == 0 {
		manualProxies = append(manualProxies, "DIRECT")
	}

	doc := clashConfig{
		AllowLAN: false,
		DNS: clashDNSConfig{
			DefaultNameserver: []string{"8.8.8.8", "1.1.1.1"},
			Enable:            true,
			EnhancedMode:      "fake-ip",
			FakeIPFilter:      []string{"*.lan", "localhost", "*.local"},
			FakeIPRange:       "198.18.0.1/16",
			Fallback:          []string{"https://1.1.1.1/dns-query", "https://8.8.8.8/dns-query"},
			FallbackFilter: clashDNSFallbackFilter{
				GeoIP:     true,
				GeoIPCode: "CN",
			},
			IPv6:                false,
			Nameserver:          []string{"https://doh.pub/dns-query", "https://1.0.0.1/dns-query"},
			ProxyServerResolver: []string{"https://1.1.1.1/dns-query", "https://8.8.8.8/dns-query"},
		},
		ExternalController: "127.0.0.1:9090",
		FindProcessMode:    "strict",
		LogLevel:           "info",
		MixedPort:          7890,
		Mode:               "rule",
		UnifiedDelay:       true,
		Tun: clashTunConfig{
			Enable:              true,
			Stack:               "mixed",
			AutoRoute:           true,
			AutoDetectInterface: true,
			DNSHijack:           []string{"any:53"},
		},
		Sniffer: clashSnifferConfig{
			Enable: true,
			Sniff: clashSniffConfig{
				HTTP: clashHTTPPortGroup{
					Ports:               []string{"80", "8080-8880"},
					OverrideDestination: true,
				},
				TLS:  clashPortGroupInt{Ports: []int{443, 8443}},
				QUIC: clashPortGroupInt{Ports: []int{443, 8443}},
			},
		},
		Proxies: proxies,
		ProxyGroups: []clashProxyGroup{
			{Name: "手动切换", Type: "select", Proxies: manualProxies},
			{Name: "国内直连", Type: "select", Proxies: []string{"DIRECT"}},
			{Name: "国外分流", Type: "select", Proxies: []string{"手动切换", "DIRECT"}},
			{Name: "广告拦截", Type: "select", Proxies: []string{"REJECT", "DIRECT"}},
			{Name: "漏网之鱼", Type: "select", Proxies: []string{"手动切换", "DIRECT"}},
		},
		RuleProviders: clashRuleProviders{
			Ads: clashRuleProvider{
				Type:     "http",
				Behavior: "domain",
				URL:      "https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/Advertising/Advertising.yaml",
				Path:     "./ruleset/ads.yaml",
				Interval: 86400,
			},
			Direct: clashRuleProvider{
				Type:     "http",
				Behavior: "domain",
				URL:      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/direct.txt",
				Path:     "./ruleset/direct.yaml",
				Interval: 86400,
			},
			Private: clashRuleProvider{
				Type:     "http",
				Behavior: "domain",
				URL:      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/private.txt",
				Path:     "./ruleset/private.yaml",
				Interval: 86400,
			},
			Proxy: clashRuleProvider{
				Type:     "http",
				Behavior: "domain",
				URL:      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/proxy.txt",
				Path:     "./ruleset/proxy.yaml",
				Interval: 86400,
			},
			GFW: clashRuleProvider{
				Type:     "http",
				Behavior: "domain",
				URL:      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/gfw.txt",
				Path:     "./ruleset/gfw.yaml",
				Interval: 86400,
			},
			CNCIDR: clashRuleProvider{
				Type:     "http",
				Behavior: "ipcidr",
				URL:      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/cncidr.txt",
				Path:     "./ruleset/cncidr.yaml",
				Interval: 86400,
			},
			LANCIDR: clashRuleProvider{
				Type:     "http",
				Behavior: "ipcidr",
				URL:      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/lancidr.txt",
				Path:     "./ruleset/lancidr.yaml",
				Interval: 86400,
			},
		},
		Rules: []string{
			"RULE-SET,ads,广告拦截",
			"RULE-SET,private,国内直连",
			"RULE-SET,direct,国内直连",
			"RULE-SET,lancidr,国内直连",
			"RULE-SET,cncidr,国内直连",
			"RULE-SET,gfw,国外分流",
			"RULE-SET,proxy,国外分流",
			"MATCH,漏网之鱼",
		},
	}

	return yaml.Marshal(doc)
}

func userNodeNames(user model.User) []string {
	names := make([]string, 0, 2)
	if user.Enabled && user.VLESSEnabled {
		names = append(names, fmt.Sprintf("%s-VLESS-Reality", user.Username))
	}
	if user.Enabled && user.Hysteria2Enabled {
		names = append(names, fmt.Sprintf("%s-Hysteria2", user.Username))
	}
	return names
}

func boolPtr(v bool) *bool {
	return &v
}
