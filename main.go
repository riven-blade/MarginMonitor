package main

import (
	"context"
	"errors"
	"log"
	"margin_monitor/config"
	"margin_monitor/margin_monitor"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	filename := "./config.yaml"
	conf, err := config.LoadConfig(filename)
	if err != nil {
		log.Fatal(err)
	}

	// 创建保证金监控器
	controller, err := margin_monitor.NewController(conf)
	if err != nil {
		log.Fatalf("初始化监控器失败: %v", err)
	}

	// 创建上下文，捕获终止信号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 捕获中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("收到终止信号，正在停止...")
		cancel()
	}()

	// 启动监控
	if err := controller.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("监控器错误: %v", err)
	}
}
