package server

import (
	"log/slog"
	"net/http"
	"time"

	webfrontend "yseren/frontend"
	coreconfig "yseren/internal/config"
	"yseren/internal/media"
	coreversion "yseren/internal/version"
)

const defaultCacheTTL = 5 * time.Second

type Options struct {
	FrontendHandler http.Handler
	VersionHandler  http.Handler
	Version         string
	Logger          *slog.Logger
	CacheTTL        time.Duration
}

// Application 是单个 YSeren HTTP 服务实例，持有独立路由和缓存。
type Application struct {
	config     coreconfig.Config
	logger     *slog.Logger
	indexCache *Cache[[]media.Entry]
	mux        *http.ServeMux
}

func New(conf *coreconfig.Config, options Options) *Application {
	cloned := conf.Clone()
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	cacheTTL := options.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	frontend := options.FrontendHandler
	if frontend == nil {
		frontend = webfrontend.Handler()
	}

	app := &Application{
		config:     cloned,
		logger:     logger,
		indexCache: NewCache[[]media.Entry](cacheTTL),
		mux:        http.NewServeMux(),
	}
	app.mux.Handle(StreamRoutePrefix, app.StreamHandler())
	app.mux.HandleFunc("/api/videos", app.VideosHandler())
	app.mux.HandleFunc("/api/tree", app.TreeHandler())
	versionHandler := options.VersionHandler
	if versionHandler == nil {
		versionHandler = coreversion.New(options.Version, logger).Handler()
	}
	app.mux.Handle("/api/version", versionHandler)
	app.mux.Handle("/", frontend)
	return app
}

func (a *Application) Handler() http.Handler {
	return a.mux
}

func (a *Application) StreamHandler() http.Handler {
	return newStreamHandler(&a.config)
}
