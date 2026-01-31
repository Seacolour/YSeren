package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// 1. 加载配置：支持 -config 指定；默认按“当前目录 -> exe 同目录”查找 v-link.yaml/yml
	configPath := flag.String("config", "", "配置文件路径（默认查找 v-link.yaml 或 v-link.yml：当前目录 -> exe 同目录）")
	flag.Parse()

	conf, usedPath, err := LoadConfigAuto(*configPath)
	if err != nil {
		log.Fatalf("无法加载配置文件: %v", err)
	}
	fmt.Printf("已加载配置: %s\n\n", usedPath)

	// 2. 循环挂载所有资源点
	for _, source := range conf.Sources {
		// 注意：URL 路径建议加个前缀，比如 /stream/我的动漫
		route := fmt.Sprintf("/stream/%s/", source.Name)

		// 创建文件服务器
		fs := http.FileServer(http.Dir(source.Path))

		// 关键点：使用 StripPrefix 确保文件路径查找正确
		http.Handle(route, http.StripPrefix(route, fs))

		fmt.Printf("已挂载资源 [%s]: http://localhost:%d%s\n", source.Name, conf.Server.Port, route)
	}

	// 2. API：给前端提供视频文件列表（递归扫描 + 搜索/分页）
	http.HandleFunc("/api/videos", ListVideosHandler(conf))
	// 2.1 API：目录树（保留层级，便于前端做“文件夹浏览”）
	http.HandleFunc("/api/tree", ListTreeHandler(conf))
	// 2.2 API：zip 解压（MVP：仅支持 .zip，且不自动展开，用户手动点击“解压”）
	http.HandleFunc("/api/zip/extract", ZipExtractHandler(conf))

	// 3. 前端：优先用 embed 的静态资源提供手机 UI；如果没打包 dist，就回退到传统目录（方便本地开发）
	http.Handle("/", FrontendHandler())

	// 3. 启动服务
	addr := fmt.Sprintf(":%d", conf.Server.Port)

	fmt.Printf("\nlv-link 视频引擎已启动，监听端口 %s\n", addr)
	fmt.Printf("本机访问: http://localhost:%d/\n", conf.Server.Port)

	lanIPs := ListLANIPv4()
	if len(lanIPs) > 0 {
		fmt.Printf("局域网访问（复制给同一 Wi-Fi/内网用户）:\n")
		for _, ip := range lanIPs {
			fmt.Printf("- http://%s:%d/\n", ip, conf.Server.Port)
		}
	} else {
		fmt.Printf("局域网访问: 未检测到可用的内网 IPv4（请确认已连接 Wi-Fi/网线）\n")
	}

	// 额外：为每个 source 打印可直达的 stream 目录链接
	fmt.Printf("\n资源直达:\n")
	for _, source := range conf.Sources {
		route := fmt.Sprintf("/stream/%s/", source.Name)
		fmt.Printf("- [%s] 本机: http://localhost:%d%s\n", source.Name, conf.Server.Port, route)
		for _, ip := range lanIPs {
			fmt.Printf("  [%s] 局域网: http://%s:%d%s\n", source.Name, ip, conf.Server.Port, route)
		}
	}

	log.Fatal(http.ListenAndServe(addr, nil))
}
