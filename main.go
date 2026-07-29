package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	webfrontend "yseren/frontend"
	coreconfig "yseren/internal/config"
	appruntime "yseren/internal/runtime"
	appserver "yseren/internal/server"
	coreversion "yseren/internal/version"
)

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			WriteCrashLog(fmt.Sprintf("panic: %v", recovered), nil)
			panic(recovered)
		}
	}()

	configPath := flag.String("config", "", "配置文件路径（默认查找 yseren.yaml 或 yseren.yml：当前目录 -> exe 同目录）")
	flag.Parse()

	conf, usedPath, err := coreconfig.LoadConfigAuto(*configPath)
	if err != nil {
		errorMessage := fmt.Sprintf("无法加载配置文件: %v", err)
		fmt.Fprintln(os.Stderr, errorMessage)
		WriteCrashLog(errorMessage, err)
		startErrorServer(errorMessage)
		return
	}

	InitLogger(conf.Server.LogLevel)
	LogInfo("配置加载完成", "path", usedPath)
	for _, source := range conf.Sources {
		LogInfo("挂载资源", "name", source.Name, "path", source.Path, "route", appserver.StreamRoutePattern(source.Name))
	}

	application := appruntime.New(appruntime.Options{
		FrontendHandler: webfrontend.Handler(),
		Version:         Version,
		Logger:          Logger,
	})
	if err := application.Start(context.Background(), *conf); err != nil {
		LogError("服务启动失败", "error", err)
		WriteCrashLog("服务启动失败", err)
		os.Exit(1)
	}

	printStartup(application.Status())
	LogInfo("服务启动", "addr", application.Status().Address, "port", application.Status().Port)

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	select {
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := application.Stop(shutdownContext); err != nil {
			LogError("服务停止失败", "error", err)
			WriteCrashLog("服务停止失败", err)
			os.Exit(1)
		}
		LogInfo("服务已停止")
	case <-application.Done():
		status := application.Status()
		if status.State == appruntime.StateFailed {
			err := fmt.Errorf("HTTP 服务异常退出: %s", status.LastError)
			LogError("服务异常退出", "error", err)
			WriteCrashLog("服务异常退出", err)
			os.Exit(1)
		}
	}
}

func printStartup(status appruntime.Status) {
	fmt.Println()
	fmt.Printf("  ✦ YSeren - 局域网媒体")
	if version := coreversion.Normalize(Version); version != "" && version != "dev" {
		fmt.Printf("  v%s", version)
	}
	fmt.Println()
	fmt.Printf("  ─────────────────────\n")
	if len(status.URLs) > 0 {
		fmt.Printf("本机访问: %s\n", status.URLs[0])
	}
	if len(status.URLs) > 1 {
		fmt.Printf("局域网访问:\n")
		for _, url := range status.URLs[1:] {
			fmt.Printf("  → %s\n", url)
		}
	} else {
		fmt.Printf("局域网: 未检测到可用的内网 IPv4\n")
	}
	fmt.Println()
}

func startErrorServer(message string) {
	const fallbackPort = 1479
	address := fmt.Sprintf(":%d", fallbackPort)
	mux := http.NewServeMux()
	mux.Handle("/", webfrontend.ErrorHandler(message))

	fmt.Println()
	fmt.Printf("  ✦ YSeren - 启动失败\n")
	fmt.Printf("  ─────────────────────\n")
	fmt.Printf("请打开浏览器查看错误信息: http://localhost:%d/\n", fallbackPort)

	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       time.Minute,
	}
	if err := server.ListenAndServe(); err != nil {
		WriteCrashLog("启动错误页失败", err)
		fmt.Fprintf(os.Stderr, "启动错误页失败: %v\n", err)
		os.Exit(1)
	}
}
