package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			WriteCrashLog(fmt.Sprintf("panic: %v", r), nil)
			panic(r)
		}
	}()

	// 1. 加载配置：支持 -config 指定；默认按"当前目录 -> exe 同目录"查找 v-link.yaml/yml
	configPath := flag.String("config", "", "配置文件路径（默认查找 v-link.yaml 或 v-link.yml：当前目录 -> exe 同目录）")
	flag.Parse()

	conf, usedPath, err := LoadConfigAuto(*configPath)
	if err != nil {
		errMsg := fmt.Sprintf("无法加载配置文件: %v", err)
		fmt.Fprintln(os.Stderr, errMsg)
		WriteCrashLog(errMsg, err)
		startErrorServer(errMsg)
		return
	}

	// 初始化日志系统
	InitLogger(conf.Server.LogLevel)
	LogInfo("配置加载完成", "path", usedPath)

	// 2. 循环挂载所有资源点
	for _, source := range conf.Sources {
		// 注意：URL 路径建议加个前缀，比如 /stream/我的动漫
		route := fmt.Sprintf("/stream/%s/", source.Name)

		// 创建文件服务器
		fs := http.FileServer(http.Dir(source.Path))

		// 关键点：使用 StripPrefix 确保文件路径查找正确
		http.Handle(route, http.StripPrefix(route, fs))

		LogInfo("挂载资源", "name", source.Name, "path", source.Path, "route", route)
	}

	// 2. API：给前端提供视频文件列表（递归扫描 + 搜索/分页）
	http.HandleFunc("/api/videos", ListVideosHandler(conf))
	// 2.1 API：目录树（保留层级，便于前端做"文件夹浏览"）
	http.HandleFunc("/api/tree", ListTreeHandler(conf))
	// 2.2 API：zip 解压（MVP：仅支持 .zip，且不自动展开，用户手动点击"解压"）
	http.HandleFunc("/api/zip/extract", ZipExtractHandler(conf))

	// 3. 前端：优先用 embed 的静态资源提供手机 UI；如果没打包 dist，就回退到传统目录（方便本地开发）
	http.Handle("/", FrontendHandler())

	// 3. 启动服务
	addr := fmt.Sprintf(":%d", conf.Server.Port)

	// 启动信息（保持用户友好的输出）
	fmt.Println()
	fmt.Printf("  ✦ YSeren - 局域网媒体\n")
	fmt.Printf("  ─────────────────────\n")
	fmt.Printf("本机访问: http://localhost:%d/\n", conf.Server.Port)

	lanIPs := ListLANIPv4()
	if len(lanIPs) > 0 {
		fmt.Printf("局域网访问:\n")
		for _, ip := range lanIPs {
			fmt.Printf("  → http://%s:%d/\n", ip, conf.Server.Port)
		}
	} else {
		fmt.Printf("局域网: 未检测到可用的内网 IPv4\n")
	}
	fmt.Println()

	LogInfo("服务启动", "addr", addr, "port", conf.Server.Port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		LogError("服务启动失败", "error", err)
		WriteCrashLog("服务启动失败", err)
		os.Exit(1)
	}
}

func startErrorServer(message string) {
	const fallbackPort = 1479
	addr := fmt.Sprintf(":%d", fallbackPort)

	mux := http.NewServeMux()
	mux.Handle("/", ErrorFrontendHandler(message))

	fmt.Println()
	fmt.Printf("  ✦ YSeren - 启动失败\n")
	fmt.Printf("  ─────────────────────\n")
	fmt.Printf("请打开浏览器查看错误信息: http://localhost:%d/\n", fallbackPort)

	if err := http.ListenAndServe(addr, mux); err != nil {
		WriteCrashLog("启动错误页失败", err)
		fmt.Fprintf(os.Stderr, "启动错误页失败: %v\n", err)
		os.Exit(1)
	}
}
