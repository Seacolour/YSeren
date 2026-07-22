package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"yseren/internal/media"
)

type MediaItem struct {
	Source    string `json:"source"`
	Name      string `json:"name"`
	RelPath   string `json:"relPath"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	ModTime   int64  `json:"modTime"`
	MediaType string `json:"mediaType"`
}

type VideoItem = MediaItem

type VideosResponse struct {
	GeneratedAt int64       `json:"generatedAt"`
	Total       int         `json:"total"`
	Items       []VideoItem `json:"items"`
}

type TreeNode struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	RelPath   string      `json:"relPath"`
	Source    string      `json:"source,omitempty"`
	URL       string      `json:"url,omitempty"`
	Size      int64       `json:"size,omitempty"`
	ModTime   int64       `json:"modTime,omitempty"`
	MediaType string      `json:"mediaType,omitempty"`
	Children  []*TreeNode `json:"children,omitempty"`
}

type TreeResponse struct {
	GeneratedAt int64     `json:"generatedAt"`
	Source      string    `json:"source,omitempty"`
	Root        *TreeNode `json:"root"`
}

func (a *Application) TreeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		source := strings.TrimSpace(r.URL.Query().Get("source"))
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		refresh := r.URL.Query().Get("refresh") == "1"

		var root *TreeNode
		if source != "" {
			entries, err := a.entriesForSource(source, refresh)
			if err != nil {
				WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			root = buildTree(source, entries)
			if query != "" {
				root = filterTree(root, strings.ToLower(query))
				if root == nil {
					root = &TreeNode{Type: "dir", Name: source, Source: source}
				}
			}
		} else {
			root = &TreeNode{Type: "dir", Name: "root"}
			for _, configuredSource := range a.config.Sources {
				entries, err := a.entriesForSource(configuredSource.Name, refresh)
				if err != nil {
					a.logger.Warn("构建目录树失败，已跳过", "source", configuredSource.Name, "error", err)
					continue
				}
				sourceRoot := buildTree(configuredSource.Name, entries)
				if query != "" {
					sourceRoot = filterTree(sourceRoot, strings.ToLower(query))
					if sourceRoot == nil {
						continue
					}
				}
				root.Children = append(root.Children, sourceRoot)
			}
			sortTree(root)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(TreeResponse{
			GeneratedAt: time.Now().Unix(),
			Source:      source,
			Root:        root,
		})
	}
}

func (a *Application) VideosHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
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
		if source != "" {
			entries, err := a.entriesForSource(source, refresh)
			if err != nil {
				WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			items = mediaItems(source, entries)
		} else {
			for _, configuredSource := range a.config.Sources {
				entries, err := a.entriesForSource(configuredSource.Name, refresh)
				if err != nil {
					a.logger.Warn("构建媒体列表失败，已跳过", "source", configuredSource.Name, "error", err)
					continue
				}
				items = append(items, mediaItems(configuredSource.Name, entries)...)
			}
		}

		if query != "" {
			queryLower := strings.ToLower(query)
			filtered := make([]VideoItem, 0, len(items))
			for _, item := range items {
				if strings.Contains(strings.ToLower(item.Name), queryLower) || strings.Contains(strings.ToLower(item.RelPath), queryLower) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(VideosResponse{
			GeneratedAt: time.Now().Unix(),
			Total:       total,
			Items:       items[offset:end],
		})
	}
}

func (a *Application) entriesForSource(sourceName string, refresh bool) ([]media.Entry, error) {
	sourcePath, ok := a.config.GetSourcePath(sourceName)
	if !ok {
		return nil, errors.New("unknown source: " + sourceName)
	}
	if refresh {
		a.indexCache.Delete(sourceName)
	}
	return a.indexCache.Get(sourceName, func() ([]media.Entry, error) {
		return media.Scan(sourcePath, &a.config)
	})
}

func mediaItems(sourceName string, entries []media.Entry) []VideoItem {
	items := make([]VideoItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, VideoItem{
			Source:    sourceName,
			Name:      entry.Name,
			RelPath:   entry.RelPath,
			URL:       BuildStreamURL(sourceName, entry.RelPath),
			Size:      entry.Size,
			ModTime:   entry.ModTime,
			MediaType: entry.MediaType,
		})
	}
	return items
}

func buildTree(sourceName string, entries []media.Entry) *TreeNode {
	root := &TreeNode{Type: "dir", Name: sourceName, Source: sourceName}
	directories := map[string]*TreeNode{"": root}

	ensureDirectory := func(relPath string) *TreeNode {
		relPath = strings.Trim(relPath, "/")
		if relPath == "" {
			return root
		}
		if existing, ok := directories[relPath]; ok {
			return existing
		}
		parts := strings.Split(relPath, "/")
		current := ""
		parent := root
		for _, part := range parts {
			if part == "" {
				continue
			}
			next := part
			if current != "" {
				next = current + "/" + part
			}
			if existing, ok := directories[next]; ok {
				parent = existing
				current = next
				continue
			}
			node := &TreeNode{Type: "dir", Name: part, RelPath: next, Source: sourceName}
			parent.Children = append(parent.Children, node)
			directories[next] = node
			parent = node
			current = next
		}
		return parent
	}

	for _, entry := range entries {
		parentPath := ""
		if separator := strings.LastIndex(entry.RelPath, "/"); separator >= 0 {
			parentPath = entry.RelPath[:separator]
		}
		parent := ensureDirectory(parentPath)
		parent.Children = append(parent.Children, &TreeNode{
			Type:      "file",
			Name:      entry.Name,
			RelPath:   entry.RelPath,
			Source:    sourceName,
			URL:       BuildStreamURL(sourceName, entry.RelPath),
			Size:      entry.Size,
			ModTime:   entry.ModTime,
			MediaType: entry.MediaType,
		})
	}
	sortTree(root)
	return root
}

func sortTree(node *TreeNode) {
	if node == nil || len(node.Children) == 0 {
		return
	}
	for _, child := range node.Children {
		sortTree(child)
	}
	sort.SliceStable(node.Children, func(i, j int) bool {
		if node.Children[i].Type != node.Children[j].Type {
			return node.Children[i].Type == "dir"
		}
		return node.Children[i].Name < node.Children[j].Name
	})
}

func filterTree(node *TreeNode, queryLower string) *TreeNode {
	if node == nil {
		return nil
	}
	matched := strings.Contains(strings.ToLower(node.Name), queryLower) || strings.Contains(strings.ToLower(node.RelPath), queryLower)
	if len(node.Children) == 0 {
		if !matched {
			return nil
		}
		clone := *node
		clone.Children = nil
		return &clone
	}

	children := make([]*TreeNode, 0, len(node.Children))
	for _, child := range node.Children {
		if filtered := filterTree(child, queryLower); filtered != nil {
			children = append(children, filtered)
		}
	}
	if !matched && len(children) == 0 {
		return nil
	}
	clone := *node
	clone.Children = children
	return &clone
}

func parseIntDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
