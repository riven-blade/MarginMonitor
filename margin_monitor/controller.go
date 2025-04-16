package margin_monitor

import (
	"context"
	"fmt"
	ccxt "github.com/ccxt/ccxt/go/v4"
	"log"
	"margin_monitor/config"
	"margin_monitor/exchange"
	"margin_monitor/model"
	"math"
	"time"
)

func NewController(conf *config.Config) (*Controller, error) {
	m, err := NewMonitor(conf)
	if err != nil {
		log.Fatalf("init monitor err: %v", err)
	}

	return &Controller{
		Conf: conf,
		M:    m,
		Pair: NewPair(conf),
	}, nil
}

type Controller struct {
	Conf *config.Config
	M    *Monitor
	Pair *Pair
}

func (c *Controller) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pairTicker := time.NewTicker(time.Duration(c.Conf.RefreshPairs.Interval) * time.Second)
	checkTicker := time.NewTicker(time.Duration(c.Conf.Monitor.CheckInterval) * time.Second)
	defer pairTicker.Stop()
	defer checkTicker.Stop()

	go c.checkPairs()

	for {
		select {
		case <-ctx.Done():
			log.Println("Controller stopped")
			return nil

		case <-pairTicker.C:
			go c.checkPairs()

		case <-checkTicker.C:
			go c.checkExchanges()
		}
	}
}

// checkExchanges 遍历所有交易所并检查持仓
func (c *Controller) checkExchanges() {
	for i := range c.M.Exchange {
		ex := c.M.Exchange[i]
		go func() {
			positions, err := ex.FetchPositions()
			if err != nil {
				log.Printf("fetch positions error: %v\n", err)
				c.M.SendTelegramMessage(fmt.Sprintf("%s: fetch positions error", ex.GetName()))
				return
			}
			c.handlePositions(ex, positions)
		}()
	}
}

// handlePositions 检查每个持仓是否超出风险阈值
func (c *Controller) handlePositions(exchange exchange.Exchange, positions interface{}) {
	switch exchange.GetName() {
	case "Binance":
		if binancePositions, ok := positions.([]ccxt.Position); ok {
			for i := range binancePositions {
				ps := binancePositions[i]
				log.Printf("Checking position: Symbol=%s, MarginRatio=%.4f, InitialMargin=%.4f\n",
					*ps.Symbol, *ps.MarginRatio, *ps.InitialMargin)

				if *ps.MarginRatio > c.Conf.Monitor.DangerThreshold {
					addAmount := math.Ceil(*ps.InitialMargin * c.Conf.AddMultiple)
					log.Printf("⚠️ Margin ratio exceeds threshold! Adding margin: Symbol=%s, Amount=%.4f\n",
						*ps.Symbol, addAmount)

					go func(symbol string, amount float64) {
						msg := exchange.AddMargin(symbol, amount)
						c.M.SendTelegramMessage(msg)
					}(*ps.Symbol, addAmount)
				}
			}
		}
	case "ByBit":
		if bybitPositions, ok := positions.(*model.PositionList); ok {
			for i := range bybitPositions.List {
				ps := bybitPositions.List[i]
				if ps.AutoAddMargin == 1 {
					log.Printf(fmt.Sprintf("📍 ByBit %s: %s", ps.Symbol, "已经配置自动追加保证金"))
					continue
				}

				go func(symbol string, amount float64) {
					// 调用 AddMargin，设置自动追加保证金为 1
					msg := exchange.AddMargin(symbol, 1)
					c.M.SendTelegramMessage(fmt.Sprintf("📍 ByBit %s: %s", symbol, msg))
				}(ps.Symbol, 0)
			}
		}
	}
}
