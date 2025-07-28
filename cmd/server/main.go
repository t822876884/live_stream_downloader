package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/luan78zao/live_stream_downloader/internal/config"
	"github.com/luan78zao/live_stream_downloader/internal/downloader"
	"github.com/luan78zao/live_stream_downloader/internal/handler"
)

func main() {
	// 解析命令行参数
	cfg := config.NewDefaultConfig()

	flag.StringVar(&cfg.ServerAddr, "addr", cfg.ServerAddr, "服务器地址")
	flag.StringVar(&cfg.DataDir, "data", cfg.DataDir, "数据目录")
	flag.Parse()

	// 确保数据目录是绝对路径
	absDataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		log.Fatalf("获取数据目录的绝对路径失败: %v", err)
	}
	cfg.DataDir = absDataDir

	// 确保数据目录存在
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	// 创建下载器
	downloader, err := downloader.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("创建下载器失败: %v", err)
	}

	// 创建HTTP处理器
	handler, err := handler.New(downloader, "web/templates", cfg.DataDir)
	if err != nil {
		log.Fatalf("创建HTTP处理器失败: %v", err)
	}

	// 创建HTTP服务器
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 启动HTTP服务器
	log.Printf("服务器启动在 %s", cfg.ServerAddr)
	log.Printf("数据目录: %s", cfg.DataDir)
	log.Fatal(http.ListenAndServe(cfg.ServerAddr, mux))
}
