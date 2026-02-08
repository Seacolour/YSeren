package main

import (
	"errors"
	"fmt"
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
		Port     int    `yaml:"port"`
		LogLevel string `yaml:"log_level"` // debug, info, warn, error
	} `yaml:"server"`
	Sources []Source `yaml:"sources"`
	// MediaExtensions 支持的媒体文件扩展名（含点号），为空时使用默认值
	MediaExtensions []string `yaml:"media_extensions"`
}

// DefaultVideoExtensions 默认支持的视频格式
var DefaultVideoExtensions = []string{".mp4", ".mkv", ".webm", ".mov", ".m4v", ".avi"}

// DefaultAudioExtensions 默认支持的音频格式
var DefaultAudioExtensions = []string{".mp3", ".flac", ".wav", ".aac", ".ogg", ".m4a", ".wma"}

// DefaultMediaExtensions 所有默认支持的媒体格式
var DefaultMediaExtensions = append(append([]string{}, DefaultVideoExtensions...), DefaultAudioExtensions...)

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

	// 端口默认值
	if conf.Server.Port == 0 {
		conf.Server.Port = 1479
	}

	// 如果没有配置 media_extensions，使用默认值
	if len(conf.MediaExtensions) == 0 {
		conf.MediaExtensions = DefaultMediaExtensions
	} else {
		// 确保扩展名都是小写且带点号
		for i, ext := range conf.MediaExtensions {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			conf.MediaExtensions[i] = ext
		}
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

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return &conf, nil
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

// GetSourcePath 根据 source 名称查找对应的路径
// 返回 (路径, true) 或 ("", false)
func (c *Config) GetSourcePath(name string) (string, bool) {
	for _, s := range c.Sources {
		if s.Name == name {
			return s.Path, true
		}
	}
	return "", false
}

// Validate 做最小必要的配置校验，避免路由/安全/可维护性问题。
// 设计原则：轻量、可预期——发现明显错误就直接报错，避免运行期“静默空结果”。
func (c *Config) Validate() error {
	seen := make(map[string]struct{}, len(c.Sources))
	for i, s := range c.Sources {
		name := strings.TrimSpace(s.Name)
		path := strings.TrimSpace(s.Path)
		if path == "" {
			return fmt.Errorf("sources[%d].path 不能为空", i)
		}
		if name == "" {
			return fmt.Errorf("sources[%d].name 不能为空", i)
		}
		// name 作为 URL 路由片段：禁止斜杠，避免被拆成多级路径导致路由错乱
		if strings.ContainsAny(name, "/\\") {
			return fmt.Errorf("sources[%d].name 不允许包含 '/' 或 '\\\\'：%q", i, name)
		}
		// 非严格安全校验：禁止明显的路径穿越片段，避免误导
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

// IsMediaFile 检查文件名是否为配置中支持的媒体格式
func (c *Config) IsMediaFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, e := range c.MediaExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// IsAudioFile 检查文件名是否为音频格式（用于前端区分播放器类型）
func IsAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, e := range DefaultAudioExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// IsPathSafe 检查 targetPath 是否安全地位于 basePath 内部（防止路径穿越）
// 两个路径都应该是绝对路径
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

	// 确保 target 在 base 内部
	// 需要处理 Windows 大小写不敏感的情况
	baseLower := strings.ToLower(baseAbs)
	targetLower := strings.ToLower(targetAbs)

	// target 必须是 base 本身或其子目录
	if targetLower == baseLower {
		return true
	}
	return strings.HasPrefix(targetLower, baseLower+string(filepath.Separator))
}
