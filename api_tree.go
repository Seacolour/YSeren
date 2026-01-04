package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

type TreeNode struct {
	Type     string      `json:"type"` // "dir" | "file"
	Name     string      `json:"name"`
	RelPath  string      `json:"relPath"`
	Source   string      `json:"source,omitempty"`
	URL      string      `json:"url,omitempty"`
	Size     int64       `json:"size,omitempty"`
	ModTime  int64       `json:"modTime,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}

type TreeResponse struct {
	GeneratedAt int64     `json:"generatedAt"`
	Source      string    `json:"source,omitempty"`
	Root        *TreeNode `json:"root"`
}

// GET /api/tree?source=xxx&q=xxx&refresh=1
// - source 为空：返回一个虚拟 root，children 是每个 source 的 root 目录
// - q 为可选搜索词（匹配 name/relPath），会“裁剪树”仅保留命中的路径与祖先
func ListTreeHandler(conf *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		source := strings.TrimSpace(r.URL.Query().Get("source"))
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		refresh := r.URL.Query().Get("refresh") == "1"

		var root *TreeNode
		if source != "" {
			items, err := listVideosForSource(conf, source, refresh)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			root = buildTreeForSource(source, items)
			if q != "" {
				root = filterTree(root, strings.ToLower(q))
				if root == nil {
					root = &TreeNode{Type: "dir", Name: source, RelPath: "", Source: source}
				}
			}
		} else {
			root = &TreeNode{Type: "dir", Name: "root", RelPath: ""}
			for _, src := range conf.Sources {
				items, err := listVideosForSource(conf, src.Name, refresh)
				if err != nil {
					continue
				}
				srcRoot := buildTreeForSource(src.Name, items)
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

func buildTreeForSource(source string, items []VideoItem) *TreeNode {
	root := &TreeNode{
		Type:    "dir",
		Name:    source,
		RelPath: "",
		Source:  source,
	}

	// 目录节点索引：relPath -> *TreeNode
	dirIndex := map[string]*TreeNode{
		"": root,
	}

	for _, it := range items {
		parts := strings.Split(it.RelPath, "/")
		// parts 最后一个是文件名
		curRel := ""
		for i := 0; i < len(parts)-1; i++ {
			seg := parts[i]
			if seg == "" {
				continue
			}
			nextRel := seg
			if curRel != "" {
				nextRel = curRel + "/" + seg
			}
			if _, ok := dirIndex[nextRel]; !ok {
				node := &TreeNode{
					Type:    "dir",
					Name:    seg,
					RelPath: nextRel,
					Source:  source,
				}
				dirIndex[curRel].Children = append(dirIndex[curRel].Children, node)
				dirIndex[nextRel] = node
			}
			curRel = nextRel
		}

		fileNode := &TreeNode{
			Type:    "file",
			Name:    it.Name,
			RelPath: it.RelPath,
			Source:  source,
			URL:     it.URL,
			Size:    it.Size,
			ModTime: it.ModTime,
		}
		dirIndex[curRel].Children = append(dirIndex[curRel].Children, fileNode)
	}

	sortTree(root)
	return root
}

func sortTree(n *TreeNode) {
	if n == nil || len(n.Children) == 0 {
		return
	}
	for _, c := range n.Children {
		sortTree(c)
	}
	sort.SliceStable(n.Children, func(i, j int) bool {
		// dir 先于 file
		if n.Children[i].Type != n.Children[j].Type {
			return n.Children[i].Type == "dir"
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
