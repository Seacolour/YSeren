package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Source struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// Config 对应 v-link.yaml
type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Sources []Source `yaml:"sources"`
}

func LoadConfig(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Config
	err = yaml.Unmarshal(buf, &conf)
	if err != nil {
		return nil, err
	}

	// 允许 sources 只写 path，不写 name（自动用目录名作为 name）
	for i := range conf.Sources {
		conf.Sources[i].Path = strings.TrimSpace(conf.Sources[i].Path)
		conf.Sources[i].Name = strings.TrimSpace(conf.Sources[i].Name)
		if conf.Sources[i].Name == "" && conf.Sources[i].Path != "" {
			// Windows 路径：用 Clean + Base
			p := filepath.Clean(conf.Sources[i].Path)
			base := filepath.Base(p)
			if base == "." || base == string(filepath.Separator) {
				base = "videos"
			}
			conf.Sources[i].Name = base
		}
	}
	return &conf, err
}
