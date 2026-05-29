package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Version 由构建时 -ldflags 注入；本地开发默认为 dev。
var Version = "dev"

const (
	githubRepoOwner  = "Seacolour"
	githubRepoName   = "YSeren"
	githubReleaseAPI = "https://api.github.com/repos/Seacolour/YSeren/releases/latest"
	releasePageURL   = "https://github.com/Seacolour/YSeren/releases"
)

type UpdateInfo struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
	Name    string `json:"name,omitempty"`
}

type VersionResponse struct {
	Version    string      `json:"version"`
	ReleaseURL string      `json:"releaseUrl"`
	Update     *UpdateInfo `json:"update,omitempty"`
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

var latestReleaseCache struct {
	sync.RWMutex
	at      time.Time
	release ghRelease
	err     error
}

const releaseCacheTTL = 6 * time.Hour

func VersionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp := VersionResponse{
			Version:    normalizeVersion(Version),
			ReleaseURL: releasePageURL,
		}

		if isComparableVersion(resp.Version) {
			if latest, ok := fetchLatestRelease(); ok && isNewerVersion(latest.version, resp.Version) {
				resp.Update = &UpdateInfo{
					Version: latest.version,
					Tag:     latest.tag,
					URL:     latest.url,
					Name:    latest.name,
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func isComparableVersion(v string) bool {
	v = normalizeVersion(v)
	if v == "" || v == "dev" {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

type parsedRelease struct {
	version string
	tag     string
	url     string
	name    string
}

func fetchLatestRelease() (parsedRelease, bool) {
	latestReleaseCache.RLock()
	if time.Since(latestReleaseCache.at) <= releaseCacheTTL && latestReleaseCache.err == nil {
		release := latestReleaseCache.release
		latestReleaseCache.RUnlock()
		return parseGHRelease(release)
	}
	if time.Since(latestReleaseCache.at) <= releaseCacheTTL && latestReleaseCache.err != nil {
		latestReleaseCache.RUnlock()
		return parsedRelease{}, false
	}
	latestReleaseCache.RUnlock()

	latestReleaseCache.Lock()
	defer latestReleaseCache.Unlock()

	if time.Since(latestReleaseCache.at) <= releaseCacheTTL {
		if latestReleaseCache.err != nil {
			return parsedRelease{}, false
		}
		return parseGHRelease(latestReleaseCache.release)
	}

	req, err := http.NewRequest(http.MethodGet, githubReleaseAPI, nil)
	if err != nil {
		latestReleaseCache.at = time.Now()
		latestReleaseCache.err = err
		return parsedRelease{}, false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("YSeren/%s", normalizeVersion(Version)))

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		latestReleaseCache.at = time.Now()
		latestReleaseCache.err = err
		LogWarn("检查 GitHub Release 失败", "error", err)
		return parsedRelease{}, false
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		latestReleaseCache.at = time.Now()
		latestReleaseCache.err = fmt.Errorf("github api status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
		LogWarn("检查 GitHub Release 失败", "status", res.StatusCode)
		return parsedRelease{}, false
	}

	var release ghRelease
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		latestReleaseCache.at = time.Now()
		latestReleaseCache.err = err
		LogWarn("解析 GitHub Release 失败", "error", err)
		return parsedRelease{}, false
	}

	latestReleaseCache.at = time.Now()
	latestReleaseCache.release = release
	latestReleaseCache.err = nil
	return parseGHRelease(release)
}

func parseGHRelease(release ghRelease) (parsedRelease, bool) {
	tag := strings.TrimSpace(release.TagName)
	version := normalizeVersion(tag)
	if version == "" {
		return parsedRelease{}, false
	}
	url := strings.TrimSpace(release.HTMLURL)
	if url == "" {
		url = releasePageURL + "/tag/" + tag
	}
	return parsedRelease{
		version: version,
		tag:     tag,
		url:     url,
		name:    strings.TrimSpace(release.Name),
	}, true
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func isNewerVersion(latest, current string) bool {
	return compareVersion(latest, current) > 0
}

func compareVersion(a, b string) int {
	ap := parseVersionParts(a)
	bp := parseVersionParts(b)
	if ap.ok && bp.ok {
		if ap.major != bp.major {
			return cmpInt(ap.major, bp.major)
		}
		if ap.minor != bp.minor {
			return cmpInt(ap.minor, bp.minor)
		}
		return cmpInt(ap.patch, bp.patch)
	}
	return strings.Compare(normalizeVersion(a), normalizeVersion(b))
}

type versionParts struct {
	major, minor, patch int
	ok                  bool
}

func parseVersionParts(v string) versionParts {
	v = normalizeVersion(v)
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return versionParts{}
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return versionParts{}
	}
	return versionParts{major: major, minor: minor, patch: patch, ok: true}
}

func cmpInt(a, b int) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}
