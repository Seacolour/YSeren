package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MediaItem struct {
	Source    string `json:"source"`
	Name      string `json:"name"`
	RelPath   string `json:"relPath"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	ModTime   int64  `json:"modTime"`   // unix seconds
	MediaType string `json:"mediaType"` // "video" | "audio"
}

// VideoItem is an alias for backwards compatibility
type VideoItem = MediaItem

type VideosResponse struct {
	GeneratedAt int64       `json:"generatedAt"`
	Total       int         `json:"total"`
	Items       []VideoItem `json:"items"`
}

// videosCache 用于缓存视频列表，5秒 TTL，使用 singleflight 防止缓存击穿
var videosCache = NewCache[[]MediaItem](5 * time.Second)

func ListVideosHandler(conf *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		// query params
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		source := strings.TrimSpace(r.URL.Query().Get("source"))
		refresh := r.URL.Query().Get("refresh") == "1"

		limit := parseIntDefault(r.URL.Query().Get("limit"), 200)
		if limit < 1 {
			limit = 1
		}
		if limit > 2000 {
			limit = 2000
		}
		offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
		if offset < 0 {
			offset = 0
		}

		var items []VideoItem
		var err error
		if source != "" {
			items, err = listVideosForSource(conf, source, refresh)
			if err != nil {
				WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
		} else {
			for _, src := range conf.Sources {
				srcItems, e := listVideosForSource(conf, src.Name, refresh)
				if e != nil {
					// 单个源失败不影响其它源（例如路径不存在）
					LogWarn("构建媒体列表失败，已跳过", "source", src.Name, "error", e)
					continue
				}
				items = append(items, srcItems...)
			}
		}

		// search filter
		if q != "" {
			qLower := strings.ToLower(q)
			filtered := make([]VideoItem, 0, len(items))
			for _, it := range items {
				if strings.Contains(strings.ToLower(it.Name), qLower) || strings.Contains(strings.ToLower(it.RelPath), qLower) {
					filtered = append(filtered, it)
				}
			}
			items = filtered
		}

		// stable sort: source -> relPath
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Source != items[j].Source {
				return items[i].Source < items[j].Source
			}
			return items[i].RelPath < items[j].RelPath
		})

		total := len(items)
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}

		resp := VideosResponse{
			GeneratedAt: time.Now().Unix(),
			Total:       total,
			Items:       items[offset:end],
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func listVideosForSource(conf *Config, sourceName string, refresh bool) ([]VideoItem, error) {
	srcPath, ok := conf.GetSourcePath(sourceName)
	if !ok {
		return nil, errors.New("unknown source: " + sourceName)
	}

	// 强制刷新时删除缓存
	if refresh {
		videosCache.Delete(sourceName)
	}

	return videosCache.Get(sourceName, func() ([]MediaItem, error) {
		return buildVideosForPath(conf, sourceName, srcPath)
	})
}

// buildVideosForPath 实际构建视频列表（无缓存）
func buildVideosForPath(conf *Config, sourceName, srcPath string) ([]MediaItem, error) {

	items := make([]MediaItem, 0, 1024)
	err := filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // 跳过不可访问路径
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !conf.IsMediaFile(name) {
			return nil
		}

		// 判断媒体类型
		mediaType := "video"
		if conf.IsAudioFile(name) {
			mediaType = "audio"
		}

		info, e := d.Info()
		if e != nil {
			return nil
		}
		rel, e := filepath.Rel(srcPath, path)
		if e != nil {
			return nil
		}

		relSlash := filepath.ToSlash(rel)
		streamURL := buildStreamURL(sourceName, relSlash)

		items = append(items, MediaItem{
			Source:    sourceName,
			Name:      name,
			RelPath:   relSlash,
			URL:       streamURL,
			Size:      info.Size(),
			ModTime:   info.ModTime().Unix(),
			MediaType: mediaType,
		})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		LogWarn("WalkDir 遇到错误", "source", sourceName, "path", srcPath, "error", err)
	}

	return items, nil
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
