package main

import (
	"errors"
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

// LoadConfigAuto:
// - 如果 explicitPath 非空：直接读取该文件
// - 否则按顺序查找：当前工作目录 -> 可执行文件所在目录
// - 文件名：v-link.yaml 或 v-link.yml
func LoadConfigAuto(explicitPath string) (*Config, string, error) {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		c, err := LoadConfig(explicitPath)
		return c, explicitPath, err
	}

	candidates := []string{"v-link.yaml", "v-link.yml"}
	// 1) 当前工作目录
	for _, name := range candidates {
		if fileExists(name) {
			c, err := LoadConfig(name)
			return c, name, err
		}
	}

	// 2) exe 所在目录
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, name := range candidates {
			p := filepath.Join(exeDir, name)
			if fileExists(p) {
				c, err := LoadConfig(p)
				return c, p, err
			}
		}
	}

	return nil, "", errors.New("未找到配置文件：请在当前目录或 exe 同目录放置 v-link.yaml/v-link.yml，或使用 -config 指定路径")
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
