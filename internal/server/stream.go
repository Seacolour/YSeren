package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	coreconfig "yseren/internal/config"
	"yseren/internal/media"
)

// NewStreamHandler 仅提供配置来源内、扩展名已允许的普通媒体文件。
func NewStreamHandler(conf *coreconfig.Config) http.Handler {
	cloned := conf.Clone()
	return newStreamHandler(&cloned)
}

func newStreamHandler(conf *coreconfig.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		sourceName, relPath, ok := parseStreamRequestPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		sourcePath, ok := conf.GetSourcePath(sourceName)
		if !ok {
			http.NotFound(w, r)
			return
		}

		file, info, resolvedPath, err := openAllowedMediaFile(conf, sourcePath, relPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", media.ContentType(resolvedPath))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		normalizedRange, validRange := normalizeSingleRange(r.Header.Get("Range"), info.Size())
		if !validRange {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(info.Size(), 10))
			WriteError(w, http.StatusRequestedRangeNotSatisfiable, "requested range not satisfiable")
			return
		}
		if normalizedRange != "" {
			r.Header.Set("Range", normalizedRange)
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	})
}

func normalizeSingleRange(value string, totalLength int64) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", true
	}
	if totalLength <= 0 || !strings.HasPrefix(trimmed, "bytes=") || strings.Contains(trimmed, ",") {
		return "", false
	}

	candidate := strings.TrimSpace(strings.TrimPrefix(trimmed, "bytes="))
	parts := strings.SplitN(candidate, "-", 2)
	if len(parts) != 2 {
		return "", false
	}
	startPart := strings.TrimSpace(parts[0])
	endPart := strings.TrimSpace(parts[1])
	if startPart == "" {
		suffixLength, err := strconv.ParseInt(endPart, 10, 64)
		if err != nil || suffixLength <= 0 {
			return "", false
		}
		return "bytes=-" + strconv.FormatInt(suffixLength, 10), true
	}

	start, err := strconv.ParseInt(startPart, 10, 64)
	if err != nil || start < 0 || start >= totalLength {
		return "", false
	}
	if endPart == "" {
		return "bytes=" + strconv.FormatInt(start, 10) + "-", true
	}
	end, err := strconv.ParseInt(endPart, 10, 64)
	if err != nil || end < start {
		return "", false
	}
	return "bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10), true
}

func parseStreamRequestPath(requestPath string) (sourceName, relPath string, ok bool) {
	if !strings.HasPrefix(requestPath, StreamRoutePrefix) {
		return "", "", false
	}
	remainder := strings.TrimPrefix(requestPath, StreamRoutePrefix)
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	segments := strings.Split(parts[1], "/")
	for _, segment := range segments {
		if segment == "" ||
			segment == "." ||
			segment == ".." ||
			strings.IndexByte(segment, '\\') >= 0 ||
			strings.IndexByte(segment, ':') >= 0 ||
			strings.IndexByte(segment, 0) >= 0 {
			return "", "", false
		}
	}
	return parts[0], strings.Join(segments, "/"), true
}

func openAllowedMediaFile(conf *coreconfig.Config, sourcePath, relSlash string) (*os.File, os.FileInfo, string, error) {
	if !conf.IsMediaFile(relSlash) {
		return nil, nil, "", os.ErrNotExist
	}
	relPath := filepath.FromSlash(relSlash)
	if filepath.IsAbs(relPath) || filepath.VolumeName(relPath) != "" {
		return nil, nil, "", os.ErrNotExist
	}

	basePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, nil, "", err
	}
	basePath = filepath.Clean(basePath)
	baseInfo, err := os.Stat(basePath)
	if err != nil || !baseInfo.IsDir() {
		return nil, nil, "", os.ErrNotExist
	}

	targetPath := filepath.Join(basePath, relPath)
	if !coreconfig.IsPathSafe(basePath, targetPath) {
		return nil, nil, "", os.ErrNotExist
	}
	resolvedBase, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		return nil, nil, "", err
	}
	resolvedTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return nil, nil, "", err
	}
	if !coreconfig.IsPathSafe(resolvedBase, resolvedTarget) || !conf.IsMediaFile(resolvedTarget) {
		return nil, nil, "", os.ErrNotExist
	}

	file, err := os.Open(resolvedTarget)
	if err != nil {
		return nil, nil, "", err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, "", err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, nil, "", os.ErrNotExist
	}
	return file, info, resolvedTarget, nil
}

func MediaContentType(filename string) string {
	return media.ContentType(filename)
}
