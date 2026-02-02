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
	"strings"
	"time"
)

type TreeNode struct {
	Type      string      `json:"type"` // "dir" | "file" | "zip"
	Name      string      `json:"name"`
	RelPath   string      `json:"relPath"`
	Source    string      `json:"source,omitempty"`
	URL       string      `json:"url,omitempty"`
	Size      int64       `json:"size,omitempty"`
	ModTime   int64       `json:"modTime,omitempty"`
	MediaType string      `json:"mediaType,omitempty"` // "video" | "audio" (only for file type)
	Children  []*TreeNode `json:"children,omitempty"`
}

type TreeResponse struct {
	GeneratedAt int64     `json:"generatedAt"`
	Source      string    `json:"source,omitempty"`
	Root        *TreeNode `json:"root"`
}

// treeCache 用于缓存目录树，5秒 TTL，使用 singleflight 防止缓存击穿
var treeCache = NewCache[*TreeNode](5 * time.Second)

// GET /api/tree?source=xxx&q=xxx&refresh=1
// - source 为空：返回一个虚拟 root，children 是每个 source 的 root 目录
// - q 为可选搜索词（匹配 name/relPath），会“裁剪树”仅保留命中的路径与祖先
func ListTreeHandler(conf *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		source := strings.TrimSpace(r.URL.Query().Get("source"))
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		refresh := r.URL.Query().Get("refresh") == "1"

		var root *TreeNode
		if source != "" {
			srcRoot, err := buildBrowsableTreeForSource(conf, source, refresh)
			if err != nil {
				WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			root = srcRoot
			if q != "" {
				root = filterTree(root, strings.ToLower(q))
				if root == nil {
					root = &TreeNode{Type: "dir", Name: source, RelPath: "", Source: source}
				}
			}
		} else {
			root = &TreeNode{Type: "dir", Name: "root", RelPath: ""}
			for _, src := range conf.Sources {
				srcRoot, err := buildBrowsableTreeForSource(conf, src.Name, refresh)
				if err != nil {
					// 乐观策略：单个源失败不影响整体；但记录日志方便排障
					LogWarn("构建目录树失败，已跳过", "source", src.Name, "error", err)
					continue
				}
				if q != "" {
					srcRoot = filterTree(srcRoot, strings.ToLower(q))
					if srcRoot == nil {
						continue
					}
				}
				root.Children = append(root.Children, srcRoot)
			}
			sortTree(root)
		}

		resp := TreeResponse{
			GeneratedAt: time.Now().Unix(),
			Source:      source,
			Root:        root,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func buildBrowsableTreeForSource(conf *Config, sourceName string, refresh bool) (*TreeNode, error) {
	srcPath, ok := conf.GetSourcePath(sourceName)
	if !ok {
		return nil, errors.New("unknown source: " + sourceName)
	}

	// 强制刷新时删除缓存
	if refresh {
		treeCache.Delete(sourceName)
	}

	return treeCache.Get(sourceName, func() (*TreeNode, error) {
		return buildTreeForPath(conf, sourceName, srcPath)
	})
}

// buildTreeForPath 实际构建目录树（无缓存）
func buildTreeForPath(conf *Config, sourceName, srcPath string) (*TreeNode, error) {

	root := &TreeNode{Type: "dir", Name: sourceName, RelPath: "", Source: sourceName}
	dirIndex := map[string]*TreeNode{"": root}

	ensureDir := func(rel string) *TreeNode {
		rel = strings.Trim(rel, "/")
		if rel == "" {
			return root
		}
		if n, ok := dirIndex[rel]; ok {
			return n
		}
		parts := strings.Split(rel, "/")
		cur := ""
		parent := root
		for _, seg := range parts {
			if seg == "" {
				continue
			}
			next := seg
			if cur != "" {
				next = cur + "/" + seg
			}
			if existing, ok := dirIndex[next]; ok {
				parent = existing
				cur = next
				continue
			}
			node := &TreeNode{Type: "dir", Name: seg, RelPath: next, Source: sourceName}
			parent.Children = append(parent.Children, node)
			dirIndex[next] = node
			parent = node
			cur = next
		}
		return parent
	}

	_ = filepath.WalkDir(srcPath, func(abs string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(srcPath, abs)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)

		ext := strings.ToLower(filepath.Ext(relSlash))
		if ext != ".zip" && !conf.IsMediaFile(relSlash) {
			return nil
		}

		parentRel := filepath.ToSlash(filepath.Dir(rel))
		if parentRel == "." {
			parentRel = ""
		}
		parent := ensureDir(parentRel)

		name := d.Name()
		if ext == ".zip" {
			parent.Children = append(parent.Children, &TreeNode{
				Type:    "zip",
				Name:    name,
				RelPath: relSlash,
				Source:  sourceName,
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
			})
			return nil
		}

		encodedRel := encodeURLPath(relSlash)
		streamURL := "/stream/" + url.PathEscape(sourceName) + "/" + encodedRel
		mediaType := "video"
		if IsAudioFile(name) {
			mediaType = "audio"
		}
		parent.Children = append(parent.Children, &TreeNode{
			Type:      "file",
			Name:      name,
			RelPath:   relSlash,
			Source:    sourceName,
			URL:       streamURL,
			Size:      info.Size(),
			ModTime:   info.ModTime().Unix(),
			MediaType: mediaType,
		})
		return nil
	})

	// 如果 source 路径不存在，也返回空树（而不是直接报错）
	if _, err := os.Stat(srcPath); err != nil && errors.Is(err, os.ErrNotExist) {
		// ignore
	}

	sortTree(root)
	return root, nil
}

func sortTree(n *TreeNode) {
	if n == nil || len(n.Children) == 0 {
		return
	}
	for _, c := range n.Children {
		sortTree(c)
	}
	sort.SliceStable(n.Children, func(i, j int) bool {
		// dir 先于 zip 先于 file
		if n.Children[i].Type != n.Children[j].Type {
			rank := func(t string) int {
				switch t {
				case "dir":
					return 0
				case "zip":
					return 1
				default: // file
					return 2
				}
			}
			return rank(n.Children[i].Type) < rank(n.Children[j].Type)
		}
		return n.Children[i].Name < n.Children[j].Name
	})
}

// filterTree：返回裁剪后的新树（不修改原树），仅保留命中节点与其祖先
func filterTree(n *TreeNode, qLower string) *TreeNode {
	if n == nil {
		return nil
	}
	matched := strings.Contains(strings.ToLower(n.Name), qLower) || strings.Contains(strings.ToLower(n.RelPath), qLower)
	if len(n.Children) == 0 {
		if matched {
			cp := *n
			cp.Children = nil
			return &cp
		}
		return nil
	}

	var kept []*TreeNode
	for _, c := range n.Children {
		if filtered := filterTree(c, qLower); filtered != nil {
			kept = append(kept, filtered)
		}
	}
	if matched || len(kept) > 0 {
		cp := *n
		cp.Children = kept
		return &cp
	}
	return nil
}
