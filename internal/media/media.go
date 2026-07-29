package media

import (
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// DefaultVideoExtensions 是默认允许扫描和共享的视频扩展名。
var DefaultVideoExtensions = []string{".mp4", ".mkv", ".webm", ".mov", ".m4v", ".avi"}

// DefaultAudioExtensions 是默认允许扫描和共享的音频扩展名。
var DefaultAudioExtensions = []string{".mp3", ".flac", ".wav", ".aac", ".ogg", ".m4a", ".wma"}

// KnownAudioExtensions 用于在自定义媒体扩展名时继续正确区分音频文件。
var KnownAudioExtensions = MergeExtensions(DefaultAudioExtensions, []string{
	".opus", ".oga", ".weba", ".aif", ".aiff", ".ape", ".alac", ".amr", ".ac3", ".dts", ".mka", ".tta",
})

// DefaultMediaExtensions 是默认允许扫描和共享的全部媒体扩展名。
var DefaultMediaExtensions = MergeExtensions(DefaultVideoExtensions, DefaultAudioExtensions)

// Classifier 描述目录扫描所需的最小媒体分类能力。
type Classifier interface {
	IsMediaFile(filename string) bool
	IsAudioFile(filename string) bool
}

// Entry 是与传输协议无关的媒体索引项。
type Entry struct {
	Name      string
	RelPath   string
	Size      int64
	ModTime   int64
	MediaType string
}

// NormalizeExtensions 将扩展名规范为小写、带点且去重的形式。
func NormalizeExtensions(exts []string) []string {
	if len(exts) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(exts))
	out := make([]string, 0, len(exts))
	for _, ext := range exts {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if _, ok := seen[ext]; ok {
			continue
		}
		seen[ext] = struct{}{}
		out = append(out, ext)
	}
	return out
}

// MergeExtensions 按输入顺序合并扩展名并去重。
func MergeExtensions(groups ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, group := range groups {
		for _, ext := range group {
			if ext == "" {
				continue
			}
			if _, ok := seen[ext]; ok {
				continue
			}
			seen[ext] = struct{}{}
			out = append(out, ext)
		}
	}
	return out
}

// HasExtension 判断文件扩展名是否在允许列表内。
func HasExtension(filename string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range extensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

// Scan 扫描 root 下可识别的媒体文件，并返回与 HTTP 无关的索引项。
// 不可访问的单个文件或子目录会被跳过，以保持原有乐观扫描语义。
func Scan(root string, classifier Classifier) ([]Entry, error) {
	entries := make([]Entry, 0, 1024)
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(resolvedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		if !classifier.IsMediaFile(entry.Name()) {
			return nil
		}

		resolvedPath := path
		if entry.Type()&os.ModeSymlink != 0 {
			resolvedPath, err = filepath.EvalSymlinks(path)
			if err != nil || !isPathWithin(resolvedRoot, resolvedPath) {
				return nil
			}
		}
		if !classifier.IsMediaFile(resolvedPath) {
			return nil
		}

		info, err := os.Stat(resolvedPath)
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(resolvedPath)
		if err != nil {
			return nil
		}
		_ = file.Close()

		relPath, err := filepath.Rel(resolvedRoot, path)
		if err != nil {
			return nil
		}

		mediaType := "video"
		if classifier.IsAudioFile(resolvedPath) {
			mediaType = "audio"
		}
		entries = append(entries, Entry{
			Name:      entry.Name(),
			RelPath:   filepath.ToSlash(relPath),
			Size:      info.Size(),
			ModTime:   info.ModTime().Unix(),
			MediaType: mediaType,
		})
		return nil
	})
	return entries, err
}

func isPathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(relative) || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// ContentType 返回常见媒体扩展名的稳定 MIME，避免依赖平台注册表差异。
func ContentType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".wav":
		return "audio/wav"
	case ".aac":
		return "audio/aac"
	case ".ogg", ".oga", ".opus":
		return "audio/ogg"
	case ".m4a", ".alac":
		return "audio/mp4"
	case ".wma":
		return "audio/x-ms-wma"
	case ".weba":
		return "audio/webm"
	case ".aif", ".aiff":
		return "audio/aiff"
	case ".ape":
		return "audio/x-ape"
	case ".amr":
		return "audio/amr"
	case ".ac3":
		return "audio/ac3"
	case ".dts":
		return "audio/vnd.dts"
	case ".mka":
		return "audio/x-matroska"
	case ".tta":
		return "audio/x-tta"
	}

	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}
