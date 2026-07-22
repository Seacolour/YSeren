package version

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	githubReleaseAPI = "https://api.github.com/repos/Seacolour/YSeren/releases/latest"
	ReleasePageURL   = "https://github.com/Seacolour/YSeren/releases"
	releaseCacheTTL  = 6 * time.Hour
)

type UpdateInfo struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
	Name    string `json:"name,omitempty"`
}

type Response struct {
	Version    string      `json:"version"`
	ReleaseURL string      `json:"releaseUrl"`
	Update     *UpdateInfo `json:"update,omitempty"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

type parsedRelease struct {
	version string
	tag     string
	url     string
	name    string
}

// Checker 持有单个服务实例的版本检查缓存。
type Checker struct {
	current string
	logger  *slog.Logger
	client  *http.Client

	mu      sync.Mutex
	at      time.Time
	release githubRelease
	err     error
}

func New(current string, logger *slog.Logger) *Checker {
	current = Normalize(current)
	if current == "" {
		current = "dev"
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Checker{
		current: current,
		logger:  logger,
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}

		response := Response{Version: c.current, ReleaseURL: ReleasePageURL}
		if IsComparable(c.current) {
			if latest, ok := c.fetchLatestRelease(); ok && IsNewer(latest.version, c.current) {
				response.Update = &UpdateInfo{
					Version: latest.version,
					Tag:     latest.tag,
					URL:     latest.url,
					Name:    latest.name,
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response)
	}
}

func (c *Checker) fetchLatestRelease() (parsedRelease, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.at) <= releaseCacheTTL {
		if c.err != nil {
			return parsedRelease{}, false
		}
		return parseGithubRelease(c.release)
	}

	request, err := http.NewRequest(http.MethodGet, githubReleaseAPI, nil)
	if err != nil {
		c.storeError(err)
		return parsedRelease{}, false
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", fmt.Sprintf("YSeren/%s", c.current))

	response, err := c.client.Do(request)
	if err != nil {
		c.storeError(err)
		c.logger.Warn("检查 GitHub Release 失败", "error", err)
		return parsedRelease{}, false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		err := fmt.Errorf("github api status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
		c.storeError(err)
		c.logger.Warn("检查 GitHub Release 失败", "status", response.StatusCode)
		return parsedRelease{}, false
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		c.storeError(err)
		c.logger.Warn("解析 GitHub Release 失败", "error", err)
		return parsedRelease{}, false
	}
	c.at = time.Now()
	c.release = release
	c.err = nil
	return parseGithubRelease(release)
}

func (c *Checker) storeError(err error) {
	c.at = time.Now()
	c.err = err
}

func parseGithubRelease(release githubRelease) (parsedRelease, bool) {
	tag := strings.TrimSpace(release.TagName)
	version := Normalize(tag)
	if version == "" {
		return parsedRelease{}, false
	}
	url := strings.TrimSpace(release.HTMLURL)
	if url == "" {
		url = ReleasePageURL + "/tag/" + tag
	}
	return parsedRelease{
		version: version,
		tag:     tag,
		url:     url,
		name:    strings.TrimSpace(release.Name),
	}, true
}

func Normalize(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func IsComparable(value string) bool {
	value = Normalize(value)
	if value == "" || value == "dev" {
		return false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func IsNewer(latest, current string) bool {
	return Compare(latest, current) > 0
}

func Compare(first, second string) int {
	firstParts := parseVersionParts(first)
	secondParts := parseVersionParts(second)
	if firstParts.ok && secondParts.ok {
		if firstParts.major != secondParts.major {
			return compareInt(firstParts.major, secondParts.major)
		}
		if firstParts.minor != secondParts.minor {
			return compareInt(firstParts.minor, secondParts.minor)
		}
		return compareInt(firstParts.patch, secondParts.patch)
	}
	return strings.Compare(Normalize(first), Normalize(second))
}

type versionParts struct {
	major int
	minor int
	patch int
	ok    bool
}

func parseVersionParts(value string) versionParts {
	parts := strings.Split(Normalize(value), ".")
	if len(parts) != 3 {
		return versionParts{}
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	patch, patchErr := strconv.Atoi(parts[2])
	if majorErr != nil || minorErr != nil || patchErr != nil {
		return versionParts{}
	}
	return versionParts{major: major, minor: minor, patch: patch, ok: true}
}

func compareInt(first, second int) int {
	if first > second {
		return 1
	}
	if first < second {
		return -1
	}
	return 0
}
