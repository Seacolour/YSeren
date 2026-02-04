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

func ErrorFrontendHandler(message string) http.Handler {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "启动失败：未知错误"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>YSeren 启动失败</title><style>body{font-family:-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;background:#f6f7fb;color:#1a1a1a;margin:0;padding:40px;} .card{max-width:720px;margin:0 auto;background:#fff;border-radius:16px;padding:28px 32px;box-shadow:0 8px 30px rgba(0,0,0,.08);} h1{font-size:22px;margin:0 0 12px;} p{margin:0 0 12px;line-height:1.6;} pre{background:#f1f3f7;border-radius:10px;padding:12px;white-space:pre-wrap;word-break:break-word;}</style></head><body><div class=\"card\"><h1>YSeren 启动失败</h1><p>应用未能正常启动，请检查配置文件或目录权限。</p><pre>"))
		_, _ = w.Write([]byte(message))
		_, _ = w.Write([]byte("</pre><p>建议：在当前目录或 exe 同目录放置 v-link.yaml，或使用 -config 指定路径。</p></div></body></html>"))
	})
}
