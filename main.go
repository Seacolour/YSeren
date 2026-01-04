package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// 1. 加载配置（假设当前目录下有 v-link.yaml）
	conf, err := LoadConfig("v-link.yaml")
	if err != nil {
		log.Fatalf("无法加载配置文件: %v", err)
	}

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

	// 3. 前端：优先用 embed 的静态资源提供手机 UI；如果没打包 dist，就回退到传统目录（方便本地开发）
	http.Handle("/", FrontendHandler())

	// 3. 启动服务
	addr := fmt.Sprintf(":%d", conf.Server.Port)
	fmt.Printf("\nlv-link 视频引擎已启动，监听端口 %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
