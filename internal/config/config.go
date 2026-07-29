package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"yseren/internal/media"
)

const DefaultPort = 1479

type Source struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type ServerConfig struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

// Config 是 Headless、Desktop 和 Go Core 共用的核心配置。
type Config struct {
	Server          ServerConfig `yaml:"server"`
	Sources         []Source     `yaml:"sources"`
	MediaExtensions []string     `yaml:"media_extensions"`
	AudioExtensions []string     `yaml:"audio_extensions"`
}

var DefaultVideoExtensions = media.DefaultVideoExtensions
var DefaultAudioExtensions = media.DefaultAudioExtensions
var KnownAudioExtensions = media.KnownAudioExtensions
var DefaultMediaExtensions = media.DefaultMediaExtensions

// PreferredConfigNames 是 YSeren 默认的配置文件名。
var PreferredConfigNames = []string{"yseren.yaml", "yseren.yml"}

func LoadConfig(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Config
	if err := yaml.Unmarshal(buf, &conf); err != nil {
		return nil, err
	}
	conf.ApplyDefaults()

	if err := conf.Validate(); err != nil {
		return nil, err
	}
	return &conf, nil
}

// DefaultConfig returns a ready-to-run configuration for callers that do not
// load YAML first, such as the Desktop first-run flow.
func DefaultConfig() Config {
	conf := Config{Server: ServerConfig{Port: DefaultPort}}
	conf.ApplyDefaults()
	return conf
}

// ApplyDefaults normalises the same defaults used by LoadConfig. It is useful
// for GUI callers that build a configuration in memory.
func (c *Config) ApplyDefaults() {
	if c == nil {
		return
	}
	if c.Server.Port == 0 {
		c.Server.Port = DefaultPort
	}
	c.MediaExtensions = media.NormalizeExtensions(c.MediaExtensions)
	c.AudioExtensions = media.NormalizeExtensions(c.AudioExtensions)
	if len(c.MediaExtensions) == 0 {
		c.MediaExtensions = media.MergeExtensions(DefaultMediaExtensions, c.AudioExtensions)
	} else {
		c.MediaExtensions = media.MergeExtensions(c.MediaExtensions, c.AudioExtensions)
	}

	for i := range c.Sources {
		c.Sources[i].Path = strings.TrimSpace(c.Sources[i].Path)
		c.Sources[i].Name = strings.TrimSpace(c.Sources[i].Name)
		if c.Sources[i].Name == "" && c.Sources[i].Path != "" {
			cleaned := filepath.Clean(c.Sources[i].Path)
			base := filepath.Base(cleaned)
			if base == "." || base == string(filepath.Separator) {
				base = "videos"
			}
			c.Sources[i].Name = base
		}
	}
}

// SaveConfig validates and atomically replaces a YAML configuration file.
func SaveConfig(path string, conf Config) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("配置文件路径不能为空")
	}
	conf.ApplyDefaults()
	if err := conf.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(&conf)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("解析配置文件路径失败: %w", err)
	}
	directory := filepath.Dir(absolutePath)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".yseren-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时配置文件失败: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("设置临时配置文件权限失败: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("写入临时配置文件失败: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("同步临时配置文件失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭临时配置文件失败: %w", err)
	}
	if err := os.Rename(temporaryPath, absolutePath); err != nil {
		return fmt.Errorf("替换配置文件失败: %w", err)
	}
	removeTemporary = false
	return nil
}

// LoadConfigAuto 保留原有发现顺序：显式路径、当前目录、可执行文件目录。
func LoadConfigAuto(explicitPath string) (*Config, string, error) {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		conf, err := LoadConfig(explicitPath)
		return conf, explicitPath, err
	}

	for _, name := range PreferredConfigNames {
		if fileExists(name) {
			conf, err := LoadConfig(name)
			return conf, name, err
		}
	}

	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		for _, name := range PreferredConfigNames {
			path := filepath.Join(executableDir, name)
			if fileExists(path) {
				conf, err := LoadConfig(path)
				return conf, path, err
			}
		}
	}

	return nil, "", errors.New("未找到配置文件：请在当前目录或 exe 同目录放置 yseren.yaml/yseren.yml，或使用 -config 指定路径")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (c *Config) GetSourcePath(name string) (string, bool) {
	for _, source := range c.Sources {
		if source.Name == name {
			return source.Path, true
		}
	}
	return "", false
}

// Validate 执行可在启动前同步反馈给 Headless 或 Desktop 的核心校验。
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("配置不能为空")
	}
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port 必须在 0 到 65535 之间：%d", c.Server.Port)
	}

	seen := make(map[string]struct{}, len(c.Sources))
	for i, source := range c.Sources {
		name := strings.TrimSpace(source.Name)
		path := strings.TrimSpace(source.Path)
		if path == "" {
			return fmt.Errorf("sources[%d].path 不能为空", i)
		}
		if name == "" {
			return fmt.Errorf("sources[%d].name 不能为空", i)
		}
		if strings.ContainsAny(name, "/\\") {
			return fmt.Errorf("sources[%d].name 不允许包含 '/' 或 '\\'：%q", i, name)
		}
		if name == "." || name == ".." || strings.Contains(name, "..") {
			return fmt.Errorf("sources[%d].name 不合法：%q", i, name)
		}

		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("sources[%d].name 重复（忽略大小写）：%q", i, name)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (c *Config) IsMediaFile(filename string) bool {
	return media.HasExtension(filename, c.MediaExtensions)
}

func (c *Config) IsAudioFile(filename string) bool {
	return media.HasExtension(filename, c.AudioExtensions) || media.HasExtension(filename, KnownAudioExtensions)
}

// Clone 返回适合由独立 Runtime 实例持有的配置副本。
func (c *Config) Clone() Config {
	if c == nil {
		return Config{}
	}
	clone := *c
	clone.Sources = append([]Source(nil), c.Sources...)
	clone.MediaExtensions = append([]string(nil), c.MediaExtensions...)
	clone.AudioExtensions = append([]string(nil), c.AudioExtensions...)
	return clone
}

// IsPathSafe 检查 targetPath 是否位于 basePath 内部。
func IsPathSafe(basePath, targetPath string) bool {
	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	baseAbs = filepath.Clean(baseAbs)
	targetAbs = filepath.Clean(targetAbs)
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if filepath.IsAbs(rel) || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
