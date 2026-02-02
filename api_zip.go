package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ZipExtractRequest struct {
	Source  string `json:"source"`
	RelPath string `json:"relPath"` // zip 文件相对 source 根目录的路径（使用 / 分隔）
}

type ZipExtractResponse struct {
	Status          string `json:"status"` // "ok" | "exists" | "error"
	Message         string `json:"message,omitempty"`
	ExtractedRelDir string `json:"extractedRelDir,omitempty"` // 解压后的相对目录
}

// POST /api/zip/extract
// body: { "source": "...", "relPath": "Anime/a.zip" }
// 行为：把 zip 解压到 zip 同目录下的同名文件夹（去掉 .zip）
func ZipExtractHandler(conf *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req ZipExtractRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "bad json")
			return
		}
		req.Source = strings.TrimSpace(req.Source)
		req.RelPath = strings.Trim(strings.ReplaceAll(req.RelPath, "\\", "/"), "/")
		if req.Source == "" || req.RelPath == "" {
			WriteError(w, http.StatusBadRequest, "missing source/relPath")
			return
		}
		if !strings.HasSuffix(strings.ToLower(req.RelPath), ".zip") {
			WriteError(w, http.StatusBadRequest, "not a .zip file")
			return
		}

		absZip, absDst, dstRel, err := resolveZipPaths(conf, req.Source, req.RelPath)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if _, err := os.Stat(absDst); err == nil {
			// 已存在就不覆盖，交给用户自行处理
			writeJSON(w, ZipExtractResponse{
				Status:          "exists",
				Message:         "目标目录已存在，未执行解压",
				ExtractedRelDir: dstRel,
			})
			return
		}

		if err := unzipTo(absZip, absDst); err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 解压后，让 /api/tree 的缓存尽快失效
		treeCache.Delete(req.Source)
		videosCache.Delete(req.Source)

		writeJSON(w, ZipExtractResponse{
			Status:          "ok",
			Message:         "解压完成",
			ExtractedRelDir: dstRel,
		})
	}
}

func resolveZipPaths(conf *Config, sourceName, relSlash string) (absZip string, absDst string, dstRel string, err error) {
	srcPath, ok := conf.GetSourcePath(sourceName)
	if !ok {
		return "", "", "", errors.New("unknown source: " + sourceName)
	}

	relOS := filepath.FromSlash(relSlash)
	absZip = filepath.Join(srcPath, relOS)

	// 安全校验：确保 absZip 在 srcPath 内（防路径穿越）
	if !IsPathSafe(srcPath, absZip) {
		return "", "", "", errors.New("invalid path")
	}

	if _, err := os.Stat(absZip); err != nil {
		return "", "", "", err
	}

	// 目标目录：同目录、同名去掉 .zip
	base := strings.TrimSuffix(filepath.Base(absZip), filepath.Ext(absZip))
	parent := filepath.Dir(absZip)
	absDst = filepath.Join(parent, base)

	// 返回给前端的相对目录（/ 分隔）
	relDstOS, _ := filepath.Rel(srcPath, absDst)
	dstRel = filepath.ToSlash(relDstOS)
	if dstRel == "." {
		dstRel = ""
	}
	return absZip, absDst, dstRel, nil
}

func unzipTo(zipPath, dstDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		name := filepath.FromSlash(f.Name)
		if name == "" || name == "." {
			continue
		}
		// ZipSlip 防护：目标路径必须在 dstDir 内
		target := filepath.Join(dstDir, name)
		if !IsPathSafe(dstDir, target) {
			return errors.New("zip contains invalid path")
		}

		// 跳过潜在符号链接（MVP：只解普通文件/目录）
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if err != nil {
			in.Close()
			return err
		}
		_, cpErr := io.Copy(out, in)
		_ = out.Close()
		_ = in.Close()
		if cpErr != nil {
			return cpErr
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
