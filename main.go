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

//
//// 保证金监控器
//type MarginMonitor struct {
//	exchange    *ccxt.Exchange
//	config      Config
//	posTrackers map[string]*PositionTracker // 跟踪各个仓位的状态
//	logger      *log.Logger
//}
//
//// 创建新的保证金监控器
//func NewMarginMonitor(config Config) (*MarginMonitor, error) {
//	// 初始化日志
//	logger := log.New(os.Stdout, "[保证金监控] ", log.LstdFlags)
//
//	// 初始化CCXT交易所
//	exchange, err := ccxt.NewExchange(config.Exchange)
//	if err != nil {
//		return nil, fmt.Errorf("初始化交易所失败: %v", err)
//	}
//
//	// 设置API凭证
//	exchange.ApiKey = config.ApiKey
//	exchange.Secret = config.ApiSecret
//
//	// 设置额外参数(如密码或其他认证信息)
//	if config.AdditionalParams != nil {
//		for key, value := range config.AdditionalParams {
//			exchange.SetParam(key, value)
//		}
//	}
//
//	// 加载市场
//	logger.Println("正在加载市场数据...")
//	if err := exchange.LoadMarkets(true); err != nil {
//		return nil, fmt.Errorf("加载市场失败: %v", err)
//	}
//	logger.Println("市场数据加载完成")
//
//	return &MarginMonitor{
//		exchange:    exchange,
//		config:      config,
//		posTrackers: make(map[string]*PositionTracker),
//		logger:      logger,
//	}, nil
//}
//
//// 启动监控
//func (m *MarginMonitor) Start(ctx context.Context) error {
//	ticker := time.NewTicker(m.config.CheckInterval)
//	defer ticker.Stop()
//
//	m.logger.Printf("开始全账户保证金监控，检查间隔: %v\n", m.config.CheckInterval)
//
//	for {
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		case <-ticker.C:
//			if err := m.checkAllPositions(); err != nil {
//				m.logger.Printf("检查账户仓位出错: %v\n", err)
//			}
//		}
//	}
//}
//
//// 检查所有仓位
//func (m *MarginMonitor) checkAllPositions() error {
//	m.logger.Println("正在检查所有合约仓位...")
//
//	// 获取所有逐仓仓位信息
//	positions, err := m.exchange.FetchPositions(nil, nil)
//	if err != nil {
//		return fmt.Errorf("获取仓位信息失败: %v", err)
//	}
//
//	m.logger.Printf("找到 %d 个仓位\n", len(positions))
//
//	if len(positions) == 0 {
//		return nil
//	}
//
//	// 遍历所有仓位
//	for _, pos := range positions {
//		posMap, ok := pos.(map[string]interface{})
//		if !ok {
//			m.logger.Println("跳过无效仓位数据")
//			continue
//		}
//
//		// 检查是否为逐仓模式
//		marginMode, _ := posMap["marginMode"].(string)
//		if marginMode != "isolated" {
//			continue // 只处理逐仓模式的仓位
//		}
//
//		// 获取仓位信息
//		symbol, _ := posMap["symbol"].(string)
//		size, _ := posMap["contracts"].(float64)
//
//		// 跳过空仓位
//		if size == 0 {
//			continue
//		}
//
//		// 检查单个仓位并在需要时添加保证金
//		if err := m.checkAndAddMargin(symbol, posMap); err != nil {
//			m.logger.Printf("处理仓位 %s 时出错: %v\n", symbol, err)
//		}
//	}
//
//	return nil
//}
//
//// 检查并添加保证金
//func (m *MarginMonitor) checkAndAddMargin(symbol string, position map[string]interface{}) error {
//	// 从仓位信息中提取数据
//	entryPrice, _ := position["entryPrice"].(float64)
//	markPrice, _ := position["markPrice"].(float64)
//	liquidationPrice, _ := position["liquidationPrice"].(float64)
//	margin, _ := position["collateral"].(float64)
//	leverage, _ := position["leverage"].(float64)
//	side, _ := position["side"].(string)
//
//	// 跳过无效数据
//	if liquidationPrice == 0 || markPrice == 0 {
//		return fmt.Errorf("仓位 %s 数据不完整", symbol)
//	}
//
//	// 计算与强平价格的距离百分比
//	var distancePercentage float64
//
//	if side == "long" {
//		// 多仓：当前价格 > 强平价格
//		distancePercentage = ((markPrice - liquidationPrice) / markPrice) * 100
//	} else {
//		// 空仓：当前价格 < 强平价格
//		distancePercentage = ((liquidationPrice - markPrice) / markPrice) * 100
//	}
//
//	// 获取仓位跟踪器
//	tracker, exists := m.posTrackers[symbol]
//	if !exists {
//		tracker = &PositionTracker{
//			AddCount:    0,
//			LastAddTime: time.Time{},
//		}
//		m.posTrackers[symbol] = tracker
//	}
//
//	m.logger.Printf("仓位: %s, 方向: %s, 杠杆: %.1fx\n", symbol, side, leverage)
//	m.logger.Printf("价格信息: 开仓价格=%.2f, 标记价格=%.2f, 强平价格=%.2f\n",
//		entryPrice, markPrice, liquidationPrice)
//	m.logger.Printf("保证金状态: 当前保证金=%.2f, 距离强平=%.2f%%, 已添加次数=%d/%d\n",
//		margin, distancePercentage, tracker.AddCount, m.config.MaxAutoAddCount)
//
//	// 如果距离强平价格小于阈值，且未超过最大添加次数，则添加保证金
//	if distancePercentage < m.config.DangerThreshold && tracker.AddCount < m.config.MaxAutoAddCount {
//		// 计算需要添加的保证金金额
//		var addAmount float64
//		if m.config.UseFixedAmount {
//			addAmount = m.config.MarginAddAmount
//		} else {
//			// 按照现有保证金的百分比添加
//			addAmount = margin * (m.config.MarginAddPercentage / 100)
//		}
//
//		// 获取账户余额
//		balance, err := m.exchange.FetchBalance(nil)
//		if err != nil {
//			return fmt.Errorf("获取余额失败: %v", err)
//		}
//
//		// 提取计价货币的可用余额
//		balanceMap, ok := balance.(map[string]interface{})
//		if !ok {
//			return fmt.Errorf("余额格式错误")
//		}
//
//		// 从交易对中提取计价货币
//		currencies := split(symbol, "/")
//		if len(currencies) < 2 {
//			currencies = split(symbol, ":") // 尝试其他分隔符
//		}
//
//		if len(currencies) < 2 {
//			return fmt.Errorf("无法从交易对 %s 解析计价货币", symbol)
//		}
//
//		// 尝试不同格式获取计价货币
//		var quoteCurrency string
//		if len(currencies) == 2 {
//			quoteCurrency = currencies[1]
//		} else {
//			// 处理形如 BTC/USDT:USDT 的情况
//			quoteCurrency = currencies[len(currencies)-1]
//		}
//
//		// 清理货币代码中的特殊字符
//		quoteCurrency = cleanCurrencyCode(quoteCurrency)
//
//		currencyBalance, ok := balanceMap[quoteCurrency].(map[string]interface{})
//		if !ok {
//			return fmt.Errorf("获取 %s 余额失败", quoteCurrency)
//		}
//
//		availableBalance, _ := currencyBalance["free"].(float64)
//
//		if availableBalance < addAmount {
//			return fmt.Errorf("可用余额不足: %.2f %s, 需要: %.2f %s",
//				availableBalance, quoteCurrency, addAmount, quoteCurrency)
//		}
//
//		// 添加保证金
//		m.logger.Printf("准备为仓位 %s 添加 %.2f %s 保证金...\n",
//			symbol, addAmount, quoteCurrency)
//
//		// 构建参数(不同交易所可能需要不同参数)
//		params := map[string]interface{}{
//			"symbol": symbol,
//			"amount": addAmount,
//		}
//
//		// 尝试添加仓位ID(如果存在)
//		if posId, ok := position["id"]; ok {
//			params["positionId"] = posId
//		}
//
//		// 添加保证金类型(如果需要)
//		params["type"] = "add"
//
//		result, err := m.exchange.AddMargin(symbol, addAmount, params)
//		if err != nil {
//			return fmt.Errorf("添加保证金失败: %v", err)
//		}
//
//		tracker.AddCount++
//		tracker.LastAddTime = time.Now()
//
//		m.logger.Printf("成功为仓位 %s 添加保证金: %.2f %s, 累计添加次数: %d/%d\n",
//			symbol, addAmount, quoteCurrency, tracker.AddCount, m.config.MaxAutoAddCount)
//		m.logger.Printf("添加保证金结果: %v\n", result)
//	}
//
//	return nil
//}
//
//// 辅助函数：分割字符串
//func split(s, sep string) []string {
//	var result []string
//	for s != "" {
//		i := 0
//		for i < len(s) && i+len(sep) <= len(s) && s[i:i+len(sep)] != sep {
//			i++
//		}
//		result = append(result, s[:i])
//		if i+len(sep) <= len(s) {
//			s = s[i+len(sep):]
//		} else {
//			break
//		}
//	}
//	return result
//}
//
//// 清理货币代码
//func cleanCurrencyCode(code string) string {
//	// 删除可能的特殊字符
//	result := ""
//	for _, c := range code {
//		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
//			result += string(c)
//		}
//	}
//	return result
//}

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
