package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type VideoItem struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	RelPath string `json:"relPath"`
	URL     string `json:"url"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"` // unix seconds
}

type VideosResponse struct {
	GeneratedAt int64       `json:"generatedAt"`
	Total       int         `json:"total"`
	Items       []VideoItem `json:"items"`
}

type cachedVideos struct {
	at    time.Time
	items []VideoItem
}

var videosCache sync.Map // key: sourceName, value: cachedVideos

func ListVideosHandler(conf *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			for _, src := range conf.Sources {
				srcItems, e := listVideosForSource(conf, src.Name, refresh)
				if e != nil {
					// 单个源失败不影响其它源（例如路径不存在）
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
	var srcPath string
	for _, s := range conf.Sources {
		if s.Name == sourceName {
			srcPath = s.Path
			break
		}
	}
	if srcPath == "" {
		return nil, errors.New("unknown source: " + sourceName)
	}

	// cache: 5s TTL (可按需调大)
	const ttl = 5 * time.Second
	if !refresh {
		if v, ok := videosCache.Load(sourceName); ok {
			c := v.(cachedVideos)
			if time.Since(c.at) <= ttl {
				return c.items, nil
			}
		}
	}

	items := make([]VideoItem, 0, 1024)
	err := filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // 跳过不可访问路径
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !isVideo(name) {
			return nil
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
		encodedRel := encodeURLPath(relSlash)
		streamURL := "/stream/" + url.PathEscape(sourceName) + "/" + encodedRel

		items = append(items, VideoItem{
			Source:  sourceName,
			Name:    name,
			RelPath: relSlash,
			URL:     streamURL,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// WalkDir 只在回调返回 error 时才会透出；这里理论上很少触发
	}

	videosCache.Store(sourceName, cachedVideos{at: time.Now(), items: items})
	return items, nil
}

func isVideo(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".mkv", ".webm", ".mov", ".m4v", ".avi":
		return true
	default:
		return false
	}
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

// encodeURLPath：对每个 path segment 做 PathEscape，保留 “/” 作为层级分隔符
func encodeURLPath(relSlash string) string {
	relSlash = strings.TrimPrefix(relSlash, "/")
	parts := strings.Split(relSlash, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
