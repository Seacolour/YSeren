package frontend

import (
	"bytes"
	"embed"
	"html"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed dist/index.html dist/assets/*
var embeddedDist embed.FS

// Handler 提供嵌入式 Web Player，并在开发构建中回退到磁盘目录。
func Handler() http.Handler {
	if handler, ok := embeddedHandler(); ok {
		return handler
	}
	return diskHandler("frontend/dist")
}

func embeddedHandler() (http.Handler, bool) {
	dist, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return nil, false
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, false
	}
	return spaFileServer(http.FileServer(http.FS(dist)), index), true
}

func diskHandler(distDir string) http.Handler {
	index, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("frontend/dist 未构建：请先在 frontend 里执行构建，或使用 embed 打包。\n"))
		})
	}
	return spaFileServer(http.FileServer(http.Dir(distDir)), index)
}

func spaFileServer(next http.Handler, index []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/stream/") {
			http.NotFound(w, r)
			return
		}

		recorder := newBufferedResponseWriter()
		next.ServeHTTP(recorder, r)
		if recorder.statusCode() != http.StatusNotFound {
			recorder.flushTo(w)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
	})
}

type bufferedResponseWriter struct {
	header http.Header
	status int
	buffer bytes.Buffer
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header)}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(code int) {
	w.status = code
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.buffer.Write(data)
}

func (w *bufferedResponseWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *bufferedResponseWriter) flushTo(target http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			target.Header().Add(key, value)
		}
	}
	target.WriteHeader(w.statusCode())
	_, _ = target.Write(w.buffer.Bytes())
}

// ErrorHandler 提供配置加载失败时的本地错误页。
func ErrorHandler(message string) http.Handler {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "启动失败：未知错误"
	}
	escapedMessage := html.EscapeString(message)
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>YSeren 启动失败</title><style>body{font-family:-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;background:#f6f7fb;color:#1a1a1a;margin:0;padding:40px;} .card{max-width:720px;margin:0 auto;background:#fff;border-radius:16px;padding:28px 32px;box-shadow:0 8px 30px rgba(0,0,0,.08);} h1{font-size:22px;margin:0 0 12px;} p{margin:0 0 12px;line-height:1.6;} pre{background:#f1f3f7;border-radius:10px;padding:12px;white-space:pre-wrap;word-break:break-word;}</style></head><body><div class=\"card\"><h1>YSeren 启动失败</h1><p>应用未能正常启动，请检查配置文件或目录权限。</p><pre>"))
		_, _ = w.Write([]byte(escapedMessage))
		_, _ = w.Write([]byte("</pre><p>建议：在当前目录或 exe 同目录放置 yseren.yaml，或使用 -config 指定路径。</p></div></body></html>"))
	})
}
