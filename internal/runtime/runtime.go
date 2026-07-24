package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	coreconfig "yseren/internal/config"
	appserver "yseren/internal/server"
)

var ErrAlreadyRunning = errors.New("YSeren 服务已在运行")

type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateFailed   State = "failed"
)

type Status struct {
	State     State     `json:"state"`
	Address   string    `json:"address,omitempty"`
	Port      int       `json:"port,omitempty"`
	URLs      []string  `json:"urls,omitempty"`
	StartedAt time.Time `json:"startedAt,omitempty"`
	LastError string    `json:"lastError,omitempty"`
}

type Controller interface {
	Start(ctx context.Context, conf coreconfig.Config) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context, conf coreconfig.Config) error
	Status() Status
	URLs() []string
	Done() <-chan struct{}
}

type HandlerFactory func(conf *coreconfig.Config) (http.Handler, error)

type Options struct {
	ListenAddress     string
	FrontendHandler   http.Handler
	StatusHandler     http.Handler
	VersionHandler    http.Handler
	Version           string
	Logger            *slog.Logger
	LANIPv4           func() []string
	HandlerFactory    HandlerFactory
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

// Runtime 管理一个可由 Headless 或 Desktop 直接控制的 HTTP 服务实例。
type Runtime struct {
	mu       sync.RWMutex
	options  Options
	logger   *slog.Logger
	server   *http.Server
	listener net.Listener
	done     chan struct{}
	stopDone chan struct{}
	status   Status
}

func New(options Options) *Runtime {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.LANIPv4 == nil {
		options.LANIPv4 = ListLANIPv4
	}
	if options.ShutdownTimeout <= 0 {
		options.ShutdownTimeout = 10 * time.Second
	}
	if options.ReadHeaderTimeout <= 0 {
		options.ReadHeaderTimeout = 10 * time.Second
	}
	if options.IdleTimeout <= 0 {
		options.IdleTimeout = 2 * time.Minute
	}
	if options.MaxHeaderBytes <= 0 {
		options.MaxHeaderBytes = 1 << 20
	}
	runtime := &Runtime{
		options: options,
		logger:  options.Logger,
		status:  Status{State: StateStopped},
	}
	if runtime.options.HandlerFactory == nil {
		runtime.options.HandlerFactory = func(conf *coreconfig.Config) (http.Handler, error) {
			statusHandler := runtime.options.StatusHandler
			if statusHandler == nil {
				sourceName := ""
				if len(conf.Sources) == 1 {
					sourceName = conf.Sources[0].Name
				}
				statusHandler = appserver.NewStatusHandler(func() appserver.StatusResponse {
					status := runtime.Status()
					return appserver.StatusResponse{
						State:    string(status.State),
						Name:     "YSeren",
						Source:   sourceName,
						RootName: sourceName,
						Port:     status.Port,
						URLs:     status.URLs,
					}
				})
			}
			application := appserver.New(conf, appserver.Options{
				FrontendHandler: runtime.options.FrontendHandler,
				StatusHandler:   statusHandler,
				VersionHandler:  runtime.options.VersionHandler,
				Version:         runtime.options.Version,
				Logger:          runtime.options.Logger,
			})
			return application.Handler(), nil
		}
	}
	return runtime
}

func (r *Runtime) Start(ctx context.Context, conf coreconfig.Config) error {
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	if r.server != nil || r.status.State == StateStarting || r.status.State == StateRunning || r.status.State == StateStopping {
		r.mu.Unlock()
		return ErrAlreadyRunning
	}
	r.status = Status{State: StateStarting}

	if err := conf.Validate(); err != nil {
		r.failStartLocked(err)
		r.mu.Unlock()
		return err
	}
	cloned := conf.Clone()
	handler, err := r.options.HandlerFactory(&cloned)
	if err != nil {
		r.failStartLocked(err)
		r.mu.Unlock()
		return err
	}
	if handler == nil {
		err = errors.New("Runtime 未能创建 HTTP Handler")
		r.failStartLocked(err)
		r.mu.Unlock()
		return err
	}

	listenAddress := net.JoinHostPort(strings.TrimSpace(r.options.ListenAddress), strconv.Itoa(cloned.Server.Port))
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", listenAddress)
	if err != nil {
		err = fmt.Errorf("无法监听 %s: %w", listenAddress, err)
		r.failStartLocked(err)
		r.mu.Unlock()
		return err
	}
	port, err := listenerPort(listener.Addr())
	if err != nil {
		_ = listener.Close()
		r.failStartLocked(err)
		r.mu.Unlock()
		return err
	}

	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: r.options.ReadHeaderTimeout,
		IdleTimeout:       r.options.IdleTimeout,
		MaxHeaderBytes:    r.options.MaxHeaderBytes,
		// WriteTimeout intentionally remains unset: large media responses can be long-lived.
	}
	done := make(chan struct{})
	urls := buildURLs(r.options.ListenAddress, port, r.options.LANIPv4())
	r.server = httpServer
	r.listener = listener
	r.done = done
	r.status = Status{
		State:     StateRunning,
		Address:   listener.Addr().String(),
		Port:      port,
		URLs:      append([]string(nil), urls...),
		StartedAt: time.Now(),
	}
	r.mu.Unlock()

	go r.serve(httpServer, listener, done)
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	if r.server == nil {
		r.status = Status{State: StateStopped}
		r.mu.Unlock()
		return nil
	}
	if r.status.State == StateStopping {
		stopDone := r.stopDone
		r.mu.Unlock()
		return waitForDone(ctx, stopDone)
	}

	httpServer := r.server
	done := r.done
	stopDone := make(chan struct{})
	r.stopDone = stopDone
	r.status.State = StateStopping
	r.mu.Unlock()

	shutdownContext, cancel := r.shutdownContext(ctx)
	defer cancel()
	shutdownErr := httpServer.Shutdown(shutdownContext)
	if shutdownErr != nil {
		_ = httpServer.Close()
	}
	waitErr := waitForDone(context.Background(), done)

	r.mu.Lock()
	if r.server == httpServer {
		r.server = nil
		r.listener = nil
		r.done = nil
		r.status = Status{State: StateStopped}
		if shutdownErr != nil {
			r.status.LastError = shutdownErr.Error()
		}
	}
	if r.stopDone == stopDone {
		r.stopDone = nil
		close(stopDone)
	}
	r.mu.Unlock()
	return errors.Join(shutdownErr, waitErr)
}

