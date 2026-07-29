package server

import (
	"encoding/json"
	"net/http"
	"strings"

	coreconfig "yseren/internal/config"
)

// StatusResponse is shared by Headless and Desktop HTTP servers. Filesystem
// paths are deliberately excluded because this endpoint is exposed to the LAN.
type StatusResponse struct {
	State    string   `json:"state"`
	Name     string   `json:"name"`
	Source   string   `json:"source,omitempty"`
	RootName string   `json:"rootName,omitempty"`
	Port     int      `json:"port"`
	URLs     []string `json:"urls"`
}

// NewStatusHandler exposes sanitized runtime state through the shared API.
func NewStatusHandler(provider func() StatusResponse) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		response := StatusResponse{State: "running", Name: "YSeren", URLs: []string{}}
		if provider != nil {
			response = provider()
		}
		if strings.TrimSpace(response.State) == "" {
			response.State = "running"
		}
		if strings.TrimSpace(response.Name) == "" {
			response.Name = "YSeren"
		}
		if response.URLs == nil {
			response.URLs = []string{}
		}
		if len(response.URLs) == 0 && strings.TrimSpace(r.Host) != "" {
			response.URLs = []string{"http://" + r.Host + "/"}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response)
	})
}

func newStaticStatusHandler(conf *coreconfig.Config) http.Handler {
	cloned := conf.Clone()
	return NewStatusHandler(func() StatusResponse {
		response := StatusResponse{
			State: "running",
			Name:  "YSeren",
			Port:  cloned.Server.Port,
			URLs:  []string{},
		}
		if len(cloned.Sources) == 1 {
			response.Source = cloned.Sources[0].Name
			response.RootName = cloned.Sources[0].Name
		}
		return response
	})
}
