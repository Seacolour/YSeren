package main

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// 注意：Go 的 embed 模式不支持 **，Vite 默认只会生成 dist/assets 一层目录，这里显式包含即可。
//
//go:embed frontend/dist/index.html frontend/dist/assets/*
var embeddedDist embed.FS

// FrontendHandler:
// - 优先使用 embed 的 dist（单文件交付）
// - 如果 dist 为空/未构建，则回退到磁盘上的 frontend/dist（方便开发）
// - 对于 SPA：找不到静态文件时回退 index.html
func FrontendHandler() http.Handler {
	// 1) 尝试用 embed
	if h, ok := tryEmbeddedFrontend(); ok {
		return h
	}
	// 2) 回退到磁盘
	return diskFrontendHandler("frontend/dist")
}

func tryEmbeddedFrontend() (http.Handler, bool) {
	sub, err := fs.Sub(embeddedDist, "frontend/dist")
	if err != nil {
		return nil, false
	}
	// 判断 index.html 是否存在，避免 embed 目录为空时“假成功”
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return spaFileServer(http.FileServer(http.FS(sub))), true
}

func diskFrontendHandler(distDir string) http.Handler {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		// dist 不存在时，至少给一个可理解的提示
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("frontend/dist 未构建：请先在 frontend 里执行构建，或使用 embed 打包。\n"))
		})
	}
	return spaFileServer(http.FileServer(http.Dir(distDir)))
}

func spaFileServer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API/流媒体路由不由这里接管（防止 SPA fallback 误吞掉）
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/stream/") {
			http.NotFound(w, r)
			return
		}

		// 先尝试静态资源
		rec := newBufferedResponseWriter()
		next.ServeHTTP(rec, r)
		if rec.statusCode() != http.StatusNotFound {
			rec.flushTo(w)
			return
		}

		// fallback 到 index.html
		r2 := *r
		r2.URL = newCopyURL(r.URL)
		r2.URL.Path = "/index.html"
		next.ServeHTTP(w, &r2)
	})
}

type bufferedResponseWriter struct {
	header http.Header
	status int
	buf    bytes.Buffer
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{
		header: make(http.Header),
	}
}

func (b *bufferedResponseWriter) Header() http.Header {
	return b.header
}

func (b *bufferedResponseWriter) WriteHeader(code int) {
	b.status = code
}

func (b *bufferedResponseWriter) Write(p []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	return b.buf.Write(p)
}

func (b *bufferedResponseWriter) statusCode() int {
	if b.status == 0 {
		return http.StatusOK
	}
	return b.status
}

func (b *bufferedResponseWriter) flushTo(w http.ResponseWriter) {
	// copy headers
	for k, vs := range b.header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(b.statusCode())
	_, _ = w.Write(b.buf.Bytes())
}

func newCopyURL(u *url.URL) *url.URL {
	u2 := *u
	return &u2
}
