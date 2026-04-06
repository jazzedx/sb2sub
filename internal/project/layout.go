package project

import "path/filepath"

type Layout struct {
	BaseDir      string
	ConfigDir    string
	ConfigFile   string
	CertsDir     string
	SingBoxFile  string
	DataDir      string
	DatabaseFile string
	RenderDir    string
	RulesDir     string
	LogDir       string
}

func DefaultLayout(base string) Layout {
	configDir := filepath.Join(base, "etc")
	dataDir := filepath.Join(base, "var")

	return Layout{
		BaseDir:      base,
		ConfigDir:    configDir,
		ConfigFile:   filepath.Join(configDir, "config.yaml"),
		CertsDir:     filepath.Join(configDir, "certs"),
		SingBoxFile:  filepath.Join(configDir, "sing-box.json"),
		DataDir:      dataDir,
		DatabaseFile: filepath.Join(dataDir, "sb2sub.db"),
		RenderDir:    filepath.Join(dataDir, "rendered"),
		RulesDir:     filepath.Join(dataDir, "rules"),
		LogDir:       filepath.Join(base, "log"),
	}
}