func (r *Runtime) Restart(ctx context.Context, conf coreconfig.Config) error {
	if err := r.Stop(ctx); err != nil {
		return err
	}
	return r.Start(ctx, conf)
}

func (r *Runtime) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status := r.status
	status.URLs = append([]string(nil), r.status.URLs...)
	return status
}

func (r *Runtime) URLs() []string {
	return r.Status().URLs
}

// Done 在当前 HTTP 服务停止时关闭；调用方应在 Start 成功后获取它。
func (r *Runtime) Done() <-chan struct{} {
	r.mu.RLock()
	done := r.done
	r.mu.RUnlock()
	if done != nil {
		return done
	}
	closed := make(chan struct{})
	close(closed)
	return closed
}

func (r *Runtime) serve(httpServer *http.Server, listener net.Listener, done chan struct{}) {
	err := httpServer.Serve(listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		close(done)
		return
	}

	r.logger.Error("YSeren HTTP 服务异常退出", "error", err)
	r.mu.Lock()
	if r.server == httpServer {
		r.server = nil
		r.listener = nil
		r.done = nil
		r.status.State = StateFailed
		r.status.LastError = err.Error()
	}
	r.mu.Unlock()
	close(done)
}

func (r *Runtime) failStartLocked(err error) {
	r.server = nil
	r.listener = nil
	r.done = nil
	r.stopDone = nil
	r.status = Status{State: StateFailed, LastError: err.Error()}
}

func (r *Runtime) shutdownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, r.options.ShutdownTimeout)
}

func listenerPort(address net.Addr) (int, error) {
	tcpAddress, ok := address.(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", address)
	}
	return tcpAddress.Port, nil
}

func waitForDone(ctx context.Context, done <-chan struct{}) error {
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func buildURLs(listenAddress string, port int, lanAddresses []string) []string {
	host := strings.Trim(strings.TrimSpace(listenAddress), "[]")
	parsedIP := net.ParseIP(host)
	unspecified := host == "" || parsedIP != nil && parsedIP.IsUnspecified()
	loopback := strings.EqualFold(host, "localhost") || parsedIP != nil && parsedIP.IsLoopback()

	urls := make([]string, 0, len(lanAddresses)+1)
	seen := make(map[string]struct{})
	add := func(address string) {
		url := "http://" + net.JoinHostPort(address, strconv.Itoa(port)) + "/"
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		urls = append(urls, url)
	}

	if unspecified || loopback {
		add("localhost")
	} else {
		add(host)
	}
	if unspecified {
		for _, address := range lanAddresses {
			if net.ParseIP(address) != nil {
				add(address)
			}
		}
	}
	return urls
}
