package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"yseren/internal/media"
)

func (a *Application) PlaylistHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		baseURL := "http://" + strings.TrimSpace(r.Host)
		if strings.TrimSpace(r.Host) == "" {
			baseURL = fmt.Sprintf("http://localhost:%d", a.config.Server.Port)
		}

		var body strings.Builder
		body.WriteString("#EXTM3U\n")
		for _, source := range a.config.Sources {
			entries, err := a.entriesForSource(source.Name, false)
			if err != nil {
				a.logger.Warn("构建播放列表失败，已跳过", "source", source.Name, "error", err)
				continue
			}
			entries = append([]media.Entry(nil), entries...)
			sort.SliceStable(entries, func(i, j int) bool {
				return entries[i].RelPath < entries[j].RelPath
			})
			for _, entry := range entries {
				body.WriteString("#EXTINF:-1,")
				body.WriteString(sanitizePlaylistLabel(entry.Name))
				body.WriteByte('\n')
				body.WriteString(baseURL)
				body.WriteString(BuildStreamURL(source.Name, entry.RelPath))
				body.WriteByte('\n')
			}
		}

		content := body.String()
		w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte(content))
	}
}

func sanitizePlaylistLabel(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}
